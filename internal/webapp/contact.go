package webapp

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ostendorf-cloud/govue/contact"
	"github.com/ostendorf-cloud/govue/mail"
	"github.com/ostendorf-cloud/govue/middleware"
	"github.com/ostendorf-cloud/govue/pow"
	"github.com/ostendorf-cloud/govue/ratelimit"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
)

// The contact routes, govue's defaults: GovueContactForm posts to
// /contact and fetches /contact/challenge below the client's /api/v1.
const (
	contactPath          = "/api/v1/contact"
	contactChallengePath = "/api/v1/contact/challenge"
)

// contactSendTimeout bounds one send. It keeps httpserver's read timeout
// (15 seconds) plus the send below its write timeout (30 seconds), and the
// send below its shutdown timeout (20 seconds), as govue's contact package
// asks of a site.
const contactSendTimeout = 10 * time.Second

// subjectNameRunes shortens a long name in the subject. The body carries
// the full name.
const subjectNameRunes = 80

// contactSubject is the subject of every contact mail: the visitor's name,
// shortened. govue's mail builder encodes it and refuses what does not
// belong in a header.
func contactSubject(r contact.Request) string {
	return "Kontaktanfrage von " + truncateRunes(r.Name, subjectNameRunes)
}

// contactBody is the plain text body of every contact mail: a German intro
// line, govue's body with the name, the address and the message, and one
// line on the consent box. Every mail goes to the German owner, so the labels
// are govue's German ones (E-Mail, Nachricht) whatever language the page
// was in: Labels.Body, the documented way for a site's own Body, not
// DefaultBody, which follows Request.Language. The visitor's address is also
// in Reply-To, set by govue.
func contactBody(r contact.Request) string {
	return "Neue Kontaktanfrage über das Formular auf ostendorf.tech\n\n" +
		contactLabels.Body(r) + "\n" + consentLine(r) + "\n"
}

// contactLabels are govue's built in German labels of the mail body.
var contactLabels = contact.LabelsFor("de")

// consentLine states the consent box as the browser sent it and the time the
// server received the request. It records a claim of the client, not a
// proof: a modified client can send a ticked box it never showed.
func consentLine(r contact.Request) string {
	state := "nicht angehakt"
	if r.Consent {
		state = "angehakt"
	}
	return "Einwilligung: Kontrollkästchen laut Browser " + state +
		", Eingang beim Server " + r.Received.UTC().Format(time.RFC3339) +
		" (Angabe des Browsers, kein Nachweis)"
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// buildContact builds govue's contact handlers from cfg. Missing keys or a
// missing SMTP host are not an error: both routes then answer 503
// not_configured and the site still starts. sender, when not nil, replaces
// the SMTP sender, for a test.
func (a *App) buildContact(cfg config.Config, sender mail.Sender, now func() time.Time) error {
	var service *pow.Service
	keys := make([][]byte, 0, 2)
	if len(cfg.ChallengeKey) > 0 {
		keys = append(keys, cfg.ChallengeKey)
		// The first key signs, every key verifies: a challenge issued just
		// before CHALLENGE_KEY was rotated still verifies while
		// CHALLENGE_KEY_PREVIOUS names the key it was issued under.
		if len(cfg.ChallengeKeyPrevious) > 0 {
			keys = append(keys, cfg.ChallengeKeyPrevious)
		}
	}
	if len(keys) > 0 {
		var err error
		// The audience is the site's own host name, so a challenge of
		// another site signed with the same key is refused.
		service, err = pow.New(pow.Config{Audience: cfg.SiteURL.Hostname(), Keys: keys, Now: now})
		if err != nil {
			return fmt.Errorf("contact proof of work: %w", err)
		}
	}

	if sender == nil && cfg.SMTP != nil {
		smtp, err := mail.NewSMTP(*cfg.SMTP)
		if err != nil {
			return fmt.Errorf("contact smtp: %w", err)
		}
		sender = smtp
	}

	// The site's own origin, taken from SITE_URL, is trusted explicitly, so
	// a page served at SITE_URL may post even when a proxy rewrote the Host
	// header. Every other origin is refused.
	origin := cfg.SiteURL.Scheme + "://" + cfg.SiteURL.Host

	handlers, err := contact.New(contact.Config{
		PoW: service,
		// sender is a nil interface, never a typed nil pointer, when SMTP is
		// not configured, so govue sees it as missing.
		Sender: sender,
		// The envelope is fixed by the site and never comes from a request.
		From: cfg.ContactFrom.String(),
		To:   []string{cfg.ContactTo.String()},
		// Every limiter is per process, see AGENTS.md for why the chart
		// keeps a single replica.
		Limiter:          ratelimit.New(cfg.ContactRateLimitBurst, cfg.ContactRateLimitPer, contact.MaxLimiterKeys),
		MailLimiter:      ratelimit.New(cfg.MailRateLimitBurst, cfg.MailRateLimitPer, 1),
		ChallengeLimiter: ratelimit.New(cfg.ChallengeRateLimitBurst, cfg.ChallengeRateLimitPer, contact.MaxLimiterKeys),
		ClientIP:         a.clientIPs,
		TrustedOrigins:   []string{origin},
		Logger:           a.logger,
		Subject:          contactSubject,
		Body:             contactBody,
		// The form shows the consent box (requireConsent in HomeView.vue),
		// so a submission without a ticked box is a field error.
		RequireConsent: true,
		SendTimeout:    contactSendTimeout,
		Now:            now,
	})
	if err != nil {
		return fmt.Errorf("contact handlers: %w", err)
	}
	a.contact = handlers
	return nil
}

// mountContact registers govue's two contact routes. They sit behind the
// general rate limit and outside every other per-route middleware: the form
// is stateless, so there is no session to read, no CSRF token to check and
// no cookie to set. The handlers check the origin themselves, and the
// challenge route has its own per client limit inside the handler. The outer
// chain (security headers, metrics, access log) still applies.
func (a *App) mountContact(mux *http.ServeMux) {
	mux.Handle("GET "+contactChallengePath, middleware.Chain(a.contact.Challenge(), a.rateLimit.Middleware))
	mux.Handle("POST "+contactPath, middleware.Chain(a.contact.Submit(), a.rateLimit.Middleware))
}
