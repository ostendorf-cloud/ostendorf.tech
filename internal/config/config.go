// Package config loads and validates the website's configuration from the
// environment, once, in main. Every problem is collected and returned
// together, so a misconfigured deployment reports all of its mistakes on
// the first start instead of one per restart.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"net/url"
	"os"
	"time"

	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/envconf"
	"github.com/ostendorf-cloud/govue/logging"
)

// Defaults applied to a field left unset in the environment.
const (
	DefaultListenAddr     = ":8080"
	DefaultSiteURL        = "https://ostendorf.tech"
	DefaultLogLevel       = "info"
	DefaultRateLimitBurst = 120
	DefaultRateLimitPer   = time.Minute
)

// Config is the website's whole configuration, read once in main.
type Config struct {
	ListenAddr string
	SiteURL    *url.URL
	LogLevel   slog.Level

	// TrustedProxies are the TCP peers whose X-Forwarded-For entry clientip
	// trusts. Empty means every request is attributed to its TCP peer.
	TrustedProxies []netip.Prefix

	// RateLimitBurst and RateLimitPer bound how many requests one client
	// address may make in a row before it has to wait; see the ratelimit
	// package for the exact token bucket behaviour.
	RateLimitBurst int
	RateLimitPer   time.Duration
}

// Load reads every setting from the environment through getenv and
// validates the result. A nil getenv reads the real process environment.
// Every problem is reported together through the returned error, never one
// at a time.
func Load(getenv func(string) string) (Config, error) {
	loader := envconf.New(getenv)
	resolved := resolvedGetenv(getenv)

	cfg := Config{
		ListenAddr:     loader.String("LISTEN_ADDR", DefaultListenAddr),
		SiteURL:        loader.URL("SITE_URL", DefaultSiteURL),
		RateLimitBurst: loader.Int("RATE_LIMIT_BURST", DefaultRateLimitBurst, 1, 100_000),
		RateLimitPer:   loader.Duration("RATE_LIMIT_PER", DefaultRateLimitPer, time.Second, time.Hour),
	}

	var problems []error

	level, err := logging.Level(loader.String("LOG_LEVEL", DefaultLogLevel))
	if err != nil {
		problems = append(problems, fmt.Errorf("config: LOG_LEVEL: %w", err))
	}
	cfg.LogLevel = level

	prefixes, err := clientip.ParseCIDRs(resolved("TRUSTED_PROXY_CIDRS"))
	if err != nil {
		problems = append(problems, fmt.Errorf("config: TRUSTED_PROXY_CIDRS: %w", err))
	}
	cfg.TrustedProxies = prefixes

	if err := errors.Join(append(problems, loader.Err())...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// resolvedGetenv defaults a nil lookup function to the real process
// environment, the same default envconf.New applies, so a caller of
// clientip.ParseCIDRs never has to special case a nil getenv.
func resolvedGetenv(getenv func(string) string) func(string) string {
	if getenv == nil {
		return os.Getenv
	}
	return getenv
}
