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

// newTestClient builds an *openbao.Client against bao, served from a real
// loopback httptest.Server, so every test here exercises *openbao.Client
// through its exported surface only, never its unexported fields.
func newTestClient(t *testing.T, bao *fakeBao, logs io.Writer) *openbao.Client {
	t.Helper()
	server := httptest.NewServer(bao)
	t.Cleanup(server.Close)

	jwtPath := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(jwtPath, []byte("test-service-account-jwt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	login := openbao.Kubernetes{AuthPath: testAuthPath, Role: testRole, JWTFile: jwtPath}
	return openbao.New(server.URL, login, slog.New(slog.NewTextHandler(logs, nil)))
}

// newTestStore builds a TokenStore on top of newTestClient, with a fake
// refresher and the fixed test clock.
func newTestStore(t *testing.T, bao *fakeBao, refresher Refresher, logs io.Writer) *TokenStore {
	t.Helper()
	return newTestStoreWithClock(t, bao, refresher, logs, func() time.Time { return testNow })
}

// newTestStoreWithClock is newTestStore with an injectable clock, for a
// test that needs to move time forward between two calls to Token.
func newTestStoreWithClock(t *testing.T, bao *fakeBao, refresher Refresher, logs io.Writer, now func() time.Time) *TokenStore {
	t.Helper()
	if logs == nil {
		logs = io.Discard
	}
	return &TokenStore{
		Store:          newTestClient(t, bao, logs),
		Refresher:      refresher,
		Path:           testPath,
		AccessTokenKey: "access_token",
		RefreshedAtKey: "refreshed_at",
		RefreshAge:     24 * time.Hour,
		Logger:         slog.New(slog.NewTextHandler(logs, nil)),
		Now:            now,
	}
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

// TestTokenStore_Token_writeFailureEscalatesToErrorOncePersistent drives two
// write failures separated by writeFailureEscalateAfter and checks the
// second is reported at Error, not Warn: a write that keeps failing must
// read differently in the log than a single hiccup, since every restart
// until it is fixed falls back to the hand-written token, and the operator
// needs to notice before Meta's 60-day expiry does the noticing instead.
func TestTokenStore_Token_writeFailureEscalatesToErrorOncePersistent(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": "stale-token",
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 1
	bao.failWriteOnce = true
	refresher := &fakeRefresher{fresh: "fresh-token"}

	now := testNow
	var logs bytes.Buffer
	store := newTestStoreWithClock(t, bao, refresher, &logs, func() time.Time { return now })

	if _, err := store.Token(t.Context()); err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if strings.Contains(logs.String(), "level=ERROR") {
		t.Errorf("a single write failure logged at Error, want only Warn:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), "level=WARN") {
		t.Errorf("a single write failure did not log at Warn:\n%s", logs.String())
	}

	// The write never succeeded, so refreshed_at in storage is still
	// stale: the next call attempts another refresh and another write,
	// writeFailureEscalateAfter later.
	logs.Reset()
	now = testNow.Add(writeFailureEscalateAfter)
	bao.failWriteOnce = true
	if _, err := store.Token(t.Context()); err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if !strings.Contains(logs.String(), "level=ERROR") {
		t.Errorf("a write failure persisting %s did not escalate to Error:\n%s", writeFailureEscalateAfter, logs.String())
	}
}

// TestTokenStore_Token_successfulWriteResetsTheEscalationClock proves a
// write failure followed by a successful write clears unstoredSince, so a
// later, unrelated failure starts its own escalation window instead of
// inheriting the first one's: without the reset, the third call below
// would wrongly log at Error, more than writeFailureEscalateAfter after
// the first (unrelated) failure even though its own failure just started.
func TestTokenStore_Token_successfulWriteResetsTheEscalationClock(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": "stale-token",
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 1
	bao.failWriteOnce = true
	refresher := &fakeRefresher{fresh: "fresh-token"}

	now := testNow
	var logs bytes.Buffer
	store := &TokenStore{
		Store:          newTestClient(t, bao, &logs),
		Refresher:      refresher,
		Path:           testPath,
		AccessTokenKey: "access_token",
		RefreshedAtKey: "refreshed_at",
		RefreshAge:     time.Hour,
		Logger:         slog.New(slog.NewTextHandler(&logs, nil)),
		Now:            func() time.Time { return now },
	}

	// First call: write fails, unstoredSince is set to now.
	if _, err := store.Token(t.Context()); err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	// Second call, 30 hours later (comfortably past writeFailureEscalateAfter
	// if the clock were not reset): the token is still stale (the earlier
	// write never succeeded), and this write succeeds, which must clear
	// unstoredSince.
	now = testNow.Add(30 * time.Hour)
	if _, err := store.Token(t.Context()); err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	// Third call, 2 hours after the second (stale again against the 1 hour
	// RefreshAge, but nowhere near writeFailureEscalateAfter on its own):
	// this write fails too. If unstoredSince carried over from the first
	// failure, 32 hours would already be past writeFailureEscalateAfter.
	now = now.Add(2 * time.Hour)
	bao.failWriteOnce = true
	logs.Reset()
	if _, err := store.Token(t.Context()); err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if strings.Contains(logs.String(), "level=ERROR") {
		t.Errorf("write failure logged at Error right after the escalation clock was reset, want Warn:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), "level=WARN") {
		t.Errorf("write failure did not log at Warn:\n%s", logs.String())
	}
}

// TestTokenStore_Token_freshTokenResetsTheEscalationClock proves the early
// return for a token still within RefreshAge clears unstoredSince, the same
// as a successful own write does. That path is exactly what a cycle sees
// after another replica, or an operator by hand, persists a token following
// an earlier write failure here: without the reset, a single unrelated
// hiccup long afterwards would wrongly escalate to Error on its very first
// occurrence, inheriting a clock that should have been cleared.
func TestTokenStore_Token_freshTokenResetsTheEscalationClock(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": "stale-token",
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 1
	bao.failWriteOnce = true
	refresher := &fakeRefresher{fresh: "fresh-token"}

	now := testNow
	var logs bytes.Buffer
	store := newTestStoreWithClock(t, bao, refresher, &logs, func() time.Time { return now })

	// First call: write fails, unstoredSince is set to now.
	if _, err := store.Token(t.Context()); err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	// Between calls, another replica (or an operator) persists a fresh
	// token directly, the way this process never managed to.
	now = testNow.Add(2 * time.Hour)
	bao.secrets[testPath] = map[string]any{
		"access_token": "someone-elses-fresh-token",
		"refreshed_at": now.Format(time.RFC3339),
	}
	bao.versions[testPath]++
	callsBefore := refresher.callCount()
	logs.Reset()
	token, err := store.Token(t.Context())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token != "someone-elses-fresh-token" {
		t.Errorf("token = %q, want the fresh token now in OpenBao", token)
	}
	if refresher.callCount() != callsBefore {
		t.Errorf("refresher called %d more time(s), want 0 for a fresh token", refresher.callCount()-callsBefore)
	}

	// Third call, comfortably past writeFailureEscalateAfter measured from
	// the first failure: the stored token above is stale again by now
	// (RefreshAge is 24h), and this write fails too. Had the early return
	// above not cleared unstoredSince, this would already be more than
	// writeFailureEscalateAfter past the first failure and log at Error.
	now = now.Add(25 * time.Hour)
	bao.failWriteOnce = true
	logs.Reset()
	if _, err := store.Token(t.Context()); err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if strings.Contains(logs.String(), "level=ERROR") {
		t.Errorf("write failure logged at Error after the fresh-token path reset the escalation clock, want Warn:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), "level=WARN") {
		t.Errorf("write failure did not log at Warn:\n%s", logs.String())
	}
}

// TestTokenStore_Token_refreshFailureResetsTheEscalationClock covers the
// fourth path that means the token is persisted, beyond the successful own
// write, the CAS mismatch re-read and the fresh-token early return: a
// refresh attempt against Meta failing leaves Token returning exactly the
// token it just read from OpenBao, unmodified and never written to, so that
// token is persisted too and the clock must clear here as well.
func TestTokenStore_Token_refreshFailureResetsTheEscalationClock(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": "stale-token",
		"refreshed_at": testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 1
	bao.failWriteOnce = true
	refresher := &fakeRefresher{fresh: "fresh-token"}

	now := testNow
	var logs bytes.Buffer
	store := newTestStoreWithClock(t, bao, refresher, &logs, func() time.Time { return now })

	// First call: write fails, unstoredSince is set to now.
	if _, err := store.Token(t.Context()); err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	// Second call, 2 hours later: the stored token is still stale (the
	// earlier write never landed), but this time the refresh against Meta
	// itself fails, so Token returns exactly what it just read from
	// OpenBao and never attempts a write.
	now = testNow.Add(2 * time.Hour)
	refresher.err = errors.New("meta: rate limited")
	logs.Reset()
	writesBefore := bao.writes
	token, err := store.Token(t.Context())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token != "stale-token" {
		t.Errorf("token = %q, want the current token unchanged", token)
	}
	if bao.writes != writesBefore {
		t.Errorf("wrote %d more time(s), want 0 when the refresh itself failed", bao.writes-writesBefore)
	}

	// Third call, comfortably past writeFailureEscalateAfter measured from
	// the first failure but not from the second: a write failure here must
	// still log at Warn if the refresh failure above reset the clock.
	now = now.Add(23 * time.Hour)
	refresher.err = nil
	bao.failWriteOnce = true
	logs.Reset()
	if _, err := store.Token(t.Context()); err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if strings.Contains(logs.String(), "level=ERROR") {
		t.Errorf("write failure logged at Error after the refresh-failure path reset the escalation clock, want Warn:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), "level=WARN") {
		t.Errorf("write failure did not log at Warn:\n%s", logs.String())
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
