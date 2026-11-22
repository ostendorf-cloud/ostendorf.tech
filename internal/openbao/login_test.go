package openbao

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/secrettest"
)

// fakeToken is the static token every test below hands the client. It is
// distinctive so any leak into an error, a log line or a formatted value is
// found by a plain substring search.
const fakeToken = "s.fake-static-token-must-never-leak-7f3a"

// bodyMarker is what the token fake answers a 403 with, so a test can prove
// OpenBao's response body never reaches the error either.
const bodyMarker = "openbao-body-marker-9c1e"

// tokenBao is a minimal OpenBao for the static token method: no login route
// at all, one KV v2 secret, and a switch that makes it refuse the token.
type tokenBao struct {
	mu        sync.Mutex
	requests  int
	logins    int
	lastToken string
	forbid    bool
}

func (f *tokenBao) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests++
	w.Header().Set("Content-Type", "application/json")
	if strings.HasPrefix(r.URL.Path, "/v1/auth/") {
		f.logins++
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	f.lastToken = r.Header.Get("X-Vault-Token")
	if f.forbid || f.lastToken != fakeToken {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprintf(w, `{"errors":["permission denied %s"]}`, bodyMarker)
		return
	}
	_, _ = w.Write([]byte(`{"data":{"data":{"k":"v"},"metadata":{"version":2}}}`))
}

func (f *tokenBao) counts() (requests, logins int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests, f.logins
}

func newTokenClient(bao http.Handler, token Token, logs *bytes.Buffer) *Client {
	c := New("https://bao.test", StaticToken{Token: token}, testLogger(logs))
	c.http = &http.Client{Transport: handlerTransport{bao}}
	return c
}

func TestStaticToken_sentAsXVaultTokenWithoutALogin(t *testing.T) {
	t.Parallel()

	bao := &tokenBao{}
	var logs bytes.Buffer
	c := newTokenClient(bao, fakeToken, &logs)

	s, err := c.Read(t.Context(), "kv/data/x")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if s.String("k") != "v" || s.Version != 2 {
		t.Errorf("secret = %+v, want k=v version 2", s)
	}
	if bao.lastToken != fakeToken {
		t.Error("the request did not carry the configured token as X-Vault-Token")
	}
	if _, logins := bao.counts(); logins != 0 {
		t.Errorf("logins = %d, want 0: the token method sends no login request", logins)
	}
	if strings.Contains(logs.String(), fakeToken) {
		t.Error("the log contains the token")
	}
}

func TestStaticToken_neverRenewed(t *testing.T) {
	t.Parallel()

	bao := &tokenBao{}
	c := newTokenClient(bao, fakeToken, &bytes.Buffer{})
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }

	for range 3 {
		if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
			t.Fatal(err)
		}
		// Far past any lease two thirds would ever cover.
		now = now.Add(1000 * time.Hour)
	}
	if requests, logins := bao.counts(); requests != 3 || logins != 0 {
		t.Errorf("requests = %d, logins = %d, want 3 reads and no login or renewal", requests, logins)
	}
}

func TestStaticToken_forbiddenIsAFixedErrorAndNotRetried(t *testing.T) {
	t.Parallel()

	bao := &tokenBao{forbid: true}
	var logs bytes.Buffer
	c := newTokenClient(bao, fakeToken, &logs)

	_, readErr := c.Read(t.Context(), "kv/data/x")
	if !errors.Is(readErr, ErrForbidden) {
		t.Fatalf("Read() error = %v, want ErrForbidden", readErr)
	}
	if requests, _ := bao.counts(); requests != 1 {
		t.Fatalf("requests = %d after one Read, want 1: a static token's 403 is never retried", requests)
	}

	writeErr := c.Write(t.Context(), "kv/data/x", map[string]any{"k": "w"}, 2)
	if !errors.Is(writeErr, ErrForbidden) {
		t.Fatalf("Write() error = %v, want ErrForbidden", writeErr)
	}
	if requests, _ := bao.counts(); requests != 2 {
		t.Fatalf("requests = %d after one Read and one Write, want 2", requests)
	}

	for _, err := range []error{readErr, writeErr} {
		secrettest.AssertHidden(t, err, fakeToken)
		secrettest.AssertHidden(t, err, bodyMarker)
	}
	if strings.Contains(logs.String(), fakeToken) {
		t.Error("the log contains the token")
	}
}

func TestStaticToken_emptyTokenIsAnError(t *testing.T) {
	t.Parallel()

	bao := &tokenBao{}
	c := newTokenClient(bao, "", &bytes.Buffer{})
	if _, err := c.Read(t.Context(), "kv/data/x"); !errors.Is(err, ErrNoToken) {
		t.Fatalf("Read() error = %v, want ErrNoToken", err)
	}
	if requests, _ := bao.counts(); requests != 0 {
		t.Errorf("requests = %d, want 0 without a token", requests)
	}
}

// TestToken_neverFormatsItsValue proves the token stays hidden in every
// value that holds it: the Token itself, an Auth, a StaticToken, a Client
// that has cached it, and a slog line that logs any of them by mistake.
func TestToken_neverFormatsItsValue(t *testing.T) {
	t.Parallel()

	bao := &tokenBao{}
	var logs bytes.Buffer
	c := newTokenClient(bao, fakeToken, &logs)
	if _, err := c.Read(t.Context(), "kv/data/x"); err != nil {
		t.Fatal(err)
	}

	tok := Token(fakeToken)
	login := StaticToken{Token: tok}
	auth, err := login.Login(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []any{tok, &tok, auth, &auth, login, &login, c} {
		secrettest.AssertFormatHidden(t, v, fakeToken)
	}
	secrettest.AssertHidden(t, fmt.Errorf("wrapped: %v", auth), fakeToken)

	slog.New(slog.NewJSONHandler(&logs, nil)).Info("oops", "token", tok, "auth", auth, "login", login)
	slog.New(slog.NewTextHandler(&logs, nil)).Info("oops", "token", tok, "auth", auth, "login", login)
	if strings.Contains(logs.String(), fakeToken) {
		t.Error("a log line contains the token")
	}
}
