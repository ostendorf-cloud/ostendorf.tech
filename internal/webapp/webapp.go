// Package webapp wires the website's two HTTP handlers from the pieces
// govue provides. Handler is the public one: the security and access log
// middleware, the client IP resolver and the rate limiter, in front of the
// pages, assets, API routes and govue's stateless contact handlers. AdminHandler is
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
	"time"

	"github.com/ostendorf-cloud/govue/analytics"
	"github.com/ostendorf-cloud/govue/api"
	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/contact"
	"github.com/ostendorf-cloud/govue/health"
	"github.com/ostendorf-cloud/govue/language"
	"github.com/ostendorf-cloud/govue/mail"
	"github.com/ostendorf-cloud/govue/metrics"
	"github.com/ostendorf-cloud/govue/middleware"
	"github.com/ostendorf-cloud/govue/ratelimit"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/feedclient"
)

// maxRateLimitKeys bounds the number of distinct client addresses a rate
// limiter tracks at once, the same bound govue's own example applies; see
// the ratelimit package for why that bound matters.
const maxRateLimitKeys = 100_000

// App holds every dependency the website's handler needs.
type App struct {
	logger *slog.Logger

	siteURL *url.URL

	health *health.Registry
	// metricsHandler answers GET /metrics on the admin handler only; see
	// AdminHandler. It is built once in New, from the one metrics.Registry
	// this App ever creates, so the admin handler never has to rebuild it.
	metricsHandler http.Handler

	limiter   *ratelimit.Limiter
	rateLimit *middleware.RateLimit
	clientIPs *clientip.Resolver

	// contact is govue's stateless contact form: the challenge and the
	// submission, see contact.go.
	contact *contact.Handler

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
	umamiHTTP  *http.Client
	mailSender mail.Sender
	now        func() time.Time
}

// WithUmamiHTTPClient sets the client that posts page views to Umami. The
// default is govue's own: no cookie jar, no redirects. A test passes a fake
// transport so nothing opens a socket.
func WithUmamiHTTPClient(c *http.Client) Option {
	return func(o *options) { o.umamiHTTP = c }
}

// WithMailSender replaces the contact form's SMTP sender. A test passes a
// fake that keeps the message in memory, so nothing opens a socket. It
// configures the sender even without SMTP_HOST.
func WithMailSender(s mail.Sender) Option {
	return func(o *options) { o.mailSender = s }
}

// WithClock sets the clock of the contact form's proof of work and of its
// mail ceiling warning. A test moves it past the minimum age of a challenge
// instead of sleeping.
func WithClock(now func() time.Time) Option {
	return func(o *options) { o.now = now }
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

	fail(a.buildContact(cfg, o.mailSender, o.now))

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

	// DefaultSecurity's policy carries worker-src 'self' since govue v0.8.0,
	// which the contact form's proof of work Worker needs: it is a
	// same-origin chunk under /assets/.
	a.security = middleware.DefaultSecurity()
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
	// general rate limit. There is no session and no CSRF protector: every
	// route below except the contact form answers GET only and changes
	// nothing, and the contact form checks the origin itself.
	route := func(h http.Handler) http.Handler {
		return middleware.Chain(h, a.rateLimit.Middleware)
	}

	mux.Handle("GET /assets/", route(a.pages.assetHandler()))
	for p := range publicFiles {
		mux.Handle("GET "+p, route(a.pages.publicFileHandler(p)))
	}
	mux.Handle("GET /robots.txt", route(a.robotsHandler()))
	mux.Handle("GET /sitemap.xml", route(a.sitemapHandler()))

	// govue's contact form: the general rate limit in front, then govue's
	// own origin check, per client limits, proof of work and mail ceiling
	// inside the handlers. See contact.go.
	a.mountContact(mux)

	// The Impressionen section's two reads. Both are plain public GETs:
	// nothing written. They answer from the
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
