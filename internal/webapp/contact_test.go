package webapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ostendorf-cloud/govue/contact"
	"github.com/ostendorf-cloud/govue/mail"
	"github.com/ostendorf-cloud/govue/pow"
)

// fakeSender is the contact form's mail.Sender in these tests. It opens no
// connection and keeps every message in memory.
type fakeSender struct {
	mu   sync.Mutex
	sent []mail.Message
}

func (f *fakeSender) Send(_ context.Context, m mail.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return nil
}

func (f *fakeSender) messages() []mail.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]mail.Message(nil), f.sent...)
}

// testClock is a clock a test moves past the proof of work's minimum age
// instead of sleeping. govue reads it from several goroutines.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// contactApp builds the App with a challenge key, the fake sender and the
// test clock, plus whatever env a test adds.
func contactApp(t *testing.T, env map[string]string) (*App, *fakeSender, *testClock) {
	t.Helper()
	merged := map[string]string{"CHALLENGE_KEY": validHexKey}
	for k, v := range env {
		merged[k] = v
	}
	sender := &fakeSender{}
	clock := &testClock{now: time.Now()}
	dist, prerendered := testFS()
	app, err := New(testConfig(t, merged), discardLogger(), dist, prerendered,
		WithMailSender(sender), WithClock(clock.Now))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return app, sender, clock
}

// fetchChallenge asks for a challenge through the public handler and solves
// it the way the browser worker does.
func fetchChallenge(t *testing.T, app *App, remoteAddr string) (pow.Solution, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, contactChallengePath, nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("challenge status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}
	var c pow.Challenge
	if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
		t.Fatalf("decode challenge: %v", err)
	}
	for n := int64(0); n <= c.MaxNumber; n++ {
		if pow.Hash(c.Salt, n) == c.Challenge {
			return pow.Solution{Algorithm: c.Algorithm, Challenge: c.Challenge, Number: n, Salt: c.Salt, Signature: c.Signature}, rec
		}
	}
	t.Fatal("no number solves the challenge")
	return pow.Solution{}, nil
}

// submission is the body GovueContactForm posts: no consent field, it stays
// in the browser.
func submission(t *testing.T, sol *pow.Solution) string {
	t.Helper()
	body := map[string]any{
		"name":    "Erika Mustermann",
		"email":   "erika@example.test",
		"message": "Hallo,\nich brauche Technik.",
		"website": "",
	}
	if sol != nil {
		body["pow"] = sol
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// postContact posts body as a same origin browser request, changed by mod.
func postContact(t *testing.T, app *App, remoteAddr, body string, mod func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, contactPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.RemoteAddr = remoteAddr
	if mod != nil {
		mod(req)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	return rec
}

// TestContactSendsTheSiteMailThroughTheRealChain proves the site mounts
// govue's handlers with its own envelope, subject and body: one valid
// challenge and submission sends exactly one mail, From and To are the
// site's, the visitor is only in Reply-To, and the body is the site's text
// without the consent timestamp the site's own builder used to add.
func TestContactSendsTheSiteMailThroughTheRealChain(t *testing.T) {
	app, sender, clock := contactApp(t, nil)
	const addr = "198.51.100.10:1234"

	sol, _ := fetchChallenge(t, app, addr)
	clock.Advance(5 * time.Second)
	rec := postContact(t, app, addr, submission(t, &sol), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}

	sent := sender.messages()
	if len(sent) != 1 {
		t.Fatalf("sent %d mails, want 1", len(sent))
	}
	m := sent[0]
	if m.From != `"ostendorf.tech" <no-reply@ostendorf.cloud>` {
		t.Errorf("From = %q, want the site's CONTACT_FROM default", m.From)
	}
	if len(m.To) != 1 || m.To[0] != "<simon@ostendorf.tech>" {
		t.Errorf("To = %q, want the site's CONTACT_TO default", m.To)
	}
	if m.ReplyTo != "erika@example.test" {
		t.Errorf("ReplyTo = %q, want the visitor's address", m.ReplyTo)
	}
	if m.Subject != "Kontaktanfrage von Erika Mustermann" {
		t.Errorf("Subject = %q, want the site's subject", m.Subject)
	}
	const wantBody = "Neue Kontaktanfrage über das Formular auf ostendorf.tech\n\n" +
		"Name: Erika Mustermann\n" +
		"E-Mail: erika@example.test\n\n" +
		"Nachricht:\nHallo,\nich brauche Technik.\n"
	if m.Body != wantBody {
		t.Errorf("Body = %q, want %q", m.Body, wantBody)
	}
	for _, gone := range []string{"Datenschutz", "Einverständnis", "erteilt am", "Uhr"} {
		if strings.Contains(m.Body, gone) {
			t.Errorf("Body contains %q, the consent timestamp is dropped", gone)
		}
	}
}

// TestContactSubjectShortensALongName pins the subject rule the site kept:
// at most 80 runes of the name, the body has the full name.
func TestContactSubjectShortensALongName(t *testing.T) {
	name := strings.Repeat("ä", 100)
	got := contactSubject(contact.Request{Name: name})
	if want := "Kontaktanfrage von " + strings.Repeat("ä", 80) + "…"; got != want {
		t.Errorf("subject = %q, want %q", got, want)
	}
	if !strings.Contains(contactBody(contact.Request{Name: name}), name) {
		t.Error("body lacks the full name")
	}
}

// TestContactRefusesAForeignOrigin proves govue's origin check runs on the
// mounted route: a cross site browser POST with a valid solution is refused
// with 403 cross_origin and sends nothing.
func TestContactRefusesAForeignOrigin(t *testing.T) {
	app, sender, clock := contactApp(t, nil)
	const addr = "198.51.100.11:1234"

	sol, _ := fetchChallenge(t, app, addr)
	clock.Advance(5 * time.Second)
	rec := postContact(t, app, addr, submission(t, &sol), func(r *http.Request) {
		r.Header.Set("Sec-Fetch-Site", "cross-site")
		r.Header.Set("Origin", "https://evil.example")
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body)
	}
	if code := errorCode(t, rec.Body.Bytes()); code != contact.CodeCrossOrigin {
		t.Errorf("code = %q, want %q", code, contact.CodeCrossOrigin)
	}
	if n := len(sender.messages()); n != 0 {
		t.Errorf("sent %d mails, want none", n)
	}
}

// TestContactTrustsTheSiteURLOrigin proves the origin SITE_URL names is
// trusted even when the Host header the server sees differs, as behind a
// proxy that rewrites it.
func TestContactTrustsTheSiteURLOrigin(t *testing.T) {
	app, sender, clock := contactApp(t, nil)
	const addr = "198.51.100.12:1234"

	sol, _ := fetchChallenge(t, app, addr)
	clock.Advance(5 * time.Second)
	rec := postContact(t, app, addr, submission(t, &sol), func(r *http.Request) {
		r.Header.Del("Sec-Fetch-Site")
		r.Host = "website.internal:8080"
		r.Header.Set("Origin", "https://ostendorf.tech")
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}
	if n := len(sender.messages()); n != 1 {
		t.Errorf("sent %d mails, want 1", n)
	}
}

// TestContactSetsNoCookie proves the contact routes are stateless: neither
// the challenge nor the submission sets a cookie.
func TestContactSetsNoCookie(t *testing.T) {
	app, _, clock := contactApp(t, nil)
	const addr = "198.51.100.13:1234"

	sol, challenge := fetchChallenge(t, app, addr)
	if got := challenge.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Errorf("challenge Set-Cookie = %q, want none", got)
	}
	clock.Advance(5 * time.Second)
	rec := postContact(t, app, addr, submission(t, &sol), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}
	if got := rec.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Errorf("submit Set-Cookie = %q, want none", got)
	}
}

// TestContactNotConfiguredWithoutKeyOrSMTP proves the site starts without a
// challenge key and without SMTP_HOST, and both routes answer 503
// not_configured.
func TestContactNotConfiguredWithoutKeyOrSMTP(t *testing.T) {
	app := testApp(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, contactChallengePath, nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("challenge status = %d, want %d, body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body)
	}
	if code := errorCode(t, rec.Body.Bytes()); code != contact.CodeNotConfigured {
		t.Errorf("challenge code = %q, want %q", code, contact.CodeNotConfigured)
	}

	post := postContact(t, app, "198.51.100.14:1234", submission(t, nil), nil)
	if post.Code != http.StatusServiceUnavailable {
		t.Fatalf("submit status = %d, want %d, body = %s", post.Code, http.StatusServiceUnavailable, post.Body)
	}
	if code := errorCode(t, post.Body.Bytes()); code != contact.CodeNotConfigured {
		t.Errorf("submit code = %q, want %q", code, contact.CodeNotConfigured)
	}
}

// TestContactNotConfiguredWithoutSMTP proves a key alone does not make the
// form send: without SMTP_HOST and without a sender, the submission answers
// 503 not_configured.
func TestContactNotConfiguredWithoutSMTP(t *testing.T) {
	dist, prerendered := testFS()
	app, err := New(testConfig(t, map[string]string{"CHALLENGE_KEY": validHexKey}), discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if app.contact.Configured() {
		t.Fatal("contact form configured without SMTP_HOST")
	}
	rec := postContact(t, app, "198.51.100.15:1234", submission(t, nil), nil)
	if code := errorCode(t, rec.Body.Bytes()); rec.Code != http.StatusServiceUnavailable || code != contact.CodeNotConfigured {
		t.Fatalf("submit = %d %q, want %d %q", rec.Code, code, http.StatusServiceUnavailable, contact.CodeNotConfigured)
	}
}

// TestContactSMTPHostConfiguresTheForm proves SMTP_HOST plus the key is
// what makes the form configured with the real SMTP sender.
func TestContactSMTPHostConfiguresTheForm(t *testing.T) {
	dist, prerendered := testFS()
	app, err := New(testConfig(t, map[string]string{
		"CHALLENGE_KEY": validHexKey,
		"SMTP_HOST":     "mail.example.test",
		"SMTP_USERNAME": "no-reply@example.test",
		"SMTP_PASSWORD": "s3cret",
	}), discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if !app.contact.Configured() {
		t.Fatal("contact form not configured with SMTP_HOST and CHALLENGE_KEY")
	}
}

// TestContactClientLimitAnswers429WithRetryAfter proves CONTACT_RATE_LIMIT_*
// reaches govue's per client limiter: the second submission of one address
// is refused with 429 and a Retry-After, and sends nothing.
func TestContactClientLimitAnswers429WithRetryAfter(t *testing.T) {
	app, sender, _ := contactApp(t, map[string]string{
		"CONTACT_RATE_LIMIT_BURST": "1",
		"CONTACT_RATE_LIMIT_PER":   "1h",
	})
	const addr = "198.51.100.16:1234"

	// Without a solution the first submission is refused after the client
	// limit, so it spends the one token.
	if rec := postContact(t, app, addr, submission(t, nil), nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("first status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body)
	}
	rec := postContact(t, app, addr, submission(t, nil), nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d, body = %s", rec.Code, http.StatusTooManyRequests, rec.Body)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("429 carries no Retry-After")
	}
	if n := len(sender.messages()); n != 0 {
		t.Errorf("sent %d mails, want none", n)
	}
}

// TestContactChallengeLimitAnswers429WithRetryAfter proves
// CHALLENGE_RATE_LIMIT_* reaches govue's challenge limiter.
func TestContactChallengeLimitAnswers429WithRetryAfter(t *testing.T) {
	app, _, _ := contactApp(t, map[string]string{
		"CHALLENGE_RATE_LIMIT_BURST": "1",
		"CHALLENGE_RATE_LIMIT_PER":   "1h",
	})
	const addr = "198.51.100.17:1234"

	fetchChallenge(t, app, addr)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, contactChallengePath, nil)
	req.RemoteAddr = addr
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second challenge status = %d, want %d, body = %s", rec.Code, http.StatusTooManyRequests, rec.Body)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("429 carries no Retry-After")
	}
}

// TestContactGeneralRateLimitAppliesInFront proves the site's general per
// client limit (RATE_LIMIT_*) sits in front of the contact routes, as on
// every other route.
func TestContactGeneralRateLimitAppliesInFront(t *testing.T) {
	app, _, _ := contactApp(t, map[string]string{
		"RATE_LIMIT_BURST": "1",
		"RATE_LIMIT_PER":   "1h",
	})
	const addr = "198.51.100.18:1234"

	fetchChallenge(t, app, addr)
	rec := postContact(t, app, addr, submission(t, nil), nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusTooManyRequests, rec.Body)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("429 carries no Retry-After")
	}
}

// TestContactMailCeilingIsTheSiteValue proves MAIL_RATE_LIMIT_* reaches
// govue's global mail ceiling: with a ceiling of one, a second valid
// submission from another address is refused with 503 mail_limit and a
// Retry-After, and only the first mail is sent.
func TestContactMailCeilingIsTheSiteValue(t *testing.T) {
	app, sender, clock := contactApp(t, map[string]string{
		"MAIL_RATE_LIMIT_BURST": "1",
		"MAIL_RATE_LIMIT_PER":   "1h",
	})

	first, _ := fetchChallenge(t, app, "198.51.100.19:1234")
	second, _ := fetchChallenge(t, app, "198.51.100.20:1234")
	clock.Advance(5 * time.Second)
	if rec := postContact(t, app, "198.51.100.19:1234", submission(t, &first), nil); rec.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}
	rec := postContact(t, app, "198.51.100.20:1234", submission(t, &second), nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("second status = %d, want %d, body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body)
	}
	if code := errorCode(t, rec.Body.Bytes()); code != contact.CodeMailLimit {
		t.Errorf("code = %q, want %q", code, contact.CodeMailLimit)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("mail_limit carries no Retry-After")
	}
	if n := len(sender.messages()); n != 1 {
		t.Errorf("sent %d mails, want 1", n)
	}
}

// TestContactAcceptsASolutionSignedWithThePreviousKey proves the rotation
// still works through govue's key list: a challenge issued under the old
// key verifies after CHALLENGE_KEY moved on while CHALLENGE_KEY_PREVIOUS
// names the old one.
func TestContactAcceptsASolutionSignedWithThePreviousKey(t *testing.T) {
	previous := strings.Repeat("cd", 32)
	old, _, clock := contactApp(t, map[string]string{"CHALLENGE_KEY": previous})
	const addr = "198.51.100.21:1234"
	sol, _ := fetchChallenge(t, old, addr)

	sender := &fakeSender{}
	dist, prerendered := testFS()
	rotated, err := New(testConfig(t, map[string]string{
		"CHALLENGE_KEY":          validHexKey,
		"CHALLENGE_KEY_PREVIOUS": previous,
	}), discardLogger(), dist, prerendered, WithMailSender(sender), WithClock(clock.Now))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	clock.Advance(5 * time.Second)
	if rec := postContact(t, rotated, addr, submission(t, &sol), nil); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body)
	}
	if n := len(sender.messages()); n != 1 {
		t.Errorf("sent %d mails, want 1", n)
	}

	// Without the previous key the same kind of solution is refused.
	sol2, _ := fetchChallenge(t, old, "198.51.100.22:1234")
	dropped, err := New(testConfig(t, map[string]string{"CHALLENGE_KEY": validHexKey}),
		discardLogger(), dist, prerendered, WithMailSender(&fakeSender{}), WithClock(clock.Now))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	clock.Advance(5 * time.Second)
	rec := postContact(t, dropped, "198.51.100.22:1234", submission(t, &sol2), nil)
	if code := errorCode(t, rec.Body.Bytes()); code != contact.CodePowInvalid {
		t.Errorf("code without the previous key = %q, want %q", code, contact.CodePowInvalid)
	}
}

// TestContactChallengeAudienceIsTheSiteHost proves the challenge is bound to
// the site's host name: a solution of a site with another SITE_URL, signed
// with the same key, is refused.
func TestContactChallengeAudienceIsTheSiteHost(t *testing.T) {
	other, _, clock := contactApp(t, map[string]string{"SITE_URL": "https://other.example"})
	sol, _ := fetchChallenge(t, other, "198.51.100.23:1234")

	sender := &fakeSender{}
	dist, prerendered := testFS()
	app, err := New(testConfig(t, map[string]string{"CHALLENGE_KEY": validHexKey}),
		discardLogger(), dist, prerendered, WithMailSender(sender), WithClock(clock.Now))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	clock.Advance(5 * time.Second)
	rec := postContact(t, app, "198.51.100.23:1234", submission(t, &sol), nil)
	if code := errorCode(t, rec.Body.Bytes()); code != contact.CodePowInvalid {
		t.Errorf("code = %q, want %q", code, contact.CodePowInvalid)
	}
	if n := len(sender.messages()); n != 0 {
		t.Errorf("sent %d mails, want none", n)
	}
}
