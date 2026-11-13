package webapp

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/ostendorf-cloud/govue/api"
	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/csrf"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/contact"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/pow"
)

// Proof of work parameters, ported from old_simon-ostendorf.de's
// cmd/website/main.go: generous enough that solving never takes more than a
// second or two on a phone, tight enough that brute forcing many challenges
// in parallel is not an attractive way to flood the form.
const (
	challengeMaxNumber = 300_000
	challengeTTL       = 10 * time.Minute
	challengeMinAge    = 3 * time.Second
)

// maxContactBytes bounds a contact submission: 5000 characters of message,
// up to four bytes each, and the proof of work solution fit easily within it.
const maxContactBytes = 64 << 10

// sendTimeout bounds the whole SMTP conversation for one submission.
const sendTimeout = 25 * time.Second

// challengeResponse is what GET /api/v1/challenge answers: the proof of
// work challenge, plus the CSRF token of the session ensureSession made
// sure exists. Carrying both in one response is what lets the contact
// form's single round trip, fired on focus before a visitor has typed
// anything, cover everything the eventual submission needs from the
// server ahead of time.
type challengeResponse struct {
	pow.Challenge
	CSRFToken string `json:"csrfToken"`
}

// contactSubmission is the JSON body POST /api/v1/contact accepts. Locale
// is the page's own active locale, not the Accept-Language header: a
// visitor who switched the page to English expects an English answer even
// on a browser still set to German.
type contactSubmission struct {
	Name    string        `json:"name"`
	Email   string        `json:"email"`
	Message string        `json:"message"`
	Consent bool          `json:"consent"`
	Website string        `json:"website"`
	Locale  string        `json:"locale"`
	PoW     *pow.Solution `json:"pow"`
}

// contactResponse is what a successful POST /api/v1/contact answers.
type contactResponse struct {
	Message string `json:"message"`
}

// requestLocale narrows raw to a locale this site serves, German otherwise:
// the same fallback preferredLocale applies to a missing translation.
func requestLocale(raw string) string {
	if slices.Contains(locales, raw) {
		return raw
	}
	return defaultLocale
}

// challengeRateLimitRespond answers a denied GET /api/v1/challenge request
// in the visitor's own locale, read the same way handleChallenge reads it,
// instead of govue's fixed English default: this route is fetched by the
// page itself, so its 429 is as much a user facing message as any other
// answer it gives.
func (a *App) challengeRateLimitRespond(w http.ResponseWriter, r *http.Request, wait time.Duration) {
	locale := requestLocale(r.URL.Query().Get("locale"))
	messages := messagesFor(locale)
	err := api.New(http.StatusTooManyRequests, api.CodeTooManyRequests, messages.tooMany)
	err.RetryAfter = wait
	api.WriteError(w, r, err)
}

// pageRateLimitRespond answers a denied page, asset, robots.txt or
// sitemap.xml request. A page request resolves to a known locale through
// splitLocale, the same way pageHandler itself does, and gets back a small
// HTML document carrying the localized tooMany message: a browser loading
// a page expects HTML, not govue's JSON error envelope, which is what it
// would otherwise see on a 429 here. A request that resolves to no locale,
// meaning an asset, robots.txt, sitemap.xml or an unknown path, keeps
// govue's default JSON shape, the same thing every other error on those
// paths already answers with.
func (a *App) pageRateLimitRespond(w http.ResponseWriter, r *http.Request, wait time.Duration) {
	locale, _, ok := splitLocale(r.URL.Path)
	if !ok && r.URL.Path == "/" {
		locale, ok = preferredLocale(r.Header.Get("Accept-Language")), true
	}
	if !ok {
		api.WriteError(w, r, api.TooManyRequests(wait))
		return
	}

	messages := messagesFor(locale)
	// The same floor api.WriteError applies to a RetryAfter: a wait under
	// one second would otherwise round down to Retry-After: 0, which asks
	// the browser to retry immediately.
	seconds := int(wait.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Retry-After", strconv.Itoa(seconds))
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(w, "<!doctype html><html lang=%q><meta charset=\"utf-8\"><title>%s</title><p>%s</p>",
		locale, messages.tooMany, messages.tooMany)
}

// handleChallenge issues a proof of work challenge for the session
// ensureSession resolved. The rate limit and the session both come from
// the route chain; this handler only has to mind the challenge key.
func (a *App) handleChallenge(w http.ResponseWriter, r *http.Request) error {
	locale := requestLocale(r.URL.Query().Get("locale"))
	messages := messagesFor(locale)

	if len(a.challengeKeys) == 0 {
		// 503 is the honest status: no challenge key is configured, so the
		// contact form really is unavailable, a dependency this deployment
		// has not supplied yet. Exposed is safe here: messages.unavailable
		// is a fixed, localized string with no interpolation anywhere in
		// it, so there is nothing for it to leak.
		return api.Unavailable(messages.unavailable).Exposed()
	}
	challenge, err := a.challenges.Create(a.challengeKeys[0])
	if err != nil {
		return api.Internal(err)
	}

	token := ""
	if s, ok := sessionFromContext(r.Context()); ok {
		token = csrf.Token(a.csrfKey, s.ID)
	}

	return api.Write(w, http.StatusOK, challengeResponse{Challenge: challenge, CSRFToken: token})
}

// handleContact validates, checks the honeypot and the proof of work, and
// mails a contact submission. It logs outcomes only, never the name,
// address or message.
func (a *App) handleContact(w http.ResponseWriter, r *http.Request) error {
	var body contactSubmission
	if err := api.DecodeLimit(r, &body, maxContactBytes); err != nil {
		return err
	}
	locale := requestLocale(body.Locale)
	messages := messagesFor(locale)

	if !a.smtpConfigured || len(a.challengeKeys) == 0 {
		// 503 is the honest status, exactly as in handleChallenge: SMTP
		// credentials or a challenge key are missing, a dependency this
		// deployment has not supplied yet, not anything the caller got
		// wrong. Exposed is safe here for the same reason: messages.unavailable
		// is the same fixed, localized string with no interpolation, so
		// there is nothing for it to leak.
		return api.Unavailable(messages.unavailable).Exposed()
	}

	req := contact.Request{Name: body.Name, Email: body.Email, Message: body.Message, Consent: body.Consent, Honeypot: body.Website}
	req.Normalize()
	if req.IsSpam() {
		// Bots get the same answer as people, so they learn nothing.
		a.logger.InfoContext(r.Context(), "contact form: honeypot filled, dropped")
		return api.Write(w, http.StatusOK, contactResponse{Message: messages.sent})
	}

	// Only after the honeypot: bots filling it must not use up the budget of
	// people behind the same address, nor the site-wide mail ceiling below.
	if allowed, wait := a.contactLimiter.Take(clientip.Key(a.clientIPs.Resolve(r))); !allowed {
		err := api.New(http.StatusTooManyRequests, api.CodeTooManyRequests, messages.tooMany)
		err.RetryAfter = wait
		return err
	}

	if fe := req.Validate(); fe != nil {
		message := fieldMessage(locale, fe.Field, fe.Code)
		return api.Invalid(message).WithDetails(map[string]any{fe.Field: message})
	}

	if body.PoW == nil {
		return api.Invalid(messages.noPoW)
	}
	if err := a.challenges.Verify(a.challengeKeys, *body.PoW); err != nil {
		a.logger.InfoContext(r.Context(), "contact form: proof of work rejected", "reason", err.Error())
		switch {
		case errors.Is(err, pow.ErrExpired):
			return api.Invalid(messages.poWExpired)
		case errors.Is(err, pow.ErrTooFast):
			return api.Invalid(messages.poWTooFast)
		case errors.Is(err, pow.ErrBusy):
			return api.New(http.StatusTooManyRequests, api.CodeTooManyRequests, messages.tooMany)
		default:
			return api.Invalid(messages.poWFailed)
		}
	}

	// mailLimiter is checked only now, once every earlier gate (honeypot,
	// validation, proof of work) has already been cleared: it is the hard
	// ceiling on the mail account itself, independent of how many client
	// addresses an attacker spreads their requests across (see mailLimiter's
	// own comment in webapp.go), and a request that could never have reached
	// a send attempt anyway must never spend a share of that ceiling.
	// Checking it any earlier, as this used to, let a flood of invalid or
	// unsolved submissions drain the whole site's mail budget without ever
	// coming close to sending a mail.
	//
	// The token taken here is never given back, even when contact.Build or
	// a.mailer.Send below goes on to fail: govue's ratelimit.Limiter offers
	// no way to return one, only Take, and adding one would mean changing
	// govue itself, out of scope for this service. A refund would also
	// reopen exactly the oracle this ceiling exists to close, since anyone
	// who found a reliable way to make Send fail after a valid proof of
	// work would then get unlimited free retries against it. A failed Send
	// still means an SMTP conversation with the mail submission server was
	// actually opened, which is worth bounding on its own, whether or not
	// the message was ultimately accepted.
	//
	// Tripping it is logged, through warnMailCeiling rather than directly:
	// a single address hitting its own contactLimiter above is ordinary and
	// expected, but every contact mail this service can ever send running
	// out in one window means either an attack or a real surge in genuine
	// enquiries, either of which the operator needs to know about, just not
	// once per refused request under a sustained flood.
	if allowed, wait := a.mailLimiter.Take(mailLimiterKey); !allowed {
		a.warnMailCeiling(r.Context(), wait)
		err := api.New(http.StatusTooManyRequests, api.CodeTooManyRequests, messages.tooMany)
		err.RetryAfter = wait
		return err
	}

	msg, err := contact.Build(a.envelope, req, time.Now(), rand.Reader)
	if err != nil {
		return api.Internal(err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), sendTimeout)
	defer cancel()
	if err := a.mailer.Send(ctx, a.smtpCreds, a.envelope.From.Address, a.envelope.To.Address, msg); err != nil {
		// The real cause is logged here, with everything the SMTP error
		// carries, and nowhere else.
		a.logger.ErrorContext(r.Context(), "contact form: send mail", "error", err.Error())
		// 503 is the honest status: the mail submission server is the
		// dependency that failed, not anything the caller got wrong. The
		// rule is not "never Exposed an error that could carry a cause", it
		// is "never let a cause reach the client" — so the fix is to never
		// hand the client an error that has one, not to avoid Exposed on an
		// error that does. api.Unavailable(messages.sendFailed) below is
		// built fresh from the fixed, localized text alone; err is never
		// passed to it and WithCause is never called on it, so there is
		// nothing for Exposed to leak. A derived error — this one made by
		// taking the err-carrying error and stripping WithCause back off —
		// would depend on that stripping happening correctly forever; a
		// fresh one can't regress that way even if this function grows more
		// error paths later.
		return api.Unavailable(messages.sendFailed).Exposed()
	}
	a.logger.InfoContext(r.Context(), "contact form: mail sent")
	return api.Write(w, http.StatusOK, contactResponse{Message: messages.sent})
}
