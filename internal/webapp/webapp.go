// Package webapp wires the website's HTTP handler from the pieces govue
// provides: health, metrics, the security and access log middleware, the
// client IP resolver, the rate limiter and the CSRF protector. See
// (*App).handler for the two chains and the order each middleware has to
// run in, the same shape govue's own example documents.
package webapp

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ostendorf-cloud/govue/api"
	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/csrf"
	"github.com/ostendorf-cloud/govue/health"
	"github.com/ostendorf-cloud/govue/metrics"
	"github.com/ostendorf-cloud/govue/middleware"
	"github.com/ostendorf-cloud/govue/ratelimit"
	"github.com/ostendorf-cloud/govue/session"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/contact"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/pow"
)

// maxRateLimitKeys bounds the number of distinct client addresses a rate
// limiter tracks at once, the same bound govue's own example applies; see
// the ratelimit package for why that bound matters.
const maxRateLimitKeys = 100_000

// maxSessions bounds how many anonymous sessions session.MemoryStore keeps
// at once, deliberately far below govue's own 10000 default.
//
// A session here carries no account: an opaque id and three timestamps,
// nothing this application ever writes into Session.Data, Groups, Subject,
// Username, Name or Email. That keeps one entry's cost to roughly the map's
// per-entry overhead plus the id string stored twice (as the map key and as
// Session.ID), well under 1KiB even counting Go's own bookkeeping. 2000
// sessions therefore cost on the order of 1-2MiB, a rounding error against
// the chart's 128Mi pod limit (charts/ostendorf.tech/values.yaml) even
// before the sweep in MemoryStore.Save ever runs, and the bound is what
// stands between that budget and an attacker who keeps minting fresh
// sessions: without it, a flood of distinct GET /api/v1/challenge requests
// (the only place a session is minted) would grow the store without limit,
// capped only by RAM.
//
// A session lives up to session.DefaultLifetime, 12 hours, so this is also
// a ceiling on how many distinct visitors can be mid-visit to the contact
// form at once within any 12 hour window before MemoryStore.Save starts
// evicting one arbitrary entry per new arrival to make room. 2000 is a
// generous multiple of any realistic concurrency for one small business's
// contact form, legitimate or abusive within the challenge rate limit's own
// per-address budget (DefaultChallengeRateLimitBurst), so that eviction is
// not expected to ever trigger in practice; it exists as the hard ceiling,
// not the working set.
const maxSessions = 2_000

// App holds every dependency the website's handler needs.
type App struct {
	logger *slog.Logger

	siteURL *url.URL

	health *health.Registry

	limiter          *ratelimit.Limiter
	rateLimit        *middleware.RateLimit
	challengeRateLim *middleware.RateLimit
	contactLimiter   *ratelimit.Limiter
	clientIPs        *clientip.Resolver
	protector        *csrf.Protector

	sessions *session.Manager
	csrfKey  []byte

	challenges    *pow.Service
	challengeKeys [][]byte

	mailer         *contact.Sender
	smtpCreds      contact.Credentials
	smtpConfigured bool
	envelope       contact.Envelope

	security  middleware.Security
	accessLog middleware.AccessLog

	pages *pageServer

	handler http.Handler
}

// New builds the App from cfg and logger, serving the built SPA and the
// prerendered HTML out of dist and prerendered. Neither is modified; both
// are read only, typically the two go:embed filesystems cmd/website embeds.
func New(cfg config.Config, logger *slog.Logger, dist, prerendered fs.FS) (*App, error) {
	a := &App{logger: logger}

	var problems []error
	fail := func(err error) {
		if err != nil {
			problems = append(problems, err)
		}
	}

	a.siteURL = cfg.SiteURL

	clientIPs := clientip.NewResolver(cfg.TrustedProxies)
	a.clientIPs = clientIPs

	pages, err := newPageServer(dist, prerendered)
	fail(err)
	a.pages = pages

	a.health = health.New()

	registry := metrics.NewRegistry()
	httpMetrics := metrics.NewHTTP(registry)
	metricsHandler := metrics.Handler(registry)

	a.limiter = ratelimit.New(cfg.RateLimitBurst, cfg.RateLimitPer, maxRateLimitKeys)
	rateLimit, err := middleware.NewRateLimit(middleware.RateLimitConfig{Limiter: a.limiter, ClientIP: clientIPs})
	fail(err)
	a.rateLimit = rateLimit

	challengeLimiter := ratelimit.New(cfg.ChallengeRateLimitBurst, cfg.ChallengeRateLimitPer, maxRateLimitKeys)
	challengeRateLim, err := middleware.NewRateLimit(middleware.RateLimitConfig{
		Limiter:  challengeLimiter,
		ClientIP: clientIPs,
		// GET /api/v1/challenge is fetched by the page itself, in the
		// visitor's own locale, so its 429 must read in that locale too,
		// not govue's fixed English default.
		Respond: a.challengeRateLimitRespond,
	})
	fail(err)
	a.challengeRateLim = challengeRateLim

	a.contactLimiter = ratelimit.New(cfg.ContactRateLimitBurst, cfg.ContactRateLimitPer, maxRateLimitKeys)

	a.sessions, err = session.New(session.Config{
		Store:          session.NewMemoryStore(nil, maxSessions),
		AllowPlainHTTP: cfg.AllowPlainHTTP,
	})
	fail(err)
	a.csrfKey = cfg.CSRFKey

	// Expected is the one place this application connects the session
	// layer to csrf, which never imports session itself: it reads the
	// session the request itself carries and derives that session's
	// token, the same read the example in govue documents. It reads the
	// store directly rather than through ensureSession's request context,
	// because the one unsafe route this checks, POST /api/v1/contact,
	// never runs ensureSession itself: its session was already minted by
	// an earlier GET, and the browser sends the cookie back on its own.
	// csrf.Protector.Check still treats "no session" as nothing to forge,
	// but requireSession now sits in front of this route and refuses a
	// request that carries no session at all before the protector's own
	// check would ever run for it; in practice this route's "no session"
	// case never reaches the protector any more.
	protector, err := csrf.New(func(r *http.Request) (string, bool) {
		s, ok, err := a.sessions.Read(r.Context(), r)
		if err != nil || !ok {
			return "", false
		}
		return csrf.Token(a.csrfKey, s.ID), true
	})
	fail(err)
	a.protector = protector

	a.challenges = pow.NewService(challengeMaxNumber, challengeTTL, challengeMinAge)
	a.challengeKeys = nil
	if len(cfg.ChallengeKey) > 0 {
		a.challengeKeys = [][]byte{cfg.ChallengeKey}
	}

	heloName := cfg.SiteURL.Hostname()
	a.mailer = contact.NewSender(cfg.SMTPAddr, heloName)
	a.smtpCreds = contact.Credentials{Username: cfg.SMTPUsername, Password: cfg.SMTPPassword}
	a.smtpConfigured = cfg.SMTPUsername != "" && cfg.SMTPPassword != ""
	a.envelope = contact.Envelope{From: cfg.ContactFrom, To: cfg.ContactTo}

	a.security = middleware.DefaultSecurity()
	if cfg.AllowPlainHTTP {
		// A browser that reached this service over plain HTTP must never
		// be told to upgrade every future visit to HTTPS.
		a.security.HSTS = 0
	}
	a.accessLog = middleware.AccessLog{ClientIP: clientIPs}

	if err := errors.Join(problems...); err != nil {
		return nil, fmt.Errorf("webapp: build app: %w", err)
	}

	a.handler = a.buildHandler(httpMetrics, metricsHandler)
	return a, nil
}

// Health is the readiness registry, so main can drain it before shutdown.
func (a *App) Health() *health.Registry {
	return a.health
}

// Handler is the website's whole HTTP handler.
func (a *App) Handler() http.Handler {
	return a.handler
}

// buildHandler assembles the two chains: one outer chain wrapping the mux,
// one route chain applied to every handler the mux dispatches to. See
// docs/concepts/govue.md in govue for why a single flat chain cannot get
// both the access log and a self-answering middleware right at once.
func (a *App) buildHandler(httpMetrics *metrics.HTTP, metricsHandler http.Handler) http.Handler {
	mux := http.NewServeMux()

	// Health and metrics are mounted bare, outside the route chain: a load
	// balancer polling /healthz must never be rate limited, and a scrape
	// must never be refused for a missing CSRF token.
	a.health.Handle(mux)
	mux.Handle("GET /metrics", metricsHandler)

	// route is the per-handler chain every page and bare resource gets: the
	// general rate limit, then the CSRF protector's origin check (the
	// token half only ever refuses an unsafe method, and nothing a page
	// GET answers is one). No page route carries a session any more: the
	// only place a session is minted is GET /api/v1/challenge below, which
	// already returns the CSRF token the frontend actually reads in the
	// same response body. A page GET's own cookie would never be read by
	// anything, so minting one there bought nothing and only grew the
	// session store unbounded for no reason. See requireSession in
	// session.go for what "no session" means for POST /api/v1/contact now.
	route := func(h http.Handler) http.Handler {
		return middleware.Chain(h, a.rateLimit.Middleware, a.protector.Middleware)
	}

	mux.Handle("GET /assets/", route(a.pages.assetHandler()))
	mux.Handle("GET /robots.txt", route(a.robotsHandler()))
	mux.Handle("GET /sitemap.xml", route(a.sitemapHandler()))

	mux.Handle("GET /api/v1/challenge", middleware.Chain(api.Handle(a.handleChallenge), a.challengeRateLim.Middleware, a.ensureSession, a.protector.Middleware))
	// The contact rate limit is not wired in here: it runs inside
	// handleContact itself, right after the honeypot check, so a bot that
	// fills the honeypot never spends the shared budget other visitors
	// rely on. requireSession refuses a request with no session before the
	// CSRF protector's own check would ever apply to this route.
	mux.Handle("POST /api/v1/contact", middleware.Chain(api.Handle(a.handleContact), a.requireSession, a.protector.Middleware))
	mux.Handle("/api/", route(api.NotFoundHandler()))

	registerRoutes(mux, a.pages, route)
	mux.Handle("/", route(a.pages.pageHandler()))

	return middleware.Chain(mux,
		middleware.Recover(a.logger),
		middleware.RequestID(a.logger),
		a.security.Middleware,
		httpMetrics.Middleware,
		a.accessLog.Middleware,
	)
}
