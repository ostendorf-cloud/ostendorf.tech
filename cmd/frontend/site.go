package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ostendorf-cloud/govue/analytics"
	"github.com/ostendorf-cloud/govue/api"
	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/frontend"
	"github.com/ostendorf-cloud/govue/language"
	"github.com/ostendorf-cloud/govue/middleware"
	"github.com/ostendorf-cloud/govue/ratelimit"
	"github.com/ostendorf-cloud/govue/themeboot"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/frontendconfig"
)

// The browser bar colours, the two ends of the theme's --color-bg in
// web/src/theme/vtech.css: light first, dark second. govue writes one
// theme-color tag per mode from them; site_test.go reads both back out of
// the theme, so they cannot drift.
const (
	themeColorLight = "#f6f5fd"
	themeColorDark  = "#07060f"
)

// socialImage is the one public file beyond govue's three favicon files:
// the 1200 by 630 preview the pages name in og:image.
const socialImage = "og-image.png"

// maxRateLimitKeys bounds the client addresses the page rate limiter
// tracks, the bound govue's own example applies.
const maxRateLimitKeys = 100_000

// languages are the site's two, German first as the default.
var languages = []string{"de", "en"}

// themeBootOptions are the options of govue's theme boot script. The site
// opens dark: with nothing stored the script keeps the prerendered
// data-mode="dark" and pins the theme-color tags to the dark colour, the
// same mode web/src/lib/theme.ts hands useTheme and the prerender writes.
// A stored choice, system included, still wins. The storage key is govue's
// default, themeboot.StorageKey, the same as web/src/lib/theme.ts.
func themeBootOptions() []themeboot.Option {
	return []themeboot.Option{themeboot.WithDefaultMode(themeboot.ModeDark)}
}

// redirects keep the former AGB paths: /<code>/agb and /<code>/agb/ answer
// 301 to the terms page of the same language. The German aliases /agb,
// /impressum, /datenschutz and /datenschutzerklaerung are govue's own legal
// handler, which frontend.New mounts.
func redirects() []frontend.Redirect {
	var out []frontend.Redirect
	for _, code := range languages {
		to := "/" + code + "/terms"
		out = append(out,
			frontend.Redirect{From: "/" + code + "/agb", To: to},
			frontend.Redirect{From: "/" + code + "/agb/", To: to},
		)
	}
	return out
}

// options are what a test replaces: the transport that reaches Umami and
// the one that reaches the API in development.
type options struct {
	umamiHTTP       *http.Client
	devAPITransport http.RoundTripper
}

// newServer builds govue's frontend server from the build and cfg. It
// counts page views when Umami is configured and forwards /api/ to the API
// container only with the development switch, FRONTEND_DEV_API_PROXY.
func newServer(cfg frontendconfig.Config, build fs.FS, logger *slog.Logger, o options) (*frontend.Server, error) {
	resolver := clientip.NewResolver(cfg.TrustedProxies)

	security := middleware.DefaultSecurity()
	if cfg.AllowPlainHTTP {
		// A browser that reached this service over plain HTTP must never
		// be told to upgrade every future visit to HTTPS.
		security.HSTS = 0
	}

	rateLimit, err := middleware.NewRateLimit(middleware.RateLimitConfig{
		Limiter:  ratelimit.New(cfg.RateLimitBurst, cfg.RateLimitPer, maxRateLimitKeys),
		ClientIP: resolver,
		Respond:  pageRateLimitRespond,
	})
	if err != nil {
		return nil, fmt.Errorf("frontend: rate limit: %w", err)
	}

	// The whole client address is never sent: FullIP stays off, so govue
	// masks it. frontend registers the drop counter and closes the client.
	views, err := analytics.New(cfg.Umami, analytics.Options{
		Hostname:   cfg.SiteURL.Hostname(),
		Logger:     logger,
		HTTPClient: o.umamiHTTP,
		ClientIP:   resolver,
	})
	if err != nil {
		return nil, fmt.Errorf("frontend: analytics: %w", err)
	}
	logger.Info("page view counting", "enabled", views.Enabled())
	if views.Enabled() && len(cfg.TrustedProxies) == 0 {
		logger.Warn("page view counting is on but TRUSTED_PROXIES is empty: behind a proxy no view is counted")
	}

	s, err := frontend.New(frontend.Config{
		Build:          build,
		Language:       language.Config{Languages: languages, Default: languages[0]},
		ThemeColor:     themeColorLight,
		ThemeColorDark: themeColorDark,
		ThemeBoot:      themeBootOptions(),
		PublicFiles:    []string{socialImage},
		Redirects:      redirects(),
		Security:       &security,
		ClientIP:       resolver,
		Analytics:      views,
		RateLimit:      rateLimit,
		Logger:         logger,

		Development:     cfg.Development(),
		DevAPIProxy:     cfg.DevAPIProxy,
		DevAPITransport: o.devAPITransport,
	})
	if err != nil {
		// Nothing was served, so nothing is queued.
		_ = views.Close(context.Background())
		return nil, err
	}
	return s, nil
}

// The text of the page a rate limited visitor sees, in the page's language.
var tooMany = map[string]string{
	"de": "Zu viele Anfragen in kurzer Zeit. Bitte versuchen Sie es später noch einmal " +
		"oder schreiben Sie mir an simon@ostendorf.tech.",
	"en": "Too many requests in a short time. Please try again later or write to me at simon@ostendorf.tech.",
}

// pageRateLimitRespond answers a rate limited page request with a small
// HTML page in the language of its path, or of Accept-Language at /, as the
// single image did. Any other path, an asset for instance, keeps govue's
// JSON answer.
func pageRateLimitRespond(w http.ResponseWriter, r *http.Request, wait time.Duration) {
	locale, ok := pathLocale(r.URL.Path)
	if !ok && r.URL.Path == "/" {
		locale, _ = language.Negotiate(r.Header.Get("Accept-Language"), languages)
		ok = true
	}
	if !ok {
		api.WriteError(w, r, api.TooManyRequests(wait))
		return
	}
	text := tooMany[locale]
	seconds := max(int(wait.Seconds()), 1)
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Retry-After", strconv.Itoa(seconds))
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = fmt.Fprintf(w, "<!doctype html><html lang=%q><meta charset=\"utf-8\"><title>%s</title><p>%s</p>",
		locale, text, text)
}

// pathLocale is the language of a path that starts with one, /de or /de/...
func pathLocale(path string) (string, bool) {
	for _, code := range languages {
		if path == "/"+code || strings.HasPrefix(path, "/"+code+"/") {
			return code, true
		}
	}
	return "", false
}
