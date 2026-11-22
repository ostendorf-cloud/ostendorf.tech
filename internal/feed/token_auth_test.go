package feed

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/feedconfig"
)

const fakeOpenBaoToken = "s.fake-feed-token-must-never-leak-e08b"

// forbiddingBao answers every request with 403, the way OpenBao answers an
// expired or revoked token, and counts what it was sent.
type forbiddingBao struct {
	mu       sync.Mutex
	requests int
	tokens   []string
}

func (f *forbiddingBao) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests++
	f.tokens = append(f.tokens, r.Header.Get("X-Vault-Token"))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"errors":["permission denied"]}`))
}

// TestTokenMethodForbiddenStaysUnreadyWithoutRetry builds the real App
// through feedconfig.Load with the token method against a fake OpenBao that
// refuses the token. One cycle runs: it must send exactly one request with
// the configured token, leave /readyz at 503, and log no token.
func TestTokenMethodForbiddenStaysUnreadyWithoutRetry(t *testing.T) {
	t.Parallel()

	bao := &forbiddingBao{}
	server := httptest.NewServer(bao)
	t.Cleanup(server.Close)

	cfg, err := feedconfig.Load(func(key string) string {
		return map[string]string{
			"OPENBAO_ADDR":        server.URL,
			"OPENBAO_AUTH_METHOD": "token",
			"OPENBAO_TOKEN":       fakeOpenBaoToken,
		}[key]
	})
	if err != nil {
		t.Fatalf("feedconfig.Load() error = %v", err)
	}

	var logs bytes.Buffer
	a := New(cfg, slog.New(slog.NewTextHandler(&logs, nil)))

	// Long enough for one cycle, far shorter than the retry wait after it.
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	a.run(ctx)

	bao.mu.Lock()
	requests, tokens := bao.requests, bao.tokens
	bao.mu.Unlock()
	if requests != 1 {
		t.Errorf("OpenBao saw %d requests in one cycle, want 1: a static token's 403 is not retried", requests)
	}
	for _, tok := range tokens {
		if tok != fakeOpenBaoToken {
			t.Error("a request did not carry the configured token as X-Vault-Token")
		}
	}

	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/readyz = %d, want 503 while OpenBao refuses the token", rec.Code)
	}
	if strings.Contains(logs.String(), fakeOpenBaoToken) || strings.Contains(rec.Body.String(), fakeOpenBaoToken) {
		t.Error("the log or the readiness answer contains the token")
	}
	if !strings.Contains(logs.String(), "permission denied for the configured token") {
		t.Error("the failed cycle did not log the fixed forbidden error")
	}
}
