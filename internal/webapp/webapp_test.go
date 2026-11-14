package webapp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ostendorf-cloud/govue/api"
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

// decodeErrorMessage decodes the api package's one error envelope,
// {"error":{"code":...,"message":...}}, and returns the message, so a test
// can tell a localized message from the fixed "internal error" placeholder
// a 5xx would otherwise collapse to.
func decodeErrorMessage(t *testing.T, body []byte) string {
	t.Helper()
	_, message := decodeError(t, body)
	return message
}

// decodeError is decodeErrorMessage plus the code, for a test that also has
// to prove a 5xx collapsed to govue's generic shape rather than merely
// checking the status.
func decodeError(t *testing.T, body []byte) (code, message string) {
	t.Helper()
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	return envelope.Error.Code, envelope.Error.Message
}

func testFS() (dist, prerendered fstest.MapFS) {
	dist = fstest.MapFS{
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('app')")},
	}
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
// is actually on the response: the contact form's Worker loads from a
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
	dist := fstest.MapFS{"assets/app.js": &fstest.MapFile{Data: []byte("x")}}
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
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body)
	}
	// 503 would normally make api.WriteError collapse this to the fixed
	// "internal error" placeholder; handleChallenge marks this message
	// Exposed, so the localized text has to survive regardless.
	message := decodeErrorMessage(t, rec.Body.Bytes())
	if message != contactMessagesDE.unavailable {
		t.Errorf("message = %q, want the localized unavailable message %q", message, contactMessagesDE.unavailable)
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

// TestAppChallengeRateLimitAnswersInTheVisitorsLocale proves
// challengeRateLimitRespond is actually wired in: a 429 from GET
// /api/v1/challenge's own rate limiter must read in the locale the request
// asked for, not govue's fixed English default.
func TestAppChallengeRateLimitAnswersInTheVisitorsLocale(t *testing.T) {
	cfg := testConfig(t, map[string]string{
		"CHALLENGE_KEY":              validHexKey,
		"CHALLENGE_RATE_LIMIT_BURST": "1",
		"CHALLENGE_RATE_LIMIT_PER":   "1m",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	get := func() *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/challenge?locale=de", nil)
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		return rec
	}

	first := get()
	if first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d, body = %s", first.Code, http.StatusOK, first.Body)
	}

	second := get()
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d, body = %s", second.Code, http.StatusTooManyRequests, second.Body)
	}
	message := decodeErrorMessage(t, second.Body.Bytes())
	if message != contactMessagesDE.tooMany {
		t.Errorf("message = %q, want the localized tooMany message %q, not govue's English default", message, contactMessagesDE.tooMany)
	}
}

func TestAppContactUnavailableWithoutCredentials(t *testing.T) {
	// CHALLENGE_KEY is set so GET /api/v1/challenge itself succeeds and
	// returns a usable CSRF token below; SMTP credentials stay unset, which
	// is what still makes handleContact's own "not configured" check fire.
	cfg := testConfig(t, map[string]string{"CHALLENGE_KEY": validHexKey})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// requireSession now refuses a contact POST with no session outright,
	// and the CSRF protector requires a matching token for one that does;
	// neither of those is what this test means to exercise. A GET first,
	// carrying its cookie and CSRF token on the POST, gets past both so
	// the request actually reaches handleContact's own "not configured"
	// check, same as before requireSession existed.
	getReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/challenge", nil)
	getRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(getRec, getReq)
	cookies := getRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie set by the challenge request")
	}
	var challenge struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &challenge); err != nil {
		t.Fatalf("decode challenge: %v", err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/contact",
		strings.NewReader(`{"name":"Erika","email":"erika@example.test","message":"hi","consent":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(csrf.HeaderName, challenge.CSRFToken)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body)
	}
	// A status-only assertion here is exactly why this bug shipped: the
	// handler answered the right status with the wrong, English-collapsed
	// body. The message has to be the localized German text, which survives
	// only because handleContact marks this specific error Exposed.
	message := decodeErrorMessage(t, rec.Body.Bytes())
	if message != contactMessagesDE.unavailable {
		t.Errorf("message = %q, want the localized unavailable message %q", message, contactMessagesDE.unavailable)
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
	cfg := testConfig(t, map[string]string{
		"CHALLENGE_KEY":            validHexKey,
		"SMTP_USERNAME":            "no-reply@example.test",
		"SMTP_PASSWORD":            "s3cret",
		"CONTACT_RATE_LIMIT_BURST": "1",
		"CONTACT_RATE_LIMIT_PER":   "1m",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// requireSession now refuses a contact POST with no session, and the
	// CSRF protector requires a matching token for one that does. A GET
	// first, carrying its cookie and CSRF token on every POST below, is
	// what the honeypot path still needs to reach handleContact at all;
	// without it, this test would only be proving requireSession works,
	// not the honeypot.
	getReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/challenge", nil)
	getRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(getRec, getReq)
	cookies := getRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie set by the challenge request")
	}
	var challenge struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &challenge); err != nil {
		t.Fatalf("decode challenge: %v", err)
	}

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/contact", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(csrf.HeaderName, challenge.CSRFToken)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		return rec
	}

	honeypotBody := `{"name":"Eve","email":"eve@example.test","message":"buy now","consent":true,"website":"https://spam.example"}`

	// The burst is 1: a honeypot-filled submission must never consume it,
	// however many times a bot sends one, since the honeypot check runs
	// before the rate limit check in handleContact.
	for i := range 3 {
		rec := post(honeypotBody)
		if rec.Code != http.StatusOK {
			t.Fatalf("honeypot request %d status = %d, want %d, body = %s", i, rec.Code, http.StatusOK, rec.Body)
		}
	}

	// A genuine-looking submission (lacking the proof of work, so it fails
	// validation further down, but only after the budget check runs) proves
	// the limiter itself still works and was never touched by the honeypot
	// requests above: the first one still has the whole burst available.
	genuineBody := `{"name":"Erika","email":"erika@example.test","message":"hi","consent":true}`
	first := post(genuineBody)
	if first.Code == http.StatusTooManyRequests {
		t.Fatalf("first genuine submission already rate limited, body = %s", first.Body)
	}
	second := post(genuineBody)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second genuine submission status = %d, want %d (rate limited), body = %s",
			second.Code, http.StatusTooManyRequests, second.Body)
	}
}

// TestAppContactRefusedWithNoSessionAtAll proves requireSession, not the
// CSRF protector's origin/token check, is what refuses a contact POST that
// never went through GET /api/v1/challenge at all. Both checks answer 403,
// so this names the missing-session path specifically, distinct from
// TestAppContactRejectsMissingCSRFTokenOnceASessionExists (session exists,
// token missing) and TestAppContactUnavailableWithoutCredentials (reaches
// handleContact's own "not configured" check, after both of those pass).
func TestAppContactRefusedWithNoSessionAtAll(t *testing.T) {
	app := contactTestApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/contact",
		strings.NewReader(`{"name":"Erika","email":"erika@example.test","message":"hi","consent":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body)
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
	//
	// The error text is deliberately shaped like a real net/smtp dial
	// failure, carrying a hostname and credentials, so the leak assertion
	// below is checking something a real SMTP error could actually
	// contain, not a string this test invented for its own sake.
	const leakedHost = "mail.ostendorf.internal"
	const leakedCreds = "user=mailer-svc@ostendorf.cloud password=correct-horse-battery-staple"
	dialErr := errors.New("dial tcp " + leakedHost + ":587: smtp auth failed for " + leakedCreds)
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
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d (the mailer's dial failure), body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body)
	}
	// handleContact marks this error Exposed, but builds it fresh from the
	// localized sendFailed text alone; err itself is never handed to the
	// client-facing error, only logged above. The localized message
	// therefore survives, same as the two "not configured" cases.
	code, message := decodeError(t, rec.Body.Bytes())
	if code != api.CodeUnavailable {
		t.Errorf("code = %q, want %q", code, api.CodeUnavailable)
	}
	if message != contactMessagesDE.sendFailed {
		t.Errorf("message = %q, want the localized sendFailed message %q", message, contactMessagesDE.sendFailed)
	}

	// The real assertion: whatever the mailer's own error said must never
	// reach the response body. Checking the whole body against the fixed
	// German text alone would still pass if a future change appended the
	// cause after it, so this checks for the leak-prone substrings
	// specifically, not just body equality.
	full := rec.Body.String()
	for _, leaked := range []string{leakedHost, leakedCreds, dialErr.Error()} {
		if strings.Contains(full, leaked) {
			t.Errorf("response body leaked %q from the mailer's own error, body = %s", leaked, full)
		}
	}
}

// TestAppContactCrossOriginPostWithValidSessionAndTokenStillRefused proves
// the CSRF protector's origin check runs independently of the token check,
// on a request that would otherwise pass every other gate: a real session
// and its matching token, same as a genuine visitor's browser would send.
// Per the go-live review's finding 8, the origin half is this form's real
// defence once a session exists (the token half only ever fails open, by
// design, when there is nothing to forge); this test is what would fail if
// that defence were ever dropped from the chain or reordered behind
// something that short-circuits first.
func TestAppContactCrossOriginPostWithValidSessionAndTokenStillRefused(t *testing.T) {
	app := contactTestApp(t)

	getReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/challenge", nil)
	getRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(getRec, getReq)
	cookies := getRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie set by the challenge request")
	}
	var challenge struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &challenge); err != nil {
		t.Fatalf("decode challenge: %v", err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/contact",
		strings.NewReader(`{"name":"Erika","email":"erika@example.test","message":"hi","consent":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(csrf.HeaderName, challenge.CSRFToken)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	// The one header this test adds on top of an otherwise fully valid
	// request: a browser reports this whenever the request's initiator
	// is a different site, cross subdomains included.
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body)
	}
}

// TestAppClientIPWiringHonoursTrustedProxy proves TRUSTED_PROXY_CIDRS
// actually changes what the rate limiter keys on, not only that config.Load
// parses it (internal/config's own tests already cover that): a request
// relayed through a trusted proxy is attributed to the address it names in
// X-Forwarded-For, not the proxy's own TCP peer, so two distinct visitors
// behind the same proxy get two distinct buckets.
func TestAppClientIPWiringHonoursTrustedProxy(t *testing.T) {
	cfg := testConfig(t, map[string]string{
		"TRUSTED_PROXY_CIDRS": "203.0.113.0/24",
		"RATE_LIMIT_BURST":    "1",
		"RATE_LIMIT_PER":      "1m",
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
	if !strings.Contains(second.Body.String(), contactMessagesDE.tooMany) {
		t.Errorf("body = %q, want it to contain the localized tooMany message %q", second.Body.String(), contactMessagesDE.tooMany)
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
	dist := fstest.MapFS{"assets/app.js": &fstest.MapFile{Data: []byte("x")}}
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

// TestAppContactGeneralRateLimitAppliesBeforeTheHandler proves
// a.rateLimit.Middleware, the general per-client limiter every other route
// already spends from, now also guards POST /api/v1/contact: a request
// beyond its burst is refused with a 429 before it ever reaches
// requireSession, the CSRF protector or handleContact, none of which either
// request in this test satisfies (no session cookie, no CSRF token). The
// response is distinguished from contactLimiter's and mailLimiter's own
// 429s, which both carry the localized tooMany message (see
// TestAppContactHoneypotSilentlyAccepted and
// TestAppContactMailCeilingOnlySpentOnARealSendAttempt), by carrying govue's
// fixed English default instead, the same fallback
// TestAppAssetRateLimitStaysJSON already proves for a non-page path.
func TestAppContactGeneralRateLimitAppliesBeforeTheHandler(t *testing.T) {
	cfg := testConfig(t, map[string]string{
		"RATE_LIMIT_BURST": "1",
		"RATE_LIMIT_PER":   "1m",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	post := func() *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/contact",
			strings.NewReader(`{"name":"Erika","email":"erika@example.test","message":"hi","consent":true}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.90:1234"
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		return rec
	}

	// The first request still has to prove something other than 429: the
	// burst of 1 has not been spent yet, and requireSession refuses it for
	// carrying no session at all, same as TestAppContactRefusedWithNoSessionAtAll.
	first := post()
	if first.Code != http.StatusForbidden {
		t.Fatalf("first request status = %d, want %d (requireSession refusing a missing session), body = %s",
			first.Code, http.StatusForbidden, first.Body)
	}

	second := post()
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d (the general limiter's burst of 1, spent by the first request), body = %s",
			second.Code, http.StatusTooManyRequests, second.Body)
	}
	code, message := decodeError(t, second.Body.Bytes())
	if code != api.CodeTooManyRequests {
		t.Errorf("code = %q, want %q", code, api.CodeTooManyRequests)
	}
	const govueDefaultTooMany = "too many requests, please wait a moment"
	if message != govueDefaultTooMany {
		t.Errorf("message = %q, want govue's default %q, not a message coming from requireSession, the CSRF "+
			"protector or handleContact's own contactLimiter/mailLimiter, none of which this request ever reached",
			message, govueDefaultTooMany)
	}
}

// TestAppContactMailCeilingOnlySpentOnARealSendAttempt proves two things
// about the mail ceiling (finding 6), across three client addresses so the
// per-client contactLimiter (kept wide open here) can never be the reason
// for any of the three outcomes below:
//
//  1. A submission that never gets far enough to attempt a send (no proof of
//     work at all, so it fails at the PoW == nil check) must never spend the
//     ceiling's one token. If it did, the ceiling would protect nothing: an
//     attacker could drain the whole mail account's budget with requests
//     that could never have sent a mail in the first place, exactly the bug
//     this test is written against.
//  2. A second, fully valid submission from a different address must
//     therefore still find the token available, and reach the mailer (it
//     gets refused there instead, for want of a real SMTP server in this
//     test, but that refusal is a 503, never a 429).
//  3. A third, equally valid submission from yet another address must now be
//     refused by the ceiling: the second submission's real attempt to send
//     is what actually spent the one token burst=1 allows, so the ceiling
//     still does its job of bounding the mail account once a submission
//     earns its way that far.
func TestAppContactMailCeilingOnlySpentOnARealSendAttempt(t *testing.T) {
	cfg := testConfig(t, map[string]string{
		"CHALLENGE_KEY":            validHexKey,
		"SMTP_USERNAME":            "no-reply@example.test",
		"SMTP_PASSWORD":            "s3cret",
		"CONTACT_RATE_LIMIT_BURST": "100",
		"CONTACT_RATE_LIMIT_PER":   "1m",
		"MAIL_RATE_LIMIT_BURST":    "1",
		"MAIL_RATE_LIMIT_PER":      "1m",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	app.challenges.Now = func() time.Time { return now }

	// challenge fetches a real challenge for remoteAddr and returns it
	// alongside the session cookies and CSRF token a POST needs, the same
	// GET-then-POST pattern every other contact test in this file follows.
	challenge := func(remoteAddr string) (pow.Challenge, []*http.Cookie, string) {
		getReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/challenge", nil)
		getReq.RemoteAddr = remoteAddr
		getRec := httptest.NewRecorder()
		app.Handler().ServeHTTP(getRec, getReq)
		if getRec.Code != http.StatusOK {
			t.Fatalf("challenge for %s status = %d, want %d, body = %s", remoteAddr, getRec.Code, http.StatusOK, getRec.Body)
		}
		var body struct {
			pow.Challenge
			CSRFToken string `json:"csrfToken"`
		}
		if err := json.Unmarshal(getRec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode challenge: %v", err)
		}
		return body.Challenge, getRec.Result().Cookies(), body.CSRFToken
	}

	post := func(remoteAddr, body string, cookies []*http.Cookie, csrfToken string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/contact", strings.NewReader(body))
		req.RemoteAddr = remoteAddr
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(csrf.HeaderName, csrfToken)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		return rec
	}

	// 1. Address A: no pow field at all. This must be refused at validation,
	// before mailLimiter.Take ever runs, so it must never touch the ceiling.
	_, cookiesA, csrfA := challenge("203.0.113.61:1111")
	first := post("203.0.113.61:1111", `{"name":"Erika","email":"erika@example.test","message":"hi","consent":true}`, cookiesA, csrfA)
	if first.Code != http.StatusBadRequest {
		t.Fatalf("address A (missing pow) status = %d, want %d, body = %s", first.Code, http.StatusBadRequest, first.Body)
	}

	// 2. Address B: a real, solved challenge. If address A's worthless
	// submission above had wrongly spent the ceiling's one token, this
	// would come back 429; it must not, since nothing has actually tried to
	// send a mail yet.
	challengeB, cookiesB, csrfB := challenge("203.0.113.62:2222")
	solutionB := solveChallenge(t, challengeB)
	now = now.Add(5 * time.Second)
	bodyB, err := json.Marshal(map[string]any{
		"name": "Erika", "email": "erika@example.test", "message": "hi", "consent": true, "pow": solutionB,
	})
	if err != nil {
		t.Fatal(err)
	}
	second := post("203.0.113.62:2222", string(bodyB), cookiesB, csrfB)
	if second.Code == http.StatusTooManyRequests {
		t.Fatalf("address B (valid submission) status = %d, want anything but %d (the ceiling must still have its token), body = %s",
			second.Code, http.StatusTooManyRequests, second.Body)
	}

	// 3. Address C: another real, solved challenge. Address B's attempt
	// above is what actually spent the one token burst=1 allows, so this
	// one must now be refused by the ceiling.
	challengeC, cookiesC, csrfC := challenge("203.0.113.63:3333")
	solutionC := solveChallenge(t, challengeC)
	now = now.Add(5 * time.Second)
	bodyC, err := json.Marshal(map[string]any{
		"name": "Erika", "email": "erika@example.test", "message": "hi", "consent": true, "pow": solutionC,
	})
	if err != nil {
		t.Fatal(err)
	}
	third := post("203.0.113.63:3333", string(bodyC), cookiesC, csrfC)
	if third.Code != http.StatusTooManyRequests {
		t.Fatalf("address C (valid submission) status = %d, want %d (the ceiling spent by address B), body = %s",
			third.Code, http.StatusTooManyRequests, third.Body)
	}
}

// recordingHandler is a minimal slog.Handler fake that keeps every record's
// message, standing in for a real sink so a test can assert on how many
// times something was logged without parsing text output.
type recordingHandler struct {
	mu       sync.Mutex
	messages []string
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, r.Message)
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

// count returns how many recorded messages equal message exactly.
func (h *recordingHandler) count(message string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, m := range h.messages {
		if m == message {
			n++
		}
	}
	return n
}

// TestAppContactMailCeilingWarningLogsOnceAcrossARepeatedTrip proves
// warnMailCeiling's own job: tripping the mail ceiling repeatedly in quick
// succession, well inside one MailRateLimitPer window, still produces
// exactly one warning line, not one per refused request, which is what a
// sustained flood would otherwise turn the access log into.
func TestAppContactMailCeilingWarningLogsOnceAcrossARepeatedTrip(t *testing.T) {
	handler := &recordingHandler{}
	cfg := testConfig(t, map[string]string{
		"CHALLENGE_KEY":         validHexKey,
		"SMTP_USERNAME":         "no-reply@example.test",
		"SMTP_PASSWORD":         "s3cret",
		"MAIL_RATE_LIMIT_BURST": "1",
		"MAIL_RATE_LIMIT_PER":   "1m",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, slog.New(handler), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	app.challenges.Now = func() time.Time { return now }

	submit := func() *httptest.ResponseRecorder {
		getReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/challenge", nil)
		getRec := httptest.NewRecorder()
		app.Handler().ServeHTTP(getRec, getReq)
		var challenge struct {
			pow.Challenge
			CSRFToken string `json:"csrfToken"`
		}
		if err := json.Unmarshal(getRec.Body.Bytes(), &challenge); err != nil {
			t.Fatalf("decode challenge: %v", err)
		}
		solution := solveChallenge(t, challenge.Challenge)
		now = now.Add(5 * time.Second)

		body, err := json.Marshal(map[string]any{
			"name": "Erika", "email": "erika@example.test", "message": "hi", "consent": true, "pow": solution,
		})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/contact", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(csrf.HeaderName, challenge.CSRFToken)
		for _, c := range getRec.Result().Cookies() {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		return rec
	}

	first := submit()
	if first.Code == http.StatusTooManyRequests {
		t.Fatalf("first submission already refused by the ceiling, body = %s", first.Body)
	}
	for i := range 2 {
		rec := submit()
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("refusal %d status = %d, want %d (the ceiling, spent by the first submission), body = %s",
				i, rec.Code, http.StatusTooManyRequests, rec.Body)
		}
	}

	const warning = "contact form: mail ceiling reached, refusing submissions"
	if n := handler.count(warning); n != 1 {
		t.Errorf("warning logged %d times across three trips of the ceiling, want exactly 1", n)
	}
}

// mustDecodeHex is validHexKey's own encoding, decoded for a test that needs
// the raw key bytes pow.Service.Create takes rather than the hex string
// config.Load reads.
func mustDecodeHex(t *testing.T, raw string) []byte {
	t.Helper()
	key, err := hex.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode hex key: %v", err)
	}
	return key
}

// TestAppContactAcceptsSolutionSignedWithThePreviousChallengeKey proves
// CHALLENGE_KEY rotation (AGENTS.md's deferred item): a solution signed
// with CHALLENGE_KEY_PREVIOUS, exactly what a visitor's browser is holding
// the moment CHALLENGE_KEY is rotated out from under them mid-visit, still
// verifies, while a fresh challenge is still issued under the new,
// current key.
func TestAppContactAcceptsSolutionSignedWithThePreviousChallengeKey(t *testing.T) {
	previousKeyHex := strings.Repeat("cd", 32)
	cfg := testConfig(t, map[string]string{
		"CHALLENGE_KEY":          validHexKey,
		"CHALLENGE_KEY_PREVIOUS": previousKeyHex,
		"SMTP_USERNAME":          "no-reply@example.test",
		"SMTP_PASSWORD":          "s3cret",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	app.challenges.Now = func() time.Time { return now }

	// A challenge signed with the previous key directly: a.handleChallenge
	// always signs with the current key, so this stands in for one issued
	// before the rotation happened.
	oldChallenge, err := app.challenges.Create(mustDecodeHex(t, previousKeyHex))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	solution := solveChallenge(t, oldChallenge)
	now = now.Add(5 * time.Second)

	// A session and its CSRF token still come from the live endpoint; only
	// the proof of work solution above is deliberately stale.
	getReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/challenge", nil)
	getRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(getRec, getReq)
	cookies := getRec.Result().Cookies()
	var challenge struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &challenge); err != nil {
		t.Fatalf("decode challenge: %v", err)
	}

	body, err := json.Marshal(map[string]any{
		"name": "Erika", "email": "erika@example.test", "message": "hi", "consent": true, "pow": solution,
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
	// The mailer itself is not wired up to a real server in this test, so
	// success here means the real SMTP dial failing, not a submission
	// that was ever rejected for its proof of work: anything other than a
	// poWFailed/poWExpired/poWTooFast 400 proves the old key verified.
	if rec.Code == http.StatusBadRequest {
		t.Fatalf("status = %d, want anything but a validation rejection (the old-key solution must verify), body = %s",
			rec.Code, rec.Body)
	}
}

func TestAppCountsNothingWithoutUmami(t *testing.T) {
	app := testApp(t)
	if app.analyticsClient != nil {
		t.Fatal("an analytics client was built without UMAMI_URL")
	}
	// trackPageView is still called by every route handler; with no
	// client it has to be a no-op rather than a nil dereference.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/", nil)
	req.RemoteAddr = "203.0.113.1:1234"
	app.trackPageView(req, "Simon Ostendorf")
}

func TestAppServesPagesWithUmamiConfigured(t *testing.T) {
	// With Umami configured, a page view is queued for the background
	// worker. Nothing about that may change what the visitor gets: the
	// page still answers 200 with its own body, and it answers whether or
	// not anything is listening at the Umami end, because nothing here
	// waits for it. No worker is started in this test, so the queue is
	// never drained, which is the slowest case there is.
	cfg := testConfig(t, map[string]string{
		"UMAMI_URL":        "http://umami.invalid",
		"UMAMI_WEBSITE_ID": "0f1e2d3c-4b5a-4968-8776-655443322110",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if app.analyticsClient == nil {
		t.Fatal("no analytics client built for a configured Umami")
	}

	for range 3 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/", nil)
		req.RemoteAddr = "203.0.113.1:1234"
		req.Header.Set("User-Agent", "Mozilla/5.0 TestBrowser/1.0")
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Body.String(), "Startseite") {
			t.Errorf("body = %q, want the prerendered page", rec.Body.String())
		}
		// No script, no cookie, no beacon: the privacy policy's claim
		// that Umami never runs in the visitor's browser is a property of
		// the response, so it is checked on the response.
		if cookies := rec.Result().Cookies(); len(cookies) != 0 {
			t.Errorf("page set cookies %v, want none", cookies)
		}
		if strings.Contains(strings.ToLower(rec.Body.String()), "umami") {
			t.Errorf("body mentions umami, want nothing of it in the browser: %q", rec.Body.String())
		}
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
