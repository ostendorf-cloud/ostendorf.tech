// Package webapp wires the website's two HTTP handlers from the pieces
// govue provides. Handler is the public one: the security and access log
// middleware, the client IP resolver, the rate limiter and the CSRF
// protector, in front of the pages, assets and API routes. AdminHandler is
// the second, smaller one: health, readiness and metrics, reached only from
// inside the cluster, never through the chart's own Ingress; see
// (*App).buildHandler and (*App).AdminHandler for the two chains and the
// order each middleware has to run in, the same shape govue's own example
// documents.
package webapp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/ostendorf-cloud/govue/api"
	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/csrf"
	"github.com/ostendorf-cloud/govue/health"
	"github.com/ostendorf-cloud/govue/metrics"
	"github.com/ostendorf-cloud/govue/middleware"
	"github.com/ostendorf-cloud/govue/ratelimit"
	"github.com/ostendorf-cloud/govue/session"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/analytics"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/contact"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/instagram"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/pow"
)

// maxRateLimitKeys bounds the number of distinct client addresses a rate
// limiter tracks at once, the same bound govue's own example applies; see
// the ratelimit package for why that bound matters.
const maxRateLimitKeys = 100_000

// mailLimiterKey is the one bucket mailLimiter ever tracks: every contact
// mail this service sends counts against it, regardless of who asked. See
// mailLimiter's own comment for why a per-client limiter cannot do this
// job, and handleContact for where it is checked.
const mailLimiterKey = "mail"

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
	// metricsHandler answers GET /metrics on the admin handler only; see
	// AdminHandler. It is built once in New, from the one metrics.Registry
	// this App ever creates, so the admin handler never has to rebuild it.
	metricsHandler http.Handler

	limiter          *ratelimit.Limiter
	rateLimit        *middleware.RateLimit
	challengeRateLim *middleware.RateLimit
	contactLimiter   *ratelimit.Limiter
	mailLimiter      *ratelimit.Limiter
	clientIPs        *clientip.Resolver
	protector        *csrf.Protector

	// mailCeilingWarnedAt is the unix nanosecond timestamp warnMailCeiling
	// last actually logged at, zero meaning "never yet"; mailCeilingWarnEvery
	// is how long it waits before logging again. Together they keep a
	// sustained flood of refused submissions from turning into one warning
	// per request: see warnMailCeiling.
	mailCeilingWarnedAt  atomic.Int64
	mailCeilingWarnEvery time.Duration

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

	// instagramFeed is always present, even when no token is configured:
	// an empty feed is a valid answer, so no handler has to nil check it.
	// instagramSyncer is what fills it, and exists only when a token is
	// configured; Start is what sets it running.
	instagramFeed     *instagram.Feed
	instagramSyncer   *instagram.Syncer
	instagramInterval time.Duration

	// analyticsClient reports page views to Umami server to server. Nil
	// counts nothing, which is what a deployment with no UMAMI_URL gets.
	analyticsClient *analytics.Client

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
	a.metricsHandler = metrics.Handler(registry)

	a.limiter = ratelimit.New(cfg.RateLimitBurst, cfg.RateLimitPer, maxRateLimitKeys)
	rateLimit, err := middleware.NewRateLimit(middleware.RateLimitConfig{
		Limiter:  a.limiter,
		ClientIP: clientIPs,
		// This limiter guards every page, asset, robots.txt and sitemap.xml
		// request, so a browser denied here is asking for HTML, not govue's
		// JSON error envelope; see pageRateLimitRespond.
		Respond: a.pageRateLimitRespond,
	})
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

	// mailLimiter has exactly one key (mailLimiterKey), so maxKeys is 1,
	// not maxRateLimitKeys: clientip.Key groups IPv6 clients by /64, which
	// is the right shape for a per-client bucket but means a single
	// routed /48 is 65,536 independent buckets through contactLimiter
	// above. That leaves no ceiling on the mail account itself, only on
	// each slice an attacker could split their traffic into. mailLimiter
	// is checked in handleContact, after the honeypot and independent of
	// contactLimiter, so it bounds every mail this service ever sends in
	// the same window, no matter how many addresses asked.
	a.mailLimiter = ratelimit.New(cfg.MailRateLimitBurst, cfg.MailRateLimitPer, 1)
	// warnMailCeiling logs at most once per window; a window the same
	// length as the ceiling's own is what keeps one log line per burst of
	// refusals, however long that burst runs, rather than one per refused
	// request.
	a.mailCeilingWarnEvery = cfg.MailRateLimitPer

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
	// a.challengeKeys[0], the current key, is what Create signs a fresh
	// challenge with; Verify accepts a solution signed with any key in
	// the slice, current first. That is what lets a challenge issued just
	// before CHALLENGE_KEY was rotated still verify, for as long as
	// CHALLENGE_KEY_PREVIOUS still names the key it was issued under.
	a.challengeKeys = nil
	if len(cfg.ChallengeKey) > 0 {
		a.challengeKeys = append(a.challengeKeys, cfg.ChallengeKey)
	}
	if len(cfg.ChallengeKeyPrevious) > 0 {
		a.challengeKeys = append(a.challengeKeys, cfg.ChallengeKeyPrevious)
	}

	heloName := cfg.SiteURL.Hostname()
	a.mailer = contact.NewSender(cfg.SMTPAddr, heloName)
	a.smtpCreds = contact.Credentials{Username: cfg.SMTPUsername, Password: cfg.SMTPPassword}
	a.smtpConfigured = cfg.SMTPUsername != "" && cfg.SMTPPassword != ""
	a.envelope = contact.Envelope{From: cfg.ContactFrom, To: cfg.ContactTo}

	// The feed exists either way; only the syncer that fills it depends on
	// a token being configured.
	a.instagramFeed = &instagram.Feed{}
	a.instagramInterval = cfg.InstagramRefreshInterval
	if cfg.InstagramToken != "" {
		a.instagramSyncer = &instagram.Syncer{
			Client:     instagram.NewClient(cfg.InstagramAPIVersion),
			Token:      cfg.InstagramToken,
			RefreshAge: cfg.InstagramTokenRefreshAge,
			Count:      cfg.InstagramPostCount,
			Feed:       a.instagramFeed,
			Logger:     logger,
			Now:        time.Now,
		}
	}

	// config.Load already refused a half-configured pair, so one of the
	// two being set is enough to decide here.
	if cfg.UmamiURL != "" {
		a.analyticsClient = analytics.New(cfg.UmamiURL, cfg.UmamiWebsiteID, cfg.SiteURL.Hostname(), nil, logger)
	}

	a.security = middleware.DefaultSecurity()
	// DefaultSecurity's policy sets no worker-src, so a browser falls back
	// to script-src 'self' for the contact form's proof of work Worker:
	// web/src/lib/contactChallenge.ts instantiates it from a same-origin
	// module URL (new Worker(new URL('../workers/pow.worker.ts',
	// import.meta.url), {type: 'module'})), compiled by Vite into a
	// same-origin chunk under /assets/, never a blob: URL, and CSP Level 3
	// (https://www.w3.org/TR/CSP3/#directive-worker-src) specifies exactly
	// that fallback when worker-src is absent and script-src is present,
	// supported by every browser this site targets. The fallback alone
	// would already be enough to let the form submit. worker-src 'self' is
	// set explicitly anyway: the cost is one directive, and it removes any
	// future dependence on fallback behaviour if the bundler ever changes
	// how this Worker is built, on a path where being wrong means the
	// contact form never submits at all.
	a.security.Policy.Add("worker-src", "'self'")
	if cfg.AllowPlainHTTP {
		// A browser that reached this service over plain HTTP must never
		// be told to upgrade every future visit to HTTPS.
		a.security.HSTS = 0
	}
	a.accessLog = middleware.AccessLog{ClientIP: clientIPs}

	if err := errors.Join(problems...); err != nil {
		return nil, fmt.Errorf("webapp: build app: %w", err)
	}

	a.handler = a.buildHandler(httpMetrics)
	return a, nil
}

// Health is the readiness registry, so main can drain it before shutdown.
func (a *App) Health() *health.Registry {
	return a.health
}

// Start launches the background work the handler itself never does: the
// Instagram feed's sync loop and the Umami reporter's send loop. Both stop
// with ctx. It returns at once, and does nothing at all for a deployment
// that configured neither, so main always calls it unconditionally.
//
// Neither loop is part of readiness. A feed that has not loaded yet, or an
// Umami that cannot be reached, must never keep this service from serving
// pages, which is the whole reason they live out here rather than inside a
// request.
func (a *App) Start(ctx context.Context) {
	if a.instagramSyncer != nil {
		go a.instagramSyncer.Run(ctx, a.instagramInterval)
	}
	if a.analyticsClient != nil {
		go a.analyticsClient.Run(ctx)
	}
}

// trackPageView reports one served page to Umami, from this server, never
// from the browser. analytics.FromRequest decides what must not be counted
// at all (a prefetch, Do Not Track, Global Privacy Control, a client that
// is not a public address), and Track never blocks: a full queue drops the
// view rather than holding the response.
func (a *App) trackPageView(r *http.Request, title string) {
	if a.analyticsClient == nil {
		return
	}
	if v, ok := analytics.FromRequest(r, a.clientIPs.Resolve(r), title, a.siteURL.Hostname()); ok {
		a.analyticsClient.Track(v)
	}
}

// warnMailCeiling logs that the mail ceiling (mailLimiter) refused a
// submission, at most once per mailCeilingWarnEvery: a sustained flood of
// refused requests, each one already answered with its own 429, must not
// also turn into one WarnContext call per request, which would make the
// access log itself the amplifier the ceiling exists to guard against. The
// compare-and-swap loop is what lets several requests race on the same
// window without more than one of them winning the log line: a losing
// attempt simply means another goroutine already logged recently enough,
// never a reason to retry logging itself.
func (a *App) warnMailCeiling(ctx context.Context, wait time.Duration) {
	now := time.Now()
	for {
		last := a.mailCeilingWarnedAt.Load()
		if now.Sub(time.Unix(0, last)) < a.mailCeilingWarnEvery {
			return
		}
		if a.mailCeilingWarnedAt.CompareAndSwap(last, now.UnixNano()) {
			a.logger.WarnContext(ctx, "contact form: mail ceiling reached, refusing submissions",
				"retry_after", wait.String())
			return
		}
	}
}

// Handler is the website's public HTTP handler: pages, assets and the API
// routes. It answers no health, readiness or metrics route any more; see
// AdminHandler for those.
func (a *App) Handler() http.Handler {
	return a.handler
}

// AdminHandler is the website's administrative HTTP handler: GET /healthz,
// GET /readyz and GET /metrics, nothing else. It is served on a second
// listener (cfg.AdminListenAddr, httpserver.RunAdmin in cmd/website), never
// on the public one, so a kubelet's probe and a Prometheus scrape reach the
// pod directly instead of through the chart's own Ingress and the public
// listener's rate limiter. The chain is deliberately short, the same shape
// govue's own example/app.go (*app).adminHandler applies: only Recover and
// RequestID, no security headers, no CORS, no rate limiter, no access log
// and no metrics instrumentation middleware. A probe or a scraper is not a
// browser, cannot be forged, must not be throttled or flood the access log,
// and must not itself become a metric about scraping the metrics endpoint.
func (a *App) AdminHandler() http.Handler {
	mux := http.NewServeMux()
	a.health.Handle(mux)
	mux.Handle("GET /metrics", a.metricsHandler)

	return middleware.Chain(mux,
		middleware.Recover(a.logger),
		middleware.RequestID(a.logger),
	)
}

// buildHandler assembles the two chains: one outer chain wrapping the mux,
// one route chain applied to every handler the mux dispatches to. See
// docs/concepts/govue.md in govue for why a single flat chain cannot get
// both the access log and a self-answering middleware right at once.
func (a *App) buildHandler(httpMetrics *metrics.HTTP) http.Handler {
	mux := http.NewServeMux()

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
	// a.rateLimit.Middleware sits outermost here, the same position route()
	// gives it on every other route: it bounds raw request volume per
	// client, the general 120-per-minute-per-client budget every page,
	// asset and other API route already spends from. That is a different
	// limiter, with a different bucket, from contactLimiter: the per-client
	// contact submission budget (5 per 10 minutes by default), which
	// handleContact still checks on its own, right after the honeypot, so a
	// bot that fills the honeypot never spends the budget real visitors
	// behind the same address rely on. mailLimiter is a third, separate
	// limiter again, one bucket shared by every client: it bounds mail this
	// service actually attempts to send, checked only once a submission has
	// cleared validation and proof of work. All three now apply to this
	// route, each closing a gap the other two leave open; none substitutes
	// for either of the others. requireSession refuses a request with no
	// session before the CSRF protector's own check would ever apply to
	// this route.
	mux.Handle("POST /api/v1/contact", middleware.Chain(api.Handle(a.handleContact), a.rateLimit.Middleware, a.requireSession, a.protector.Middleware))

	// The Impressionen section's two reads. Both are plain public GETs:
	// no session, no CSRF token, nothing written. They answer from the
	// copy of the feed this server already fetched from Meta, which is
	// what keeps a visitor's browser away from Meta entirely.
	mux.Handle("GET /api/v1/instagram", route(api.Handle(a.handleInstagramFeed)))
	mux.Handle("GET /api/v1/instagram/{id}", route(http.HandlerFunc(a.handleInstagramImage)))

	mux.Handle("/api/", route(api.NotFoundHandler()))

	registerRoutes(mux, a.pages, route, a.trackPageView)
	mux.Handle("/", route(a.pages.pageHandler()))

	return middleware.Chain(mux,
		middleware.Recover(a.logger),
		middleware.RequestID(a.logger),
		a.security.Middleware,
		httpMetrics.Middleware,
		a.accessLog.Middleware,
	)
}
