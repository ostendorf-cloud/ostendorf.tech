package webapp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ostendorf-cloud/govue/csrf"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/pow"
)

// discardLogger is a logger that writes nowhere, for tests that care about
// behaviour, not log output.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// validHexKey is a 32 byte key, hex encoded, long enough for CSRF_KEY and
// CHALLENGE_KEY.
var validHexKey = strings.Repeat("ab", 32)

// testConfig loads a real config.Config through config.Load, so every
// default (SiteURL, ContactFrom/To, the rate limits) is the one the real
// service starts with, with only the fields a test cares about overridden
// on top. CSRF_KEY is always set: config.Load fails without one.
func testConfig(t *testing.T, env map[string]string) config.Config {
	t.Helper()
	merged := map[string]string{"CSRF_KEY": validHexKey}
	for k, v := range env {
		merged[k] = v
	}
	cfg, err := config.Load(func(key string) string { return merged[key] })
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	return cfg
}

func testFS() (dist, prerendered fstest.MapFS) {
	dist = fstest.MapFS{
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('app')")},
	}
	prerendered = fstest.MapFS{
		"de/index.html": &fstest.MapFile{Data: []byte("<html><body>Startseite</body></html>")},
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

func TestAppServesLivenessAndReadiness(t *testing.T) {
	app := testApp(t)
	handler := app.Handler()

	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, http.StatusOK)
		}
	}
}

func TestAppDrainingFailsReadiness(t *testing.T) {
	app := testApp(t)
	app.Health().Drain()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestAppServesMetrics(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
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
	dist := fstest.MapFS{"assets/app.js": &fstest.MapFile{Data: []byte("x")}}
	prerendered := fstest.MapFS{
		"de/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
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
		"<loc>https://ostendorf.tech/de/imprint/</loc>",
		"<loc>https://ostendorf.tech/en/privacy/</loc>",
		"<loc>https://ostendorf.tech/de/agb/</loc>",
		`hreflang="en" href="https://ostendorf.tech/en/"`,
		`hreflang="x-default" href="https://ostendorf.tech/de/"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("sitemap lacks %q", want)
		}
	}
}

func TestAppChallengeUnavailableWithoutKey(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/challenge", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestAppChallengeIssuesTokenAndSessionCookie(t *testing.T) {
	cfg := testConfig(t, map[string]string{"CHALLENGE_KEY": validHexKey})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/challenge", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}
	if rec.Result().Cookies() == nil {
		t.Fatal("no session cookie set")
	}

	var body struct {
		Challenge string `json:"challenge"`
		Salt      string `json:"salt"`
		Signature string `json:"signature"`
		MaxNumber int64  `json:"maxnumber"`
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Challenge == "" || body.Salt == "" || body.Signature == "" {
		t.Errorf("challenge = %+v, missing a field", body)
	}
	if body.CSRFToken == "" {
		t.Error("csrfToken is empty")
	}
}

func TestAppContactUnavailableWithoutCredentials(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/contact",
		strings.NewReader(`{"name":"Erika","email":"erika@example.test","message":"hi","consent":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body)
	}
}

// contactTestApp builds an App with SMTP credentials and a challenge key
// configured, so the contact route runs its whole pipeline instead of
// stopping at the availability check.
func contactTestApp(t *testing.T) *App {
	t.Helper()
	cfg := testConfig(t, map[string]string{
		"CHALLENGE_KEY": validHexKey,
		"SMTP_USERNAME": "no-reply@example.test",
		"SMTP_PASSWORD": "s3cret",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return app
}

func TestAppContactHoneypotSilentlyAccepted(t *testing.T) {
	app := contactTestApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/contact",
		strings.NewReader(`{"name":"Eve","email":"eve@example.test","message":"buy now","consent":true,"website":"https://spam.example"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}
}

func TestAppContactRejectsMissingCSRFTokenOnceASessionExists(t *testing.T) {
	app := contactTestApp(t)

	// A GET first, to mint a session and its cookie, the same order a
	// real visitor's browser follows.
	getReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/challenge", nil)
	getRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(getRec, getReq)
	cookies := getRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie set by the challenge request")
	}

	postReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/contact",
		strings.NewReader(`{"name":"Erika","email":"erika@example.test","message":"hi","consent":true}`))
	postReq.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		postReq.AddCookie(c)
	}
	// No X-CSRF-Token header: the session now exists, so there is
	// something to forge, and the request must be refused.
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, postReq)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body)
	}
}

func TestAppContactFullPipelineReachesTheMailer(t *testing.T) {
	app := contactTestApp(t)

	// The challenge enforces a minimum age between issue and submission;
	// a fake clock lets the test cross it without a real sleep.
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	app.challenges.Now = func() time.Time { return now }

	getReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/challenge", nil)
	getRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("challenge status = %d, want %d, body = %s", getRec.Code, http.StatusOK, getRec.Body)
	}
	cookies := getRec.Result().Cookies()

	var challenge struct {
		pow.Challenge
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &challenge); err != nil {
		t.Fatalf("decode challenge: %v", err)
	}

	solution := solveChallenge(t, challenge.Challenge)
	now = now.Add(5 * time.Second)

	// The mailer never reaches a real network: Dial is replaced with one
	// that always fails, so this test proves the pipeline runs end to end
	// up to that boundary (session, CSRF, honeypot, validation, proof of
	// work) without needing a real SMTP server; internal/contact's own
	// tests already cover the SMTP conversation itself.
	dialErr := errors.New("dial refused")
	app.mailer.Dial = func(context.Context, string, string) (net.Conn, error) {
		return nil, dialErr
	}

	body, err := json.Marshal(map[string]any{
		"name": "Erika Mustermann", "email": "erika@example.test", "message": "Hallo, ich brauche Technik.",
		"consent": true, "pow": solution,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/contact", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(csrf.HeaderName, challenge.CSRFToken)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d (the mailer's dial failure), body = %s", rec.Code, http.StatusBadGateway, rec.Body)
	}
}

// solveChallenge finds the number the browser worker would, the same brute
// force internal/pow's own tests use, and returns the full solution ready
// to post back.
func solveChallenge(t *testing.T, challenge pow.Challenge) pow.Solution {
	t.Helper()
	for n := int64(0); n <= challenge.MaxNumber; n++ {
		if pow.Hash(challenge.Salt, n) == challenge.Challenge {
			return pow.Solution{
				Algorithm: challenge.Algorithm,
				Challenge: challenge.Challenge,
				Number:    n,
				Salt:      challenge.Salt,
				Signature: challenge.Signature,
			}
		}
	}
	t.Fatal("no solution found")
	return pow.Solution{}
}
