// Package webapp wires the API's two HTTP handlers from the pieces govue
// provides. Handler is the public one: the security and access log
// middleware, the client IP resolver and the rate limiter, in front of
// govue's stateless contact handlers and the Impressionen feed, and a JSON
// 404 for everything else. It serves no page: cmd/frontend does, in an image
// of its own, and the Ingress routes /api here. AdminHandler is the second,
// smaller one: health, readiness and metrics, reached only from inside the
// cluster, never through the chart's own Ingress; see (*App).buildHandler
// and (*App).AdminHandler for the two chains and the order each middleware
// has to run in, the same shape govue's own example documents.
package webapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ostendorf-cloud/govue/api"
	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/contact"
	"github.com/ostendorf-cloud/govue/health"
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

// App holds every dependency the API's handler needs.
type App struct {
	logger *slog.Logger

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

	handler http.Handler
}

// Option changes how New builds the App. It exists for tests.
type Option func(*options)

type options struct {
	mailSender mail.Sender
	now        func() time.Time
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

// New builds the App from cfg and logger.
func New(cfg config.Config, logger *slog.Logger, opts ...Option) (*App, error) {
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

	a.clientIPs = clientip.NewResolver(cfg.TrustedProxies)
	a.health = health.New()

	registry := metrics.NewRegistry()
	httpMetrics := metrics.NewHTTP(registry)
	a.metricsHandler = metrics.Handler(registry)

	a.limiter = ratelimit.New(cfg.RateLimitBurst, cfg.RateLimitPer, maxRateLimitKeys)
	// The API answers JSON only, so a client denied here gets govue's JSON
	// error envelope with Retry-After, the default Respond.
	rateLimit, err := middleware.NewRateLimit(middleware.RateLimitConfig{
		Limiter:  a.limiter,
		ClientIP: a.clientIPs,
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

	// govue's default policy, worker-src 'self' included since v0.8.0. The
	// API serves no page, so it adds no script source.
	a.security = middleware.DefaultSecurity()
	if cfg.AllowPlainHTTP {
		// A browser that reached this service over plain HTTP must never
		// be told to upgrade every future visit to HTTPS.
		a.security.HSTS = 0
	}
	a.accessLog = middleware.AccessLog{ClientIP: a.clientIPs}

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
// feed service sync loop. It stops with ctx. It returns at once, and does
// nothing at all for a deployment that configured no feed service, so main
// always calls it unconditionally.
//
// The loop is not part of readiness. A cache that has not loaded yet must
// never keep this service from serving, which is the whole reason it lives
// out here rather than inside a request.
func (a *App) Start(ctx context.Context) {
	if a.feedSyncer != nil {
		go a.feedSyncer.Run(ctx, a.feedRefreshInterval)
	}
}

// Handler is the API's public HTTP handler. It answers no health,
// readiness or metrics route; see AdminHandler for those.
func (a *App) Handler() http.Handler {
	return a.handler
}

// AdminHandler is the API's administrative HTTP handler: GET /healthz,
// GET /readyz and GET /metrics, nothing else. It is served on a second
// listener (cfg.AdminListenAddr, httpserver.RunAdmin in cmd/api), never on
// the public one, so a kubelet's probe and a Prometheus scrape reach the pod
// directly instead of through the chart's own Ingress and the public
// listener's rate limiter. The chain is deliberately short, the same shape
// govue's own example/app.go (*app).adminHandler applies: only Recover and
// RequestID.
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

	// route is the per-handler chain every route gets: the general rate
	// limit. There is no session and no CSRF protector: every route below
	// except the contact form answers GET only and changes nothing, and the
	// contact form checks the origin itself.
	route := func(h http.Handler) http.Handler {
		return middleware.Chain(h, a.rateLimit.Middleware)
	}

	// govue's contact form: the general rate limit in front, then govue's
	// own origin check, per client limits, proof of work and mail ceiling
	// inside the handlers. See contact.go.
	a.mountContact(mux)

	// The Impressionen section's two reads. Both are plain public GETs:
	// nothing written. They answer from the copy feedCache already fetched
	// from the feed service, which is what keeps a visitor's browser away
	// from the feed service, Instagram and Meta entirely. The route paths
	// are this website's public contract with the browser, documented by
	// web/src.
	mux.Handle("GET /api/v1/instagram", route(api.Handle(a.handleInstagramFeed)))
	mux.Handle("GET /api/v1/instagram/{id}", route(http.HandlerFunc(a.handleInstagramImage)))

	// Everything else is a JSON 404: under /api/ an unknown route, and any
	// other path one the Ingress never sends here.
	mux.Handle("/", route(api.NotFoundHandler()))

	return middleware.Chain(mux,
		middleware.Recover(a.logger),
		middleware.RequestID(a.logger),
		a.security.Middleware,
		httpMetrics.Middleware,
		a.accessLog.Middleware,
	)
}
