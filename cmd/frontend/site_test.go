package main

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ostendorf-cloud/govue/frontend"
	"github.com/ostendorf-cloud/govue/themeboot"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/embedfs"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/frontendconfig"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/headertest"
)

const (
	testWebsiteID = "0f1e2d3c-4b5a-4968-8776-655443322110"
	visitorUA     = "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0"
	// visitorAddr is a documentation address that counts as public.
	visitorAddr = "203.0.113.7:4000"
)

// build is the real embedded build: the output "task web:build" and "task
// prerender" wrote. Every test here runs against it.
func build(t *testing.T) fs.FS {
	t.Helper()
	b, err := embedfs.Subtree(distFS, "dist")
	if err != nil {
		t.Fatalf("embedded build: %v; run task web:build and task prerender first", err)
	}
	return b
}

func loadConfig(t *testing.T, env map[string]string) frontendconfig.Config {
	t.Helper()
	merged := map[string]string{"RATE_LIMIT_BURST": "100000", "RATE_LIMIT_PER": "1s"}
	for k, v := range env {
		merged[k] = v
	}
	cfg, err := frontendconfig.Load(func(k string) string { return merged[k] })
	if err != nil {
		t.Fatalf("frontendconfig.Load() error = %v", err)
	}
	return cfg
}

func server(t *testing.T, env map[string]string, o options) *frontend.Server {
	t.Helper()
	s, err := newServer(loadConfig(t, env), build(t), slog.New(slog.DiscardHandler), o)
	if err != nil {
		t.Fatalf("newServer() error = %v", err)
	}
	return s
}

func visit(t *testing.T, h http.Handler, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	req.RemoteAddr = visitorAddr
	req.Header.Set("User-Agent", visitorUA)
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestRealBuildStartsTheFrontend feeds the real prerender output to
// govue's frontend.New: a page with its own icon links, theme-color tag or
// boot script, a missing 404 page or legal page, or a bad asset name would
// make it refuse to start.
func TestRealBuildStartsTheFrontend(t *testing.T) {
	server(t, nil, options{})
}

// TestPageHeadersEqualTheFormerImage proves pages and the 404 page carry the
// security headers the single image sent, with only the nonce and the boot
// script hash differing, and that the hash in the policy is govue's for the
// options this site passes.
func TestPageHeadersEqualTheFormerImage(t *testing.T) {
	h := server(t, nil, options{}).Handler()
	wantHash := themeboot.Hash(themeBootOptions()...)
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/de/", http.StatusOK},
		{"/en/", http.StatusOK},
		{"/de/imprint", http.StatusOK},
		{"/en/privacy", http.StatusOK},
		{"/nope", http.StatusNotFound},
		{"/de/nope", http.StatusNotFound},
	} {
		rec := visit(t, h, tc.path, nil)
		if rec.Code != tc.status {
			t.Fatalf("GET %s status = %d, want %d", tc.path, rec.Code, tc.status)
		}
		headertest.Equal(t, tc.path, rec.Header())
		if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
			t.Errorf("GET %s Cache-Control = %q, want no-cache", tc.path, got)
		}
		if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
			t.Errorf("GET %s Content-Type = %q", tc.path, got)
		}
		if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, wantHash) {
			t.Errorf("GET %s policy %q lacks the boot script hash %q", tc.path, csp, wantHash)
		}
	}
}

// TestPlainHTTPDropsOnlyHSTS proves ALLOW_PLAIN_HTTP removes only
// Strict-Transport-Security, as it did on the single image.
func TestPlainHTTPDropsOnlyHSTS(t *testing.T) {
	h := server(t, map[string]string{"ALLOW_PLAIN_HTTP": "true"}, options{}).Handler()
	rec := visit(t, h, "/de/", nil)
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Error("Strict-Transport-Security is set with ALLOW_PLAIN_HTTP")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" || rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("other security headers are missing")
	}
}

// TestRedirects proves the former paths keep working: the AGB pages answer
// 301 to the terms page of their language with the query kept, the German
// aliases and the language neutral legal paths answer govue's 302, the root
// sends to a language, and the slash forms are canonicalised.
func TestRedirects(t *testing.T) {
	h := server(t, nil, options{}).Handler()
	for _, tc := range []struct {
		path, location string
		status         int
	}{
		{"/de/agb", "/de/terms", http.StatusMovedPermanently},
		{"/de/agb/", "/de/terms", http.StatusMovedPermanently},
		{"/en/agb", "/en/terms", http.StatusMovedPermanently},
		{"/en/agb/?ref=old", "/en/terms?ref=old", http.StatusMovedPermanently},
		{"/de", "/de/", http.StatusMovedPermanently},
		{"/de/imprint/", "/de/imprint", http.StatusMovedPermanently},
		{"/impressum", "/de/imprint", http.StatusFound},
		{"/datenschutz", "/de/privacy", http.StatusFound},
		{"/datenschutzerklaerung", "/de/privacy", http.StatusFound},
		{"/agb", "/de/terms", http.StatusFound},
		{"/terms", "/de/terms", http.StatusFound},
		{"/", "/de/", http.StatusFound},
	} {
		rec := visit(t, h, tc.path, http.Header{"Accept-Language": {"de"}})
		if rec.Code != tc.status || rec.Header().Get("Location") != tc.location {
			t.Errorf("GET %s = %d %q, want %d %q", tc.path, rec.Code, rec.Header().Get("Location"),
				tc.status, tc.location)
		}
		headertest.Equal(t, tc.path, rec.Header())
	}
	if rec := visit(t, h, "/", http.Header{"Accept-Language": {"en-GB,en;q=0.9"}}); rec.Header().Get("Location") != "/en/" {
		t.Errorf("GET / with English = %q, want /en/", rec.Header().Get("Location"))
	}
}

// TestPublicFiles proves the favicon files, the social preview image,
// robots.txt and the sitemap are served, and the sitemap names the build
// origin.
func TestPublicFiles(t *testing.T) {
	h := server(t, nil, options{}).Handler()
	for path, contentType := range map[string]string{
		"/favicon.svg":          "image/svg+xml",
		"/favicon-32x32.png":    "image/png",
		"/apple-touch-icon.png": "image/png",
		"/og-image.png":         "image/png",
		"/robots.txt":           "text/plain; charset=utf-8",
		"/sitemap.xml":          "application/xml; charset=utf-8",
	} {
		rec := visit(t, h, path, nil)
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != contentType {
			t.Errorf("GET %s = %d %q, want 200 %q", path, rec.Code, rec.Header().Get("Content-Type"), contentType)
		}
	}
	robots := visit(t, h, "/robots.txt", nil).Body.String()
	if !strings.Contains(robots, "Sitemap: https://ostendorf.tech/sitemap.xml") {
		t.Errorf("robots.txt = %q, want the production sitemap", robots)
	}
}

// TestPageRateLimitAnswersHTMLInTheVisitorsLanguage proves the page rate
// limit answers a small page in the path's language, and keeps govue's JSON
// for an asset.
func TestPageRateLimitAnswersHTMLInTheVisitorsLanguage(t *testing.T) {
	h := server(t, map[string]string{"RATE_LIMIT_BURST": "1", "RATE_LIMIT_PER": "1m"}, options{}).Handler()
	visit(t, h, "/en/", nil)
	rec := visit(t, h, "/en/", nil)
	if rec.Code != http.StatusTooManyRequests || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("second page request = %d %q, want 429 HTML", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), tooMany["en"]) || rec.Header().Get("Retry-After") == "" {
		t.Errorf("body = %q, want the English text and Retry-After", rec.Body.String())
	}
	if rec := visit(t, h, "/favicon.svg", nil); strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Errorf("a rate limited file answered HTML")
	}
}

// fakeUmami records every body posted to it and opens no socket.
type fakeUmami struct {
	mu    sync.Mutex
	posts []string
}

func (f *fakeUmami) RoundTrip(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	f.mu.Lock()
	f.posts = append(f.posts, string(body))
	f.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK, Header: http.Header{},
		Body: io.NopCloser(strings.NewReader("")), Request: req,
	}, nil
}

// urls are the page URLs of every view posted.
func (f *fakeUmami) urls(t *testing.T) []string {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, p := range f.posts {
		var event struct {
			Payload struct {
				URL      string `json:"url"`
				Hostname string `json:"hostname"`
			} `json:"payload"`
		}
		if err := json.Unmarshal([]byte(p), &event); err != nil {
			t.Fatalf("decode view %q: %v", p, err)
		}
		if event.Payload.Hostname != "ostendorf.tech" {
			t.Errorf("view hostname = %q, want ostendorf.tech", event.Payload.Hostname)
		}
		out = append(out, event.Payload.URL)
	}
	return out
}

// TestAnalyticsCountsOnlyPages proves page views are counted for a page
// answered with 200 or revalidated with 304, by its pattern and without the
// query, and never for a redirect, an alias, a 404, an asset, a public file,
// robots.txt or the sitemap.
func TestAnalyticsCountsOnlyPages(t *testing.T) {
	fake := &fakeUmami{}
	s := server(t, map[string]string{
		"UMAMI_URL":        "https://umami.example.test",
		"UMAMI_WEBSITE_ID": testWebsiteID,
	}, options{umamiHTTP: &http.Client{Transport: fake}})
	h := s.Handler()

	page := visit(t, h, "/de/imprint?utm_source=x", nil)
	visit(t, h, "/en/", nil)
	visit(t, h, "/de/imprint", http.Header{"If-None-Match": {page.Header().Get("ETag")}})
	for _, path := range []string{
		"/de/agb", "/impressum", "/de", "/", "/nope", "/de/nope",
		"/robots.txt", "/sitemap.xml", "/favicon.svg", "/og-image.png", "/api/v1/instagram",
	} {
		visit(t, h, path, http.Header{"Accept-Language": {"de"}})
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	// The client posts from several workers, so the order is not fixed.
	got := fake.urls(t)
	slices.Sort(got)
	want := []string{"/de/imprint", "/de/imprint", "/en/"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("counted %q, want %q", got, want)
	}
}

// TestDevProxyOffByDefault proves /api/ is the 404 of production without
// FRONTEND_DEV_API_PROXY: nothing is forwarded, whatever else is set.
func TestDevProxyOffByDefault(t *testing.T) {
	transport := &recordingTransport{}
	h := server(t, map[string]string{"ALLOW_PLAIN_HTTP": "true", "DEVELOPMENT": "true"},
		options{devAPITransport: transport}).Handler()
	rec := visit(t, h, "/api/v1/instagram", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /api/v1/instagram status = %d, want 404", rec.Code)
	}
	if transport.calls() != 0 {
		t.Errorf("the API was called %d times, want none", transport.calls())
	}
}

// TestDevProxyForwardsTheAPI proves the development switch forwards /api/
// to the API container, so a compose stack has one browser origin.
func TestDevProxyForwardsTheAPI(t *testing.T) {
	transport := &recordingTransport{}
	h := server(t, map[string]string{"FRONTEND_DEV_API_PROXY": "http://api:8080"},
		options{devAPITransport: transport}).Handler()
	rec := visit(t, h, "/api/v1/instagram", nil)
	if rec.Code != http.StatusOK || transport.calls() != 1 {
		t.Fatalf("GET /api/v1/instagram = %d with %d calls, want 200 and one", rec.Code, transport.calls())
	}
	if got := transport.lastURL(); got != "http://api:8080/api/v1/instagram" {
		t.Errorf("forwarded to %q", got)
	}
}

type recordingTransport struct {
	mu   sync.Mutex
	urls []string
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.urls = append(r.urls, req.URL.String())
	r.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"posts":[]}`)), Request: req,
	}, nil
}

func (r *recordingTransport) calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.urls)
}

func (r *recordingTransport) lastURL() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.urls[len(r.urls)-1]
}

// TestFrontendHoldsNoSecret proves the frontend image reads no secret: its
// configuration and this command read no _FILE twin, no key, no SMTP, no
// OpenBao and no feed variable.
func TestFrontendHoldsNoSecret(t *testing.T) {
	forbidden := regexp.MustCompile(`SecretOrFile|_FILE"|CHALLENGE_KEY|SMTP_|OPENBAO|FEED_SERVICE|CONTACT_`)
	for _, dir := range []string{".", "../../internal/frontendconfig"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			src, err := os.ReadFile(dir + "/" + e.Name())
			if err != nil {
				t.Fatal(err)
			}
			if m := forbidden.Find(src); m != nil {
				t.Errorf("%s/%s reads %q", dir, e.Name(), m)
			}
		}
	}
}

// TestThemeColorsMatchTheTheme proves the two browser bar colours are the
// two ends of --color-bg in the theme.
func TestThemeColorsMatchTheTheme(t *testing.T) {
	css, err := os.ReadFile("../../web/src/theme/vtech.css")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`--color-bg:\s*light-dark\(\s*(#[0-9a-fA-F]{6})\s*,\s*(#[0-9a-fA-F]{6})\s*\)`).FindSubmatch(css)
	if m == nil {
		t.Fatal("vtech.css declares no --color-bg as light-dark(<hex>, <hex>)")
	}
	if string(m[1]) != themeColorLight || string(m[2]) != themeColorDark {
		t.Errorf("theme colours %s/%s, want the theme's %s/%s", themeColorLight, themeColorDark, m[1], m[2])
	}
	page := visit(t, server(t, nil, options{}).Handler(), "/de/", nil).Body.String()
	for _, tag := range []string{
		`<meta name="theme-color" content="` + themeColorLight + `" media="(prefers-color-scheme: light)">`,
		`<meta name="theme-color" content="` + themeColorDark + `" media="(prefers-color-scheme: dark)">`,
	} {
		if !strings.Contains(page, tag) {
			t.Errorf("page lacks %s", tag)
		}
	}
}

// TestThemeBootKeepsTheDarkDefault is the seam for the site's dark default.
// It is skipped until govue v0.9.0: then themeBootOptions returns
// themeboot.WithDefaultMode("dark") and this test drops its t.Skip.
func TestThemeBootKeepsTheDarkDefault(t *testing.T) {
	t.Skip("waits for govue v0.9.0 themeboot.WithDefaultMode (govue#254): " +
		"without it govue's boot script turns a first visit into the system mode")
	if len(themeBootOptions()) == 0 {
		t.Fatal("themeBootOptions passes no default mode, the dark default is lost on a first visit")
	}
}
