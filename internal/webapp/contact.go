package webapp

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"slices"
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

// handleChallenge issues a proof of work challenge for the session
// ensureSession resolved. The rate limit and the session both come from
// the route chain; this handler only has to mind the challenge key.
func (a *App) handleChallenge(w http.ResponseWriter, r *http.Request) error {
	locale := requestLocale(r.URL.Query().Get("locale"))
	messages := messagesFor(locale)

	if len(a.challengeKeys) == 0 {
		// Below 500 on purpose: api.WriteError collapses any 5xx to a fixed
		// "internal error", which would swallow this localized message
		// before it ever reached the visitor. The Code stays unavailable;
		// only the status moves.
		return api.New(http.StatusConflict, api.CodeUnavailable, messages.unavailable)
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
		// Below 500 on purpose: api.WriteError collapses any 5xx to a fixed
		// "internal error", which would swallow this localized message
		// before it ever reached the visitor. The Code stays unavailable;
		// only the status moves.
		return api.New(http.StatusConflict, api.CodeUnavailable, messages.unavailable)
	}

	req := contact.Request{Name: body.Name, Email: body.Email, Message: body.Message, Consent: body.Consent, Honeypot: body.Website}
	req.Normalize()
	if req.IsSpam() {
		// Bots get the same answer as people, so they learn nothing.
		a.logger.InfoContext(r.Context(), "contact form: honeypot filled, dropped")
		return api.Write(w, http.StatusOK, contactResponse{Message: messages.sent})
	}

	// Only after the honeypot: bots filling it must not use up the budget of
	// people behind the same address.
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

	msg, err := contact.Build(a.envelope, req, time.Now(), rand.Reader)
	if err != nil {
		return api.Internal(err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), sendTimeout)
	defer cancel()
	if err := a.mailer.Send(ctx, a.smtpCreds, a.envelope.From.Address, a.envelope.To.Address, msg); err != nil {
		a.logger.ErrorContext(r.Context(), "contact form: send mail", "error", err.Error())
		// Below 500 on purpose: api.WriteError collapses any 5xx to a fixed
		// "internal error", which would swallow this localized message
		// before it ever reached the visitor. The Code stays unavailable;
		// only the status moves.
		return api.New(http.StatusUnprocessableEntity, api.CodeUnavailable, messages.sendFailed).WithCause(err)
	}
	a.logger.InfoContext(r.Context(), "contact form: mail sent")
	return api.Write(w, http.StatusOK, contactResponse{Message: messages.sent})
}
