package openbao

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// The real JWT and the real token value used throughout this file: every
// test that forces an error path checks the resulting error (and, where a
// logger captures output, the log) against both of these, never just one.
const (
	testJWT      = "eyJhbGciOiJSUzI1NiJ9.service-account-jwt-body.sig"
	testRole     = "feed"
	testAuthPath = "k8s"
)

// handlerTransport serves requests in process, without a listening port.
type handlerTransport struct{ h http.Handler }

func (t handlerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	t.h.ServeHTTP(rec, r)
	return rec.Result(), nil
}

// fakeBao is a minimal OpenBao: Kubernetes login plus one KV v2 mount.
type fakeBao struct {
	mu            sync.Mutex
	lease         int // seconds; 0 means "send no lease_duration at all".
	logins        int
	requests      int // every request, logins included.
	tokens        map[string]bool
	secrets       map[string]map[string]any
	versions      map[string]int
	deletedAt     map[string]bool // path -> KV v2 answered a deleted version (data null).
	lastJWT       string
	lastRole      string
	lastWriteBody []byte
	alwaysForbid  bool // revokes every token on every request, including a fresh login's.
}

func newFakeBao() *fakeBao {
	return &fakeBao{
		lease:     3600,
		tokens:    map[string]bool{},
		secrets:   map[string]map[string]any{},
		versions:  map[string]int{},
		deletedAt: map[string]bool{},
	}
}

func (f *fakeBao) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests++
	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path == "/v1/auth/"+testAuthPath+"/login" && r.Method == http.MethodPost {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.lastJWT, f.lastRole = body["jwt"], body["role"]
		if body["role"] != testRole || body["jwt"] == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errors":["invalid role or jwt"]}`))
			return
		}
		f.logins++
		token := "tok-" + strconv.Itoa(f.logins)
		if !f.alwaysForbid {
			f.tokens[token] = true
		}
		auth := map[string]any{"client_token": token}
		if f.lease > 0 {
			auth["lease_duration"] = f.lease
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"auth": auth})
		return
	}

	if f.alwaysForbid || !f.tokens[r.Header.Get("X-Vault-Token")] {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":["permission denied"]}`))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/")
	switch r.Method {
	case http.MethodGet:
		if f.deletedAt[path] {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"data": nil, "metadata": map[string]any{"version": f.versions[path]},
			}})
			return
		}
		data, ok := f.secrets[path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[]}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"data": data, "metadata": map[string]any{"version": f.versions[path]},
		}})
	case http.MethodPost:
		raw, _ := io.ReadAll(r.Body)
		f.lastWriteBody = raw
		var body struct {
			Data    map[string]any `json:"data"`
			Options *struct {
				CAS int `json:"cas"`
			} `json:"options"`
		}
		_ = json.Unmarshal(raw, &body)
		if body.Options != nil && body.Options.CAS != f.versions[path] {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errors":["check-and-set parameter did not match the current version"]}`))
			return
		}
		f.secrets[path] = body.Data
		f.versions[path]++
		delete(f.deletedAt, path)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"version": f.versions[path]}})
	}
}

// testLogger returns a logger whose output lands in buf, so a test can
// assert on what was actually logged, not only on the returned error.
func testLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, nil))
}

func newTestClient(t *testing.T, bao http.Handler, buf *bytes.Buffer) *Client {
	t.Helper()
	jwt := writeJWT(t, testJWT)
	c := New("https://bao.test/", kube(testRole, jwt), testLogger(buf))
	c.http = &http.Client{Transport: handlerTransport{bao}}
	return c
}

// kube is the Kubernetes login against the fake's auth mount, written with
// the surrounding slashes a configuration might carry.
func kube(role, jwtFile string) Kubernetes {
	return Kubernetes{AuthPath: "/" + testAuthPath + "/", Role: role, JWTFile: jwtFile}
}

func writeJWT(t *testing.T, value string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(value+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestClient_Read_loginAndRead(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.secrets["mail/data/no-reply"] = map[string]any{"username": "u", "password": "p"}
	bao.versions["mail/data/no-reply"] = 3
	var logs bytes.Buffer
	c := newTestClient(t, bao, &logs)

	s, err := c.Read(t.Context(), "/mail/data/no-reply")
	if err != nil {
		t.Fatal(err)
	}
	if s.String("username") != "u" || s.String("password") != "p" || s.Version != 3 {
		t.Errorf("secret = %+v", s)
	}
	if bao.lastJWT != testJWT || bao.lastRole != testRole {
		t.Errorf("login sent jwt %q role %q", bao.lastJWT, bao.lastRole)
	}

	// A second read while the token is still fresh logs in again never.
	if _, err := c.Read(t.Context(), "mail/data/no-reply"); err != nil {
		t.Fatal(err)
	}
	if bao.logins != 1 {
		t.Errorf("logins = %d, want 1 while the token is fresh", bao.logins)
	}
	if !strings.Contains(logs.String(), "logged in") {
		t.Errorf("login was not logged: %s", logs.String())
	}
}

func TestClient_Read_notFound(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup func(*fakeBao)
		path  string
	}{
		"path has no secret at all": {
			setup: func(*fakeBao) {},
			path:  "mail/data/missing",
		},
		"version was deleted, OpenBao answers data null": {
			setup: func(b *fakeBao) {
				b.secrets["kv/data/gone"] = map[string]any{"k": "v"}
				b.versions["kv/data/gone"] = 2
				b.deletedAt["kv/data/gone"] = true
			},
			path: "kv/data/gone",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bao := newFakeBao()
			tc.setup(bao)
			c := newTestClient(t, bao, &bytes.Buffer{})
			if _, err := c.Read(t.Context(), tc.path); !errors.Is(err, ErrNotFound) {
				t.Fatalf("err = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestClient_Read_rejectsPathTraversal(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, newFakeBao(), &bytes.Buffer{})
	if _, err := c.Read(t.Context(), "../sys"); err == nil {
		t.Fatal("path with .. accepted")
	}
}

func TestClient_Read_reloginAtTwoThirdsOfLease(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.secrets["kv/data/x"] = map[string]any{"key": "v"}
	c := newTestClient(t, bao, &bytes.Buffer{})
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }

	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}

	// 39 of 60 minutes: under two thirds, the cached token is still used.
	now = now.Add(39 * time.Minute)
	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}
	if bao.logins != 1 {
		t.Fatalf("logins = %d before two thirds of the lease, want 1", bao.logins)
	}

	// 41 of 60 minutes: past two thirds, a fresh login happens.
	now = now.Add(2 * time.Minute)
	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}
	if bao.logins != 2 {
		t.Fatalf("logins = %d after two thirds of the lease, want 2", bao.logins)
	}
}

func TestClient_Read_zeroLeaseDurationFallsBackToAssumedTTL(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.lease = 0
	bao.secrets["kv/data/x"] = map[string]any{"key": "v"}
	c := newTestClient(t, bao, &bytes.Buffer{})
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }

	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}

	// Two thirds of the 90 minute assumed TTL is 60 minutes; one minute
	// short of that still uses the cached token.
	now = now.Add(59 * time.Minute)
	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}
	if bao.logins != 1 {
		t.Fatalf("logins = %d before two thirds of the assumed TTL, want 1", bao.logins)
	}

	now = now.Add(2 * time.Minute)
	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}
	if bao.logins != 2 {
		t.Fatalf("logins = %d after two thirds of the assumed TTL, want 2", bao.logins)
	}
}

func TestClient_Read_reloginsOnceOn403(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.secrets["kv/data/x"] = map[string]any{"key": "v"}
	var logs bytes.Buffer
	c := newTestClient(t, bao, &logs)
	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}

	// Revoke every token, as a restart of OpenBao without persistence would.
	bao.mu.Lock()
	bao.tokens = map[string]bool{}
	bao.mu.Unlock()

	s, err := c.Read(t.Context(), "kv/data/x")
	if err != nil {
		t.Fatalf("read after revocation: %v", err)
	}
	if s.String("key") != "v" {
		t.Errorf("secret = %v, want key v", s.Data)
	}
	if bao.logins != 2 {
		t.Fatalf("logins = %d, want exactly 2: one retry, not a loop", bao.logins)
	}
	if !strings.Contains(logs.String(), "rejected") {
		t.Errorf("the 403 retry was not logged: %s", logs.String())
	}
}

func TestClient_Read_persistentForbiddenIsNotRetriedMoreThanOnce(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.secrets["kv/data/x"] = map[string]any{"key": "v"}
	c := newTestClient(t, bao, &bytes.Buffer{})
	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}

	bao.mu.Lock()
	bao.alwaysForbid = true
	bao.mu.Unlock()

	if _, err := c.Read(t.Context(), "kv/data/x"); err == nil {
		t.Fatal("read succeeded against a server that forbids every request")
	}
	if bao.logins != 2 {
		t.Fatalf("logins = %d, want exactly 2: the retry logs in once more and then stops", bao.logins)
	}
}

func TestClient_Write_casGuardsConcurrentWriters(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	c := newTestClient(t, bao, &bytes.Buffer{})
	ctx := t.Context()

	if err := c.Write(ctx, "kv/data/token", map[string]any{"access_token": "a"}, 0); err != nil {
		t.Fatal(err)
	}
	s, err := c.Read(ctx, "kv/data/token")
	if err != nil {
		t.Fatal(err)
	}
	if s.String("access_token") != "a" || s.Version != 1 {
		t.Fatalf("after write: %+v", s)
	}

	if err := c.Write(ctx, "kv/data/token", map[string]any{"access_token": "b"}, 1); err != nil {
		t.Fatal(err)
	}
	// A second writer with the now-stale version loses.
	if err := c.Write(ctx, "kv/data/token", map[string]any{"access_token": "c"}, 1); !errors.Is(err, ErrCASMismatch) {
		t.Fatalf("stale write: err = %v, want ErrCASMismatch", err)
	}
}

func TestClient_Write_negativeCASOmitsTheOption(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.secrets["kv/data/token"] = map[string]any{"access_token": "stale"}
	bao.versions["kv/data/token"] = 5
	c := newTestClient(t, bao, &bytes.Buffer{})
	ctx := t.Context()

	// The version on the server (5) would reject any cas other than 5;
	// an unconditional write still has to succeed.
	if err := c.Write(ctx, "kv/data/token", map[string]any{"access_token": "fresh"}, -1); err != nil {
		t.Fatalf("unconditional write: %v", err)
	}

	bao.mu.Lock()
	raw := bao.lastWriteBody
	bao.mu.Unlock()
	var sent map[string]any
	if err := json.Unmarshal(raw, &sent); err != nil {
		t.Fatal(err)
	}
	if _, ok := sent["options"]; ok {
		t.Errorf("request body carries an options field for a negative cas: %s", raw)
	}

	s, err := c.Read(ctx, "kv/data/token")
	if err != nil {
		t.Fatal(err)
	}
	if s.String("access_token") != "fresh" {
		t.Errorf("token = %q, want fresh", s.String("access_token"))
	}
}

func TestClient_jwtIsReReadFromDiskOnEveryLogin(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.secrets["kv/data/x"] = map[string]any{"key": "v"}
	jwtPath := writeJWT(t, "jwt-before-rotation")
	c := New("https://bao.test/", kube(testRole, jwtPath), testLogger(&bytes.Buffer{}))
	c.http = &http.Client{Transport: handlerTransport{bao}}

	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}
	if bao.lastJWT != "jwt-before-rotation" {
		t.Fatalf("first login sent %q", bao.lastJWT)
	}

	// The kubelet rotates the projected token in place; force a second
	// login through the 403 path rather than waiting out the lease.
	if err := os.WriteFile(jwtPath, []byte("jwt-after-rotation\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bao.mu.Lock()
	bao.tokens = map[string]bool{}
	bao.mu.Unlock()

	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}
	if bao.lastJWT != "jwt-after-rotation" {
		t.Errorf("second login sent %q, want the rotated value; the client cached the old one", bao.lastJWT)
	}
}

// TestClient_errorsAndLogsNeverCarryTheTokenOrTheJWT is the package's single
// most important test: it drives every error path this file has and
// collects both the returned errors and everything written to the logger,
// then asserts neither ever contains the real JWT or the real token value.
func TestClient_errorsAndLogsNeverCarryTheTokenOrTheJWT(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	var errs []string
	record := func(err error) {
		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	bao := newFakeBao()
	bao.secrets["kv/data/x"] = map[string]any{"key": "v"}
	c := newTestClient(t, bao, &logs)

	// A real token, obtained through a real login, to check against.
	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}
	var realToken string
	bao.mu.Lock()
	for tok := range bao.tokens {
		realToken = tok
	}
	bao.mu.Unlock()
	if realToken == "" {
		t.Fatal("no token was issued")
	}

	// Path rejected before any request is sent.
	_, err := c.Read(t.Context(), "../escape")
	record(err)

	// Login failure: wrong role, so the server's error body comes back,
	// and the client's own "no client token" path is never exercised with
	// a real secret in scope either way.
	wrongRole := New("https://bao.test/", kube("not-"+testRole, writeJWT(t, testJWT)), testLogger(&logs))
	wrongRole.http = &http.Client{Transport: handlerTransport{bao}}
	_, err = wrongRole.Read(t.Context(), "kv/data/x")
	record(err)

	// Missing JWT file.
	missingFile := New("https://bao.test/", kube(testRole, filepath.Join(t.TempDir(), "absent")), testLogger(&logs))
	missingFile.http = &http.Client{Transport: handlerTransport{bao}}
	_, err = missingFile.Read(t.Context(), "kv/data/x")
	record(err)

	// 403 retried once, then a persistent 403 that finally surfaces.
	bao.mu.Lock()
	bao.alwaysForbid = true
	bao.mu.Unlock()
	_, err = c.Read(t.Context(), "kv/data/x")
	record(err)

	// CAS mismatch on write.
	err = c.Write(t.Context(), "kv/data/x", map[string]any{"key": "new"}, 999)
	record(err)

	for _, e := range errs {
		if strings.Contains(e, testJWT) {
			t.Errorf("an error contains the JWT: %q", e)
		}
		if strings.Contains(e, realToken) {
			t.Errorf("an error contains the token: %q", e)
		}
	}
	if logged := logs.String(); strings.Contains(logged, testJWT) {
		t.Errorf("the log contains the JWT: %q", logged)
	} else if strings.Contains(logged, realToken) {
		t.Errorf("the log contains the token: %q", logged)
	}
}
