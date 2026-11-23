package instagram

import (
	"errors"
	"testing"
	"time"
)

const (
	customAccessKey  = "ig_token"
	customRefreshKey = "ig_refreshed"
)

// newCustomKeyStore is newTestStore with custom key names.
func newCustomKeyStore(t *testing.T, bao *fakeBao, refresher Refresher) *TokenStore {
	t.Helper()
	store := newTestStore(t, bao, refresher, nil)
	store.AccessTokenKey = customAccessKey
	store.RefreshedAtKey = customRefreshKey
	return store
}

func TestTokenStore_customKeys_freshTokenReadFromTheConfiguredKey(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		customAccessKey:  "custom-current",
		customRefreshKey: testNow.Add(-time.Hour).Format(time.RFC3339),
		// The default names hold something else: they must be ignored.
		"access_token": "default-key-value",
		"refreshed_at": testNow.Add(-1000 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 2
	refresher := &fakeRefresher{fresh: "should-not-be-used"}

	token, err := newCustomKeyStore(t, bao, refresher).Token(t.Context())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token != "custom-current" {
		t.Errorf("token = %q, want the value under the configured key", token)
	}
	if refresher.callCount() != 0 || bao.writes != 0 {
		t.Errorf("refreshes = %d, writes = %d, want none: the configured refresh time is fresh", refresher.callCount(), bao.writes)
	}
}

func TestTokenStore_customKeys_refreshWritesOnlyTheTwoKeysAndKeepsTheRest(t *testing.T) {
	t.Parallel()

	staleAt := testNow.Add(-100 * time.Hour).Format(time.RFC3339)
	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		customAccessKey:  "custom-stale",
		customRefreshKey: staleAt,
		"app_id":         "unrelated-value",
		"access_token":   "default-key-value",
	}
	bao.versions[testPath] = 4
	refresher := &fakeRefresher{fresh: "custom-fresh"}

	token, err := newCustomKeyStore(t, bao, refresher).Token(t.Context())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token != "custom-fresh" || refresher.lastToken != "custom-stale" {
		t.Errorf("token = %q, refreshed %q, want custom-fresh from custom-stale", token, refresher.lastToken)
	}

	want := map[string]any{
		customAccessKey:  "custom-fresh",
		customRefreshKey: testNow.UTC().Format(time.RFC3339),
		"app_id":         "unrelated-value",
		"access_token":   "default-key-value",
	}
	stored := bao.secrets[testPath]
	if len(stored) != len(want) {
		t.Errorf("stored %d keys, want exactly %d: no key added, none dropped", len(stored), len(want))
	}
	for k, v := range want {
		if stored[k] != v {
			t.Errorf("stored[%s] = %v, want %v", k, stored[k], v)
		}
	}
	if _, ok := stored["refreshed_at"]; ok {
		t.Error("a refreshed_at key was written although the configured key is another")
	}
	// The check-and-set version was the one read, so it is now one higher.
	if bao.versions[testPath] != 5 {
		t.Errorf("version = %d, want 5 after one check-and-set write", bao.versions[testPath])
	}
}

func TestTokenStore_customKeys_onlyTheDefaultKeyPresentIsNoToken(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		"access_token": "default-key-value",
		"refreshed_at": testNow.Format(time.RFC3339),
	}
	bao.versions[testPath] = 1
	refresher := &fakeRefresher{fresh: "should-not-be-used"}

	if _, err := newCustomKeyStore(t, bao, refresher).Token(t.Context()); err == nil {
		t.Fatal("Token() error = nil, want an error: the configured key is missing")
	}
	if refresher.callCount() != 0 {
		t.Errorf("refresher called %d times, want 0", refresher.callCount())
	}
}

func TestTokenStore_customKeys_casMismatchRereadsTheConfiguredKey(t *testing.T) {
	t.Parallel()

	bao := newFakeBao()
	bao.exists[testPath] = true
	bao.secrets[testPath] = map[string]any{
		customAccessKey:  "custom-stale",
		customRefreshKey: testNow.Add(-100 * time.Hour).Format(time.RFC3339),
	}
	bao.versions[testPath] = 1
	bao.simulateRaceOnce = true
	bao.winnerSecret = map[string]any{
		customAccessKey:  "custom-winner",
		customRefreshKey: testNow.Format(time.RFC3339),
		"access_token":   "default-key-value",
	}
	refresher := &fakeRefresher{fresh: "custom-fresh"}

	token, err := newCustomKeyStore(t, bao, refresher).Token(t.Context())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token != "custom-winner" {
		t.Errorf("token = %q, want the other replica's token under the configured key", token)
	}
}

func TestTokenStore_missingKeyNamesIsAnErrorBeforeAnyRequest(t *testing.T) {
	t.Parallel()

	for name, unset := range map[string]func(*TokenStore){
		"access token key": func(s *TokenStore) { s.AccessTokenKey = "" },
		"refreshed at key": func(s *TokenStore) { s.RefreshedAtKey = "" },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bao := newFakeBao()
			store := newTestStore(t, bao, &fakeRefresher{}, nil)
			unset(store)
			if _, err := store.Token(t.Context()); !errors.Is(err, errNoKeys) {
				t.Fatalf("Token() error = %v, want errNoKeys", err)
			}
		})
	}
}
