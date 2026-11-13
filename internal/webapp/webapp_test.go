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
