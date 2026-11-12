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

	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/csrf"
	"github.com/ostendorf-cloud/govue/health"
	"github.com/ostendorf-cloud/govue/metrics"
	"github.com/ostendorf-cloud/govue/middleware"
	"github.com/ostendorf-cloud/govue/ratelimit"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
)

// maxRateLimitKeys bounds the number of distinct client addresses the rate
// limiter tracks at once, the same bound govue's own example applies; see
// the ratelimit package for why that bound matters.
const maxRateLimitKeys = 100_000

// App holds every dependency the website's handler needs.
type App struct {
	logger *slog.Logger

	health *health.Registry

	limiter   *ratelimit.Limiter
	rateLimit *middleware.RateLimit
	protector *csrf.Protector

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

	clientIPs := clientip.NewResolver(cfg.TrustedProxies)

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

	// No session exists yet in this milestone, so Expected always reports
	// that there is nothing to forge: csrf.Check then passes every unsafe
	// method on that half alone. The origin check, the other half, still
	// runs on every unsafe method regardless, which is what this is wired
	// up for ahead of the contact form a later milestone adds: that form
	// only has to start deriving a real per-session token, never wire the
	// protector itself.
	protector, err := csrf.New(func(*http.Request) (string, bool) { return "", false })
	fail(err)
	a.protector = protector

	a.security = middleware.DefaultSecurity()
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

	route := func(h http.Handler) http.Handler {
		return middleware.Chain(h, a.rateLimit.Middleware, a.protector.Middleware)
	}

	mux.Handle("GET /assets/", route(a.pages.assetHandler()))
	mux.Handle("/", route(a.pages.pageHandler()))

	return middleware.Chain(mux,
		middleware.Recover(a.logger),
		middleware.RequestID(a.logger),
		a.security.Middleware,
		httpMetrics.Middleware,
		a.accessLog.Middleware,
	)
}
