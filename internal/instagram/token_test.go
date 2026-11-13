package instagram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/openbao"
)

const (
	testAuthPath = "k8s"
	testRole     = "feed"
	testPath     = "instagram.com/data/test"
)

// testNow (declared in instagram_test.go) is the fixed clock every test
// below reads through TokenStore.Now too, so nothing here depends on the
// wall clock either, and the tests stay safe to run in parallel.

// fakeBao is a minimal OpenBao: Kubernetes login plus one KV v2 path, driven
// over a real loopback httptest.Server so the test exercises *openbao.Client
// through its exported surface only, never its unexported fields.
type fakeBao struct {
	mu       sync.Mutex
	secrets  map[string]map[string]any
	versions map[string]int
	exists   map[string]bool
	writes   int

	// simulateRaceOnce makes the next write land after another replica's
	// write already changed the version, so the CAS check this client
	// sends fails the way a real concurrent refresh would.
	simulateRaceOnce bool
	winnerSecret     map[string]any

	// failWriteOnce makes the next write answer with a server error that is
	// not a CAS mismatch, the way an OpenBao outage would.
	failWriteOnce bool
}

func newFakeBao() *fakeBao {
	return &fakeBao{secrets: map[string]map[string]any{}, versions: map[string]int{}, exists: map[string]bool{}}
}

func (f *fakeBao) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path == "/v1/auth/"+testAuthPath+"/login" && r.Method == http.MethodPost {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"auth": map[string]any{"client_token": "test-vault-token", "lease_duration": 3600},
		})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/")
	switch r.Method {
	case http.MethodGet:
		if !f.exists[path] {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[]}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"data": f.secrets[path], "metadata": map[string]any{"version": f.versions[path]},
		}})
	case http.MethodPost:
		f.writes++
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			Data    map[string]any `json:"data"`
			Options *struct {
				CAS int `json:"cas"`
			} `json:"options"`
		}
		_ = json.Unmarshal(raw, &body)

		if f.failWriteOnce {
			f.failWriteOnce = false
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":["internal error"]}`))
			return
		}
		if f.simulateRaceOnce {
			f.simulateRaceOnce = false
			f.secrets[path] = f.winnerSecret
			f.versions[path]++
			f.exists[path] = true
		}
		if body.Options != nil && body.Options.CAS != f.versions[path] {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errors":["check-and-set parameter did not match the current version"]}`))
			return
		}
		f.secrets[path] = body.Data
		f.versions[path]++
		f.exists[path] = true
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"version": f.versions[path]}})
	}
}

// newTestStore builds an *openbao.Client against bao, served from a real
// loopback httptest.Server, and a TokenStore on top of it with a fake
// refresher and the fixed test clock.
func newTestStore(t *testing.T, bao *fakeBao, refresher Refresher, logs io.Writer) *TokenStore {
	t.Helper()
	server := httptest.NewServer(bao)
	t.Cleanup(server.Close)

	jwtPath := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(jwtPath, []byte("test-service-account-jwt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if logs == nil {
		logs = io.Discard
	}
	client := openbao.New(server.URL, testAuthPath, testRole, jwtPath, slog.New(slog.NewTextHandler(logs, nil)))

	store := &TokenStore{
		Store:      client,
		Refresher:  refresher,
		Path:       testPath,
		RefreshAge: 24 * time.Hour,
		Logger:     slog.New(slog.NewTextHandler(logs, nil)),
		Now:        func() time.Time { return testNow },
	}
	return store
}

// fakeRefresher is a hand written stand in for Meta's refresh endpoint.
type fakeRefresher struct {
	mu        sync.Mutex
	calls     int
	lastToken string
	fresh     string
	err       error
}

func (f *fakeRefresher) RefreshToken(_ context.Context, token string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastToken = token
	if f.err != nil {
		return "", f.err
	}
	return f.fresh, nil
}

func (f *fakeRefresher) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestTokenStore_Token_freshTokenUsedAsIs(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": "current-token",
		"refreshed_at": testNow.Add(-1 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 3
	refresher := &fakeRefresher{fresh: "should-not-be-used"}
	store := newTestStore(t, bao, refresher, nil)

	token, err := store.Token(t.Context())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token != "current-token" {
		t.Errorf("token = %q, want the unrefreshed current token", token)
	}
	if refresher.callCount() != 0 {
		t.Errorf("refresher called %d times, want 0 for a fresh token", refresher.callCount())
	}
	if bao.writes != 0 {
		t.Errorf("wrote %d times, want 0 for a fresh token", bao.writes)
	}
}

func TestTokenStore_Token_staleTokenRefreshesAndWritesBackPreservingOtherKeys(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": "stale-token",
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
		"note":         "keep me",
	}
	bao.versions[testPath] = 5
	refresher := &fakeRefresher{fresh: "fresh-token"}
	store := newTestStore(t, bao, refresher, nil)

	token, err := store.Token(t.Context())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token != "fresh-token" {
		t.Errorf("token = %q, want the refreshed token", token)
	}
	if refresher.callCount() != 1 {
		t.Errorf("refresher called %d times, want 1", refresher.callCount())
	}
	if refresher.lastToken != "stale-token" {
		t.Errorf("refresher was asked to refresh %q, want the stale token", refresher.lastToken)
	}

	stored := bao.secrets[testPath]
	if stored["access_token"] != "fresh-token" {
		t.Errorf("stored access_token = %v, want fresh-token", stored["access_token"])
	}
	if stored["refreshed_at"] != testNow.UTC().Format(time.RFC3339) {
		t.Errorf("stored refreshed_at = %v, want %v", stored["refreshed_at"], testNow.UTC().Format(time.RFC3339))
	}
	if stored["note"] != "keep me" {
		t.Errorf("stored note = %v, want the preexisting key preserved", stored["note"])
	}
}

func TestTokenStore_Token_writeUsesTheVersionThatWasRead(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": "stale-token",
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 7
	refresher := &fakeRefresher{fresh: "fresh-token"}
	store := newTestStore(t, bao, refresher, nil)

	if _, err := store.Token(t.Context()); err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	// The write succeeded at all only because the CAS value matched the
	// version 7 that was read; version is now 8.
	if bao.versions[testPath] != 8 {
		t.Errorf("version after write = %d, want 8 (one past the version that was read)", bao.versions[testPath])
	}
}

func TestTokenStore_Token_casMismatchReReadsAndUsesTheStoredToken(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": "stale-token",
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 3
	bao.simulateRaceOnce = true
	bao.winnerSecret = map[string]any{
		"access_token": "winner-token",
		"refreshed_at": testNow.Add(-1 * time.Minute).Format(time.RFC3339),
	}
	refresher := &fakeRefresher{fresh: "fresh-token"}
	store := newTestStore(t, bao, refresher, nil)

	token, err := store.Token(t.Context())
	if err != nil {
		t.Fatalf("Token() error = %v, want nil: a CAS mismatch is not a failure", err)
	}
	if token != "winner-token" {
		t.Errorf("token = %q, want the token the other replica stored", token)
	}
}

func TestTokenStore_Token_refreshFailureKeepsCurrentTokenAndDoesNotWrite(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": "current-token",
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 2
	refresher := &fakeRefresher{err: errors.New("meta: rate limited")}
	store := newTestStore(t, bao, refresher, nil)

	token, err := store.Token(t.Context())
	if err != nil {
		t.Fatalf("Token() error = %v, want nil: a refresh failure keeps the current token", err)
	}
	if token != "current-token" {
		t.Errorf("token = %q, want the current token unchanged", token)
	}
	if bao.writes != 0 {
		t.Errorf("wrote %d times, want 0 when the refresh itself failed", bao.writes)
	}
}

func TestTokenStore_Token_writeFailureAfterRefreshUsesTheFreshTokenThisCycle(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": "stale-token",
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 4
	bao.failWriteOnce = true
	refresher := &fakeRefresher{fresh: "fresh-token"}
	store := newTestStore(t, bao, refresher, nil)

	token, err := store.Token(t.Context())
	if err != nil {
		t.Fatalf("Token() error = %v, want nil: a write failure after a refresh still uses the fresh token", err)
	}
	if token != "fresh-token" {
		t.Errorf("token = %q, want the freshly fetched token even though storing it failed", token)
	}
	if bao.secrets[testPath]["access_token"] != "stale-token" {
		t.Errorf("stored access_token = %v, want the stale value left untouched by the failed write", bao.secrets[testPath]["access_token"])
	}
}

func TestTokenStore_Token_noUsableToken(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup   func(*fakeBao)
		wantErr error // checked with errors.Is when non-nil
	}{
		"nothing was ever written at the path": {
			setup:   func(*fakeBao) {},
			wantErr: openbao.ErrNotFound,
		},
		"a secret exists but carries no access_token key": {
			setup: func(b *fakeBao) {
				b.exists[testPath] = true
				b.secrets[testPath] = map[string]any{"refreshed_at": testNow.Format(time.RFC3339)}
				b.versions[testPath] = 1
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bao := newFakeBao()
			tc.setup(bao)
			refresher := &fakeRefresher{fresh: "should-not-be-used"}
			store := newTestStore(t, bao, refresher, nil)

			_, err := store.Token(t.Context())
			if err == nil {
				t.Fatal("Token() error = nil, want an error: OpenBao is the only source and there is no fallback")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want errors.Is(err, %v)", err, tc.wantErr)
			}
			if refresher.callCount() != 0 {
				t.Errorf("refresher called %d times, want 0 with no usable token to refresh", refresher.callCount())
			}
		})
	}
}

func TestTokenStore_Token_missingOrMalformedRefreshedAtTriggersARefresh(t *testing.T) {
	t.Parallel()

	tests := map[string]map[string]any{
		"refreshed_at key is absent": {
			"access_token": "current-token",
		},
		"refreshed_at is not RFC3339": {
			"access_token": "current-token",
			"refreshed_at": "2026-09-30",
		},
		"refreshed_at is not a string at all": {
			"access_token": "current-token",
			"refreshed_at": 12345,
		},
	}

	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bao := newFakeBao()
			bao.exists[testPath] = true
			bao.secrets[testPath] = data
			bao.versions[testPath] = 1
			refresher := &fakeRefresher{fresh: "fresh-token"}
			store := newTestStore(t, bao, refresher, nil)

			token, err := store.Token(t.Context())
			if err != nil {
				t.Fatalf("Token() error = %v", err)
			}
			if token != "fresh-token" {
				t.Errorf("token = %q, want a refresh to have been attempted and used", token)
			}
			if refresher.callCount() != 1 {
				t.Errorf("refresher called %d times, want exactly 1", refresher.callCount())
			}
		})
	}
}

// TestTokenStore_Token_neverLeaksATokenIntoAnErrorOrALogLine drives every
// path through Token, including the degraded ones, with distinctive token
// values, and checks that none of them ever appears in a returned error or
// in anything written to the logger.
func TestTokenStore_Token_neverLeaksATokenIntoAnErrorOrALogLine(t *testing.T) {
	t.Parallel()

	const (
		currentToken = "current-token-should-never-be-logged"
		freshToken   = "fresh-token-should-never-be-logged"
		winnerToken  = "winner-token-should-never-be-logged"
	)
	var logs bytes.Buffer
	var errs []string
	record := func(err error) {
		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	// Fresh read, no refresh: nothing to leak, but exercised anyway.
	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": currentToken,
		"refreshed_at": testNow.Add(-1 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 1
	refresher := &fakeRefresher{fresh: freshToken}
	store := newTestStore(t, bao, refresher, &logs)
	_, err := store.Token(t.Context())
	record(err)

	// Stale: refresh and write back successfully.
	bao2 := newFakeBao()
	bao2.exists[testPath] = true
	bao2.secrets[testPath] = map[string]any{
		"access_token": currentToken,
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao2.versions[testPath] = 1
	store2 := newTestStore(t, bao2, refresher, &logs)
	_, err = store2.Token(t.Context())
	record(err)

	// CAS mismatch: re-reads and uses the winner's token.
	bao3 := newFakeBao()
	bao3.exists[testPath] = true
	bao3.secrets[testPath] = map[string]any{
		"access_token": currentToken,
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao3.versions[testPath] = 1
	bao3.simulateRaceOnce = true
	bao3.winnerSecret = map[string]any{"access_token": winnerToken, "refreshed_at": testNow.Format(time.RFC3339)}
	store3 := newTestStore(t, bao3, refresher, &logs)
	_, err = store3.Token(t.Context())
	record(err)

	// Refresh failure: the error from Meta might itself be adversarial;
	// keep it free of token-shaped content on purpose.
	bao4 := newFakeBao()
	bao4.exists[testPath] = true
	bao4.secrets[testPath] = map[string]any{
		"access_token": currentToken,
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao4.versions[testPath] = 1
	failingRefresher := &fakeRefresher{err: errors.New("meta: rate limited")}
	store4 := newTestStore(t, bao4, failingRefresher, &logs)
	_, err = store4.Token(t.Context())
	record(err)

	// Write failure after a successful refresh.
	bao5 := newFakeBao()
	bao5.exists[testPath] = true
	bao5.secrets[testPath] = map[string]any{
		"access_token": currentToken,
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao5.versions[testPath] = 1
	bao5.failWriteOnce = true
	store5 := newTestStore(t, bao5, refresher, &logs)
	_, err = store5.Token(t.Context())
	record(err)

	// No token stored at all.
	bao6 := newFakeBao()
	store6 := newTestStore(t, bao6, refresher, &logs)
	_, err = store6.Token(t.Context())
	record(err)

	for _, e := range errs {
		for _, secret := range []string{currentToken, freshToken, winnerToken} {
			if strings.Contains(e, secret) {
				t.Errorf("an error contains a token value: %q", e)
			}
		}
	}
	logged := logs.String()
	for _, secret := range []string{currentToken, freshToken, winnerToken} {
		if strings.Contains(logged, secret) {
			t.Errorf("the log contains a token value: %q", logged)
		}
	}
}
