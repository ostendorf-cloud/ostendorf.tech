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

	"github.com/ostendorf-cloud/govue/analytics"
	"github.com/ostendorf-cloud/govue/api"
	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/csrf"
	"github.com/ostendorf-cloud/govue/health"
	"github.com/ostendorf-cloud/govue/language"
	"github.com/ostendorf-cloud/govue/metrics"
	"github.com/ostendorf-cloud/govue/middleware"
	"github.com/ostendorf-cloud/govue/ratelimit"
	"github.com/ostendorf-cloud/govue/session"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/contact"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/feedclient"
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

	// feedCache is always present, even when no feed service is
	// configured: an empty cache is a valid answer, so no handler has to
	// nil check it. feedSyncer is what fills it from the feed service
	// (internal/feedclient), and exists only when FEED_SERVICE_URL is
	// set; Start is what sets it running. There is no Meta client, no
	// Instagram credential and no OpenBao client anywhere in this
	// package or its callers: internal/feed, a separate binary, owns
	// all of that instead.
	feedCache           *feedclient.Cache
	feedSyncer          *feedclient.Syncer
	feedRefreshInterval time.Duration

	// views counts page views server to server, after a page was served,
	// and reports them to Umami. Never nil after New; a deployment with no
	// UMAMI_URL gets a client that counts nothing. Close sends what is
	// still queued on shutdown.
	views *analytics.Client

	handler http.Handler
}

// Option changes how New builds the App. It exists for tests and for the
// one thing a caller may need to swap, the transport that reaches Umami.
type Option func(*options)

type options struct {
	umamiHTTP *http.Client
}

// WithUmamiHTTPClient sets the client that posts page views to Umami. The
// default is govue's own: no cookie jar, no redirects. A test passes a fake
// transport so nothing opens a socket.
func WithUmamiHTTPClient(c *http.Client) Option {
	return func(o *options) { o.umamiHTTP = c }
}

// viewsDroppedMetric names the counter of page views lost to a full queue
// or to a shutdown before they reached Umami.
const viewsDroppedMetric = "analytics_views_dropped_total"

// New builds the App from cfg and logger, serving the built SPA and the
// prerendered HTML out of dist and prerendered. Neither is modified; both
// are read only, typically the two go:embed filesystems cmd/website embeds.
// The caller calls Close once the server has stopped.
func New(cfg config.Config, logger *slog.Logger, dist, prerendered fs.FS, opts ...Option) (*App, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
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

	scriptHash, err := themeBootScriptHash(prerendered)
	fail(err)

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

	// The cache exists either way; only the syncer that fills it depends
	// on a feed service being configured.
	a.feedCache = &feedclient.Cache{}
	a.feedRefreshInterval = cfg.FeedRefreshInterval
	if cfg.FeedServiceURL != "" {
		a.feedSyncer = &feedclient.Syncer{
			Client: feedclient.New(cfg.FeedServiceURL),
			Cache:  a.feedCache,
			Logger: logger,
		}
	}

	// config.Load already refused a half-configured pair. The whole client
	// address is never sent: FullIP stays off, so govue masks it to its /24
	// or /48. The resolver is the same trusted proxy one the rate limiter
	// and the access log use. Dropped views are counted in /metrics.
	views, err := analytics.New(cfg.Umami, analytics.Options{
		Hostname:   cfg.SiteURL.Hostname(),
		Logger:     logger,
		HTTPClient: o.umamiHTTP,
		ClientIP:   clientIPs,
		OnDrop: metrics.NewCounter(registry, viewsDroppedMetric,
			"Page views lost to a full queue or a shutdown before they reached Umami."),
	})
	fail(err)
	a.views = views
	if err == nil {
		logViewsStatus(logger, cfg, views)
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
	// The visitor's stored theme choice is corrected before first paint by a
	// small inline script in the prerendered <head> (see
	// web/scripts/prerender.ts and internal/webapp/themeboot.go), which this
	// site's CSP would otherwise block: script-src carries no 'unsafe-inline'
	// and the prerendered HTML has no per-response nonce to carry, since it
	// is served as static bytes. Allowing exactly this script's own hash,
	// computed above from the real prerendered output rather than a second
	// copy of its text, is what lets it run.
	if scriptHash != "" {
		a.security.Policy.Add("script-src", "'sha256-"+scriptHash+"'")
	}
	if cfg.AllowPlainHTTP {
		// A browser that reached this service over plain HTTP must never
		// be told to upgrade every future visit to HTTPS.
		a.security.HSTS = 0
	}
	a.accessLog = middleware.AccessLog{ClientIP: clientIPs}

	if err := errors.Join(problems...); err != nil {
		a.closeViewsNow()
		return nil, fmt.Errorf("webapp: build app: %w", err)
	}

	handler, err := a.buildHandler(httpMetrics)
	if err != nil {
		a.closeViewsNow()
		return nil, fmt.Errorf("webapp: build app: %w", err)
	}
	a.handler = handler
	return a, nil
}

// logViewsStatus says once at startup whether page views are counted: govue
// logs nothing when counting is off. Counting on with no trusted proxy is a
// warning, because behind an ingress every request then comes from the
// proxy's private address and nothing is counted, silently.
func logViewsStatus(logger *slog.Logger, cfg config.Config, views *analytics.Client) {
	logger.Info("page view counting", "enabled", views.Enabled())
	if views.Enabled() && len(cfg.TrustedProxies) == 0 {
		logger.Warn("page view counting is on but TRUSTED_PROXIES is empty: behind a proxy no view is counted")
	}
}

// closeViewsNow stops the page view workers of an App that never got
// built. Nothing was served, so nothing is queued.
func (a *App) closeViewsNow() {
	if a.views != nil {
		_ = a.views.Close(context.Background())
	}
}

// Close sends the page views still queued and stops the workers, within
// ctx. Call it once, after the server has stopped, with a bounded context.
func (a *App) Close(ctx context.Context) error {
	return a.views.Close(ctx)
}

// Health is the readiness registry, so main can drain it before shutdown.
func (a *App) Health() *health.Registry {
	return a.health
}

// Start launches the background work the handler itself never does: the
// feed service sync loop. It stops with ctx. It returns at once, and does
// nothing at all for a deployment that configured no feed service, so main
// always calls it unconditionally. The page view workers start in New and
// stop in Close.
//
// The loop is not part of readiness. A cache that has not loaded yet must
// never keep this service from serving pages, which is the whole reason it
// lives out here rather than inside a request.
func (a *App) Start(ctx context.Context) {
	if a.feedSyncer != nil {
		go a.feedSyncer.Run(ctx, a.feedRefreshInterval)
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
func (a *App) buildHandler(httpMetrics *metrics.HTTP) (http.Handler, error) {
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
	for p := range publicFiles {
		mux.Handle("GET "+p, route(a.pages.publicFileHandler(p)))
	}
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
	// copy feedCache already fetched from the feed service, which is what
	// keeps a visitor's browser away from the feed service, Instagram and
	// Meta entirely. The route paths keep their original, content-shaped
	// names rather than taking the feed service's own "feed" naming: it
	// is this website's public contract with the browser, documented by
	// web/src, and changing it is not a consequence of swapping what
	// serves it.
	mux.Handle("GET /api/v1/instagram", route(api.Handle(a.handleInstagramFeed)))
	mux.Handle("GET /api/v1/instagram/{id}", route(http.HandlerFunc(a.handleInstagramImage)))

	mux.Handle("/api/", route(api.NotFoundHandler()))

	registerRoutes(mux, a.pages, route)

	// govue's legal routes: /imprint, /privacy and /terms to the page in the
	// visitor's language, and the German aliases /impressum, /datenschutz,
	// /datenschutzerklaerung and /agb to the German page. They answer a bare
	// 302 and need no session and no CSRF check, so they sit outside the
	// route chain, as govue's example does; the outer chain still applies.
	// They answer 302, so no alias is counted as a page view.
	// The site has terms (the AGB page), hence Terms.
	if err := language.HandleLegal(mux, legalConfig(), nil); err != nil {
		return nil, fmt.Errorf("legal routes: %w", err)
	}

	mux.Handle("/", route(a.pages.pageHandler()))

	// The page view counter wraps the mux directly and nothing sits between
	// the two: it reads the route pattern the mux sets on the request, and a
	// middleware in between that replaces the request (RequestID, security
	// headers, ...) would hide that pattern, so no view would be counted.
	// With counting off, Middleware returns the mux unchanged.
	return middleware.Chain(a.views.Middleware(mux),
		middleware.Recover(a.logger),
		middleware.RequestID(a.logger),
		a.security.Middleware,
		httpMetrics.Middleware,
		a.accessLog.Middleware,
	), nil
}

// legalConfig is the language configuration of the legal routes: the two
// locales the site serves, German first as the default, and govue's
// default language cookie. Terms is on because the site has an AGB page.
func legalConfig() language.LegalConfig {
	return language.LegalConfig{
		Config: language.Config{Languages: locales, Default: defaultLocale},
		Terms:  true,
	}
}
