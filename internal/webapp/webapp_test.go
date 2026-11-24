package webapp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
)

// discardLogger is a logger that writes nowhere, for tests that care about
// behaviour, not log output.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// validHexKey is a 32 byte key, hex encoded, long enough for CHALLENGE_KEY.
var validHexKey = strings.Repeat("ab", 32)

// testConfig loads a real config.Config through config.Load, so every
// default (SiteURL, ContactFrom/To, the rate limits) is the one the real
// service starts with, with only the fields a test cares about overridden
// on top.
func testConfig(t *testing.T, env map[string]string) config.Config {
	t.Helper()
	cfg, err := config.Load(func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	return cfg
}

// errorCode decodes the api package's one error envelope,
// {"error":{"code":...,"message":...}}, and returns the stable code, the
// part the frontend translates by.
func errorCode(t *testing.T, body []byte) string {
	t.Helper()
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	return envelope.Error.Code
}

func testApp(t *testing.T) *App {
	t.Helper()
	return testAppWith(t, map[string]string{
		"RATE_LIMIT_BURST": "100",
		"RATE_LIMIT_PER":   "1s",
	})
}

func testAppWith(t *testing.T, env map[string]string) *App {
	t.Helper()
	app, err := New(testConfig(t, env), discardLogger())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return app
}

func get(t *testing.T, h http.Handler, path, remoteAddr string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestAppPublicHandlerServesNoOperationalEndpoints proves /healthz, /readyz
// and /metrics answer on the admin handler only: the public handler gives
// its JSON 404 for all three.
func TestAppPublicHandlerServesNoOperationalEndpoints(t *testing.T) {
	app := testApp(t)
	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		if rec := get(t, app.Handler(), path, "", nil); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s on the public handler status = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
	}
}

// TestAppAdminHandlerServesHealthReadinessAndMetrics proves the admin
// handler answers all three operational routes; see AdminHandler.
func TestAppAdminHandlerServesHealthReadinessAndMetrics(t *testing.T) {
	app := testApp(t)
	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		if rec := get(t, app.AdminHandler(), path, "", nil); rec.Code != http.StatusOK {
			t.Errorf("GET %s on the admin handler status = %d, want %d", path, rec.Code, http.StatusOK)
		}
	}
}

func TestAppDrainingFailsReadiness(t *testing.T) {
	app := testApp(t)
	app.Health().Drain()
	if rec := get(t, app.AdminHandler(), "/readyz", "", nil); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

// TestAppSecurityPolicyAllowsTheProofOfWorkWorker proves worker-src 'self'
// is on the response exactly once, with no other source: govue's contact
// form Worker loads from a same-origin module URL, and the consequence of
// this header being wrong is that the form never submits. govue's default
// policy carries the directive since v0.8.0, so the site adds nothing.
func TestAppSecurityPolicyAllowsTheProofOfWorkWorker(t *testing.T) {
	csp := get(t, testApp(t).Handler(), "/api/v1/instagram", "", nil).Header().Get("Content-Security-Policy")
	var worker []string
	for _, directive := range strings.Split(csp, ";") {
		if fields := strings.Fields(directive); len(fields) > 0 && fields[0] == "worker-src" {
			worker = append(worker, strings.Join(fields[1:], " "))
		}
	}
	if len(worker) != 1 || worker[0] != "'self'" {
		t.Errorf("worker-src directives = %q in %q, want one with 'self' only", worker, csp)
	}
}

// TestAppServesNoPage proves the API serves no page and no file of the
// build: every path outside its routes is the JSON 404.
func TestAppServesNoPage(t *testing.T) {
	app := testApp(t)
	for _, path := range []string{
		"/", "/de/", "/de/imprint", "/impressum", "/sitemap.xml", "/robots.txt",
		"/favicon.svg", "/assets/index.js", "/api/v1/nope",
	} {
		rec := get(t, app.Handler(), path, "", nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("GET %s Content-Type = %q, want JSON", path, ct)
		}
	}
}

// TestAppRateLimitAnswersJSON proves the general rate limit sits in front
// of the API routes and answers govue's JSON envelope with Retry-After.
func TestAppRateLimitAnswersJSON(t *testing.T) {
	app := testAppWith(t, map[string]string{"RATE_LIMIT_BURST": "1", "RATE_LIMIT_PER": "1m"})
	const addr = "203.0.113.1:1234"
	if rec := get(t, app.Handler(), "/api/v1/instagram", addr, nil); rec.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d", rec.Code, http.StatusOK)
	}
	rec := get(t, app.Handler(), "/api/v1/instagram", addr, nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("no Retry-After")
	}
}

// TestAppClientIPWiringHonoursTrustedProxy proves TRUSTED_PROXIES changes
// what the rate limiter keys on: a request relayed through a trusted proxy
// is attributed to the address it names in X-Forwarded-For, and an
// untrusted peer's header is ignored.
func TestAppClientIPWiringHonoursTrustedProxy(t *testing.T) {
	app := testAppWith(t, map[string]string{
		"TRUSTED_PROXIES":  "203.0.113.0/24",
		"RATE_LIMIT_BURST": "1",
		"RATE_LIMIT_PER":   "1m",
	})
	via := func(remoteAddr, forwardedFor string) int {
		return get(t, app.Handler(), "/api/v1/instagram", remoteAddr,
			http.Header{"X-Forwarded-For": {forwardedFor}}).Code
	}
	const trustedProxy = "203.0.113.1:1234"
	for i, tc := range []struct {
		peer, forwarded string
		want            int
	}{
		{trustedProxy, "198.51.100.1", http.StatusOK},
		{trustedProxy, "198.51.100.2", http.StatusOK},
		{trustedProxy, "198.51.100.1", http.StatusTooManyRequests},
		{"192.0.2.1:1234", "198.51.100.3", http.StatusOK},
		{"192.0.2.1:1234", "198.51.100.4", http.StatusTooManyRequests},
	} {
		if got := via(tc.peer, tc.forwarded); got != tc.want {
			t.Errorf("request %d status = %d, want %d", i, got, tc.want)
		}
	}
}
