package webapp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

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

func testFS() (dist, prerendered fstest.MapFS) {
	dist = testDist()
	prerendered = fstest.MapFS{
		// de/index.html carries the theme-boot marker every real
		// prerendered page does, so New's themeBootScriptHash finds
		// something to hash, the same as it would against the real
		// build output.
		"de/index.html": &fstest.MapFile{Data: []byte(`<html><head><script id="theme-boot">(function(){})();</script></head><body>Startseite</body></html>`)},
		"de/404.html":   &fstest.MapFile{Data: []byte("<html><body>Nicht gefunden</body></html>")},
		"en/index.html": &fstest.MapFile{Data: []byte("<html><body>Home</body></html>")},
		"en/404.html":   &fstest.MapFile{Data: []byte("<html><body>Not found</body></html>")},
	}
	return dist, prerendered
}

func testApp(t *testing.T) *App {
	t.Helper()
	cfg := testConfig(t, map[string]string{
		"RATE_LIMIT_BURST": "100",
		"RATE_LIMIT_PER":   "1s",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return app
}

// TestAppPublicHandlerServesNoOperationalEndpoints proves the move to the
// admin listener actually happened on the public side: /healthz, /readyz
// and /metrics moved to AdminHandler (see
// TestAppAdminHandlerServesHealthReadinessAndMetrics), so none of the three
// may still answer on the public handler. pageHandler's catch-all answers
// every unknown path with the locale's own 404 page, which is what a
// request for any of these three now actually is from the public
// handler's point of view.
func TestAppPublicHandlerServesNoOperationalEndpoints(t *testing.T) {
	app := testApp(t)
	handler := app.Handler()

	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s on the public handler status = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
	}
}

// TestAppAdminHandlerServesHealthReadinessAndMetrics proves the admin
// handler, not the public one, now answers all three operational routes;
// see AdminHandler.
func TestAppAdminHandlerServesHealthReadinessAndMetrics(t *testing.T) {
	app := testApp(t)
	handler := app.AdminHandler()

	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s on the admin handler status = %d, want %d", path, rec.Code, http.StatusOK)
		}
	}
}

func TestAppDrainingFailsReadiness(t *testing.T) {
	app := testApp(t)
	app.Health().Drain()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	app.AdminHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

// TestAppSecurityPolicyAllowsTheProofOfWorkWorker proves worker-src 'self'
// is actually on the response: govue's contact form Worker loads from a
// same-origin module URL (see webapp.go's own comment on why this is set
// explicitly rather than left to CSP's script-src fallback), and the
// consequence of this header being wrong is that the form never submits.
func TestAppSecurityPolicyAllowsTheProofOfWorkWorker(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "worker-src 'self'") {
		t.Errorf("Content-Security-Policy = %q, want it to contain worker-src 'self'", csp)
	}
}

func TestAppServesPrerenderedHome(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAppRateLimitsExcessRequests(t *testing.T) {
	cfg := testConfig(t, map[string]string{
		"RATE_LIMIT_BURST": "1",
		"RATE_LIMIT_PER":   "1m",
	})
	dist := testDist()
	prerendered := fstest.MapFS{
		"de/index.html": &fstest.MapFile{Data: []byte(`<html><head><script id="theme-boot">(function(){})();</script></head></html>`)},
		"de/404.html":   &fstest.MapFile{Data: []byte("<html></html>")},
	}
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	for i, wantStatus := range []int{http.StatusOK, http.StatusTooManyRequests} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/", nil)
		req.RemoteAddr = "203.0.113.1:1234"
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != wantStatus {
			t.Errorf("request %d status = %d, want %d", i, rec.Code, wantStatus)
		}
	}
}

func TestAppUnknownRouteServesNotFound(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/does-not-exist/", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAppServesRobotsTxt(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/robots.txt", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Allow: /") || !strings.Contains(body, "Sitemap: https://ostendorf.tech/sitemap.xml") {
		t.Errorf("body = %q, missing Allow or Sitemap line", body)
	}
}

func TestAppSitemapListsEveryRouteOfEveryLocaleWithAlternates(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sitemap.xml", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<loc>https://ostendorf.tech/de/</loc>",
		"<loc>https://ostendorf.tech/en/</loc>",
		"<loc>https://ostendorf.tech/de/imprint</loc>",
		"<loc>https://ostendorf.tech/en/privacy</loc>",
		"<loc>https://ostendorf.tech/de/terms</loc>",
		`hreflang="en" href="https://ostendorf.tech/en/"`,
		`hreflang="x-default" href="https://ostendorf.tech/de/"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sitemap lacks %q", want)
		}
	}
}

// TestAppClientIPWiringHonoursTrustedProxy proves TRUSTED_PROXIES
// actually changes what the rate limiter keys on, not only that config.Load
// parses it (internal/config's own tests already cover that): a request
// relayed through a trusted proxy is attributed to the address it names in
// X-Forwarded-For, not the proxy's own TCP peer, so two distinct visitors
// behind the same proxy get two distinct buckets.
func TestAppClientIPWiringHonoursTrustedProxy(t *testing.T) {
	cfg := testConfig(t, map[string]string{
		"TRUSTED_PROXIES":  "203.0.113.0/24",
		"RATE_LIMIT_BURST": "1",
		"RATE_LIMIT_PER":   "1m",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	get := func(remoteAddr, forwardedFor string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/", nil)
		req.RemoteAddr = remoteAddr
		if forwardedFor != "" {
			req.Header.Set("X-Forwarded-For", forwardedFor)
		}
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		return rec
	}

	const trustedProxy = "203.0.113.1:1234"
	if rec := get(trustedProxy, "198.51.100.1"); rec.Code != http.StatusOK {
		t.Fatalf("first forwarded client status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}
	if rec := get(trustedProxy, "198.51.100.2"); rec.Code != http.StatusOK {
		t.Fatalf("second, distinct forwarded client status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}
	if rec := get(trustedProxy, "198.51.100.1"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("repeat of the first forwarded client status = %d, want %d (its own burst spent), body = %s",
			rec.Code, http.StatusTooManyRequests, rec.Body)
	}

	// An untrusted peer's X-Forwarded-For must never be honoured: both
	// requests collapse into the one bucket the TCP peer itself gets.
	const untrustedPeer = "192.0.2.1:1234"
	if rec := get(untrustedPeer, "198.51.100.3"); rec.Code != http.StatusOK {
		t.Fatalf("first request from the untrusted peer status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}
	if rec := get(untrustedPeer, "198.51.100.4"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request from the same untrusted peer with a different spoofed X-Forwarded-For status = %d, want %d "+
			"(the spoofed header must be ignored), body = %s", rec.Code, http.StatusTooManyRequests, rec.Body)
	}
}

// TestAppPageRateLimitAnswersHTMLInTheVisitorsLocale proves
// pageRateLimitRespond is wired into the page route's rate limiter: a 429 on
// a page GET has to read as the locale's own page, not govue's JSON error
// envelope a browser navigating to a page never expects.
func TestAppPageRateLimitAnswersHTMLInTheVisitorsLocale(t *testing.T) {
	cfg := testConfig(t, map[string]string{
		"RATE_LIMIT_BURST": "1",
		"RATE_LIMIT_PER":   "1m",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	get := func() *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/", nil)
		req.RemoteAddr = "203.0.113.42:1234"
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		return rec
	}

	if first := get(); first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d, body = %s", first.Code, http.StatusOK, first.Body)
	}
	second := get()
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d, body = %s", second.Code, http.StatusTooManyRequests, second.Body)
	}
	if ct := second.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html, not govue's JSON error envelope", ct)
	}
	if !strings.Contains(second.Body.String(), pageMessagesDE.tooMany) {
		t.Errorf("body = %q, want it to contain the localized tooMany message %q", second.Body.String(), pageMessagesDE.tooMany)
	}
}

// TestAppAssetRateLimitStaysJSON proves pageRateLimitRespond's fallback: a
// path that resolves to no locale, an asset in this case, keeps govue's
// default JSON error shape rather than the page branch's HTML.
func TestAppAssetRateLimitStaysJSON(t *testing.T) {
	cfg := testConfig(t, map[string]string{
		"RATE_LIMIT_BURST": "1",
		"RATE_LIMIT_PER":   "1m",
	})
	dist := testDist()
	prerendered := fstest.MapFS{
		"de/index.html": &fstest.MapFile{Data: []byte(`<html><head><script id="theme-boot">(function(){})();</script></head></html>`)},
		"de/404.html":   &fstest.MapFile{Data: []byte("<html></html>")},
	}
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	get := func() *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/assets/app.js", nil)
		req.RemoteAddr = "203.0.113.43:1234"
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		return rec
	}

	if first := get(); first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d, body = %s", first.Code, http.StatusOK, first.Body)
	}
	second := get()
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d, body = %s", second.Code, http.StatusTooManyRequests, second.Body)
	}
	if ct := second.Header().Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want govue's JSON shape, not HTML, for a non-page path", ct)
	}
}
