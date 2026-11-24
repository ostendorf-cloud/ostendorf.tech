// Package frontendconfig loads and validates the frontend image's
// configuration from the environment, once, in cmd/frontend's main. Every
// problem is collected and returned together. The frontend holds no secret:
// no variable here has a _FILE twin, and none is a key or a password.
package frontendconfig

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"net/url"
	"os"
	"time"

	"github.com/ostendorf-cloud/govue/analytics"
	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/envconf"
	"github.com/ostendorf-cloud/govue/logging"
)

// Defaults applied to a field left unset in the environment, the same as
// the API's for the variables both images read.
const (
	DefaultListenAddr      = ":8080"
	DefaultAdminListenAddr = ":8081"
	DefaultSiteURL         = "https://ostendorf.tech"
	DefaultLogLevel        = "info"
	DefaultRateLimitBurst  = 120
	DefaultRateLimitPer    = time.Minute
)

// EnvDevAPIProxy is the development switch: the URL of the API container
// govue's frontend forwards /api/ to, so a compose stack has one browser
// origin as the Ingress gives production. Set, it also turns on govue's
// Config.Development, which allows the proxy at all. The Helm chart never
// sets it.
const EnvDevAPIProxy = "FRONTEND_DEV_API_PROXY"

// Config is the frontend image's whole configuration.
type Config struct {
	ListenAddr string
	// AdminListenAddr serves health, readiness and metrics, never the
	// pages; govue's frontend refuses an empty one.
	AdminListenAddr string
	// SiteURL is the site's public origin. Its host name is the hostname
	// page views are reported under. The pages, the sitemap and robots.txt
	// carry the origin the build was made with (VITE_SITE_URL).
	SiteURL  *url.URL
	LogLevel slog.Level

	// TrustedProxies are the TCP peers whose X-Forwarded-For entry clientip
	// trusts, for the access log, the rate limit and the page view address.
	TrustedProxies []netip.Prefix

	// RateLimitBurst and RateLimitPer bound how many requests one client
	// address may make in a row before it has to wait.
	RateLimitBurst int
	RateLimitPer   time.Duration

	// AllowPlainHTTP drops the Strict-Transport-Security header, for a
	// plain HTTP development run.
	AllowPlainHTTP bool

	// Umami is the page view counting, read by govue's analytics.LoadConfig
	// (UMAMI_URL, UMAMI_WEBSITE_ID, UMAMI_TIMEOUT, UMAMI_QUEUE_SIZE). The
	// zero value counts nothing.
	Umami analytics.Config

	// DevAPIProxy is FRONTEND_DEV_API_PROXY, empty in production. Only a
	// development stack sets it; see EnvDevAPIProxy.
	DevAPIProxy string
}

// Development reports whether the development switch is set. It is true
// exactly when FRONTEND_DEV_API_PROXY is set, never from another variable.
func (c Config) Development() bool {
	return c.DevAPIProxy != ""
}

// Load reads every setting through getenv and validates the result. A nil
// getenv reads the real process environment.
func Load(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	loader := envconf.New(getenv)

	cfg := Config{
		ListenAddr:      loader.String("LISTEN_ADDR", DefaultListenAddr),
		AdminListenAddr: loader.String("ADMIN_ADDR", DefaultAdminListenAddr),
		SiteURL:         loader.URL("SITE_URL", DefaultSiteURL),
		RateLimitBurst:  loader.Int("RATE_LIMIT_BURST", DefaultRateLimitBurst, 1, 100_000),
		RateLimitPer:    loader.Duration("RATE_LIMIT_PER", DefaultRateLimitPer, time.Second, time.Hour),
		AllowPlainHTTP:  loader.Bool("ALLOW_PLAIN_HTTP", false),
		// The literal name, so internal/envsync finds the variable here.
		DevAPIProxy: loader.String("FRONTEND_DEV_API_PROXY", ""),
	}

	var problems []error
	fail := func(err error) {
		if err != nil {
			problems = append(problems, err)
		}
	}

	level, err := logging.Level(loader.String("LOG_LEVEL", DefaultLogLevel))
	fail(wrap("LOG_LEVEL", err))
	cfg.LogLevel = level

	prefixes, err := clientip.ParseCIDRs(getenv("TRUSTED_PROXIES"))
	fail(wrap("TRUSTED_PROXIES", err))
	cfg.TrustedProxies = prefixes

	if cfg.AdminListenAddr == "" {
		fail(wrap("ADMIN_ADDR", errors.New("must not be empty, the probes and the metrics live there")))
	}

	// govue's own messages name the variable at fault, so no wrap here.
	umami, err := analytics.LoadConfig(getenv)
	fail(err)
	cfg.Umami = umami

	if err := errors.Join(append(problems, loader.Err())...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// wrap names the variable a problem came from, or returns nil.
func wrap(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("frontendconfig: %s: %w", name, err)
}
