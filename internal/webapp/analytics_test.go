package webapp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ostendorf-cloud/govue/analytics"
)

const (
	testWebsiteID = "0f1e2d3c-4b5a-4968-8776-655443322110"
	testUmamiURL  = "https://umami.example.test"
	visitorUA     = "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0"
	// visitorAddr is a documentation address that counts as public.
	visitorAddr = "203.0.113.7:4000"
)

// fakeUmami is the transport standing in for Umami: it records every body
// posted to it and opens no socket. When block is set, a post waits for it
// or for the request context.
type fakeUmami struct {
	mu    sync.Mutex
	posts []string
	block chan struct{}
}

func (f *fakeUmami) RoundTrip(req *http.Request) (*http.Response, error) {
	if f.block != nil {
		select {
		case <-f.block:
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}
	body, _ := io.ReadAll(req.Body)
	f.mu.Lock()
	f.posts = append(f.posts, string(body))
	f.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func (f *fakeUmami) bodies() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.posts...)
}

// analyticsApp is the full App with Umami configured and its transport
// replaced by fake, with every legal page prerendered, so a request goes
// through the real handler chain and the real mux.
func analyticsApp(t *testing.T, fake *fakeUmami, env map[string]string) *App {
	t.Helper()
	merged := map[string]string{
		"RATE_LIMIT_BURST": "100000",
		"RATE_LIMIT_PER":   "1s",
		"UMAMI_URL":        testUmamiURL,
		"UMAMI_WEBSITE_ID": testWebsiteID,
	}
	for k, v := range env {
		merged[k] = v
	}
	dist, prerendered := testFS()
	for _, locale := range locales {
		for _, slug := range []string{"imprint", "privacy", "terms"} {
			prerendered[locale+"/"+slug+"/index.html"] = &fstest.MapFile{
				Data: []byte("<html><body>" + locale + " " + slug + "</body></html>"),
			}
		}
	}
	app, err := New(testConfig(t, merged), discardLogger(), dist, prerendered,
		WithUmamiHTTPClient(&http.Client{Transport: fake}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return app
}

// settle sends what is queued, so fake.bodies() is final.
func settle(t *testing.T, app *App) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := app.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func visit(t *testing.T, h http.Handler, method, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
	req.RemoteAddr = visitorAddr
	req.Header.Set("User-Agent", visitorUA)
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// reportedView is what Umami receives of one page view.
type reportedView struct {
	Website   string `json:"website"`
	Hostname  string `json:"hostname"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	Referrer  string `json:"referrer"`
	Language  string `json:"language"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
}

func reportedViews(t *testing.T, bodies []string) []reportedView {
	t.Helper()
	var views []reportedView
	for _, b := range bodies {
		var ev struct {
			Payload reportedView `json:"payload"`
		}
		if err := json.Unmarshal([]byte(b), &ev); err != nil {
			t.Fatalf("Umami body %q is not JSON: %v", b, err)
		}
		views = append(views, ev.Payload)
	}
	return views
}

func TestHandlerCountsOnlyPagesAndReportsThePatternNotThePath(t *testing.T) {
	fake := &fakeUmami{}
	app := analyticsApp(t, fake, nil)
	h := app.Handler()

	// Counted: a legal page with a query string and a token in it, and the
	// home page. The referrer carries a path and a query that must not
	// leave the process.
	visit(t, h, http.MethodGet, "/de/imprint?token=s3cret&utm_source=x", http.Header{
		"Referer":         {"https://example.org/some/page?secret=1"},
		"Accept-Language": {"de-DE,de;q=0.9,en;q=0.5"},
	})
	visit(t, h, http.MethodGet, "/de/", nil)

	// Not counted.
	for name, req := range map[string]struct {
		method string
		target string
		header http.Header
	}{
		"HEAD":                     {http.MethodHead, "/de/", nil},
		"POST":                     {http.MethodPost, "/de/", nil},
		"Do Not Track":             {http.MethodGet, "/de/", http.Header{"Dnt": {"1"}}},
		"Global Privacy Control":   {http.MethodGet, "/de/", http.Header{"Sec-Gpc": {"1"}}},
		"prefetch":                 {http.MethodGet, "/de/", http.Header{"Sec-Purpose": {"prefetch"}}},
		"bot":                      {http.MethodGet, "/de/", http.Header{"User-Agent": {"Googlebot/2.1"}}},
		"no user agent":            {http.MethodGet, "/de/", http.Header{"User-Agent": {""}}},
		"static asset":             {http.MethodGet, "/assets/app.js", nil},
		"robots.txt":               {http.MethodGet, "/robots.txt", nil},
		"sitemap.xml":              {http.MethodGet, "/sitemap.xml", nil},
		"api route":                {http.MethodGet, "/api/v1/instagram", nil},
		"unknown api route":        {http.MethodGet, "/api/v1/nowhere?token=s3cret", nil},
		"unknown page 404":         {http.MethodGet, "/de/nowhere?token=s3cret", nil},
		"unknown locale 404":       {http.MethodGet, "/fr/", nil},
		"locale redirect":          {http.MethodGet, "/", nil},
		"trailing slash redirect":  {http.MethodGet, "/de/imprint/", nil},
		"former agb redirect":      {http.MethodGet, "/de/agb/", nil},
		"former agb redirect bare": {http.MethodGet, "/de/agb", nil},
		"legal alias imprint":      {http.MethodGet, "/imprint", nil},
		"legal alias privacy":      {http.MethodGet, "/privacy", nil},
		"legal alias terms":        {http.MethodGet, "/terms", nil},
		"legal alias impressum":    {http.MethodGet, "/impressum", nil},
		"legal alias datenschutz":  {http.MethodGet, "/datenschutz", nil},
		"legal alias long":         {http.MethodGet, "/datenschutzerklaerung", nil},
		"legal alias agb":          {http.MethodGet, "/agb", nil},
	} {
		t.Run(name, func(t *testing.T) {
			visit(t, h, req.method, req.target, req.header)
		})
	}

	settle(t, app)
	bodies := fake.bodies()
	got := reportedViews(t, bodies)
	if len(got) != 2 {
		t.Fatalf("Umami received %d views %v, want exactly 2", len(got), bodies)
	}
	byURL := map[string]reportedView{}
	for _, v := range got {
		byURL[v.URL] = v
	}
	imprint, ok := byURL["/de/imprint"]
	if !ok {
		t.Fatalf("reported URLs = %v, want /de/imprint among them", byURL)
	}
	if _, ok := byURL["/de/"]; !ok {
		t.Fatalf("reported URLs = %v, want /de/ among them (the {$} of the pattern cut off)", byURL)
	}

	if imprint.Website != testWebsiteID || imprint.Hostname != "ostendorf.tech" {
		t.Errorf("website/hostname = %q/%q, want %q/ostendorf.tech", imprint.Website, imprint.Hostname, testWebsiteID)
	}
	if imprint.IP != "203.0.113.0" {
		t.Errorf("ip = %q, want the address masked to its /24, 203.0.113.0", imprint.IP)
	}
	if imprint.Referrer != "https://example.org/" {
		t.Errorf("referrer = %q, want the origin only", imprint.Referrer)
	}
	if imprint.Language != "de-DE" {
		t.Errorf("language = %q, want the first tag, de-DE", imprint.Language)
	}
	if imprint.UserAgent != visitorUA {
		t.Errorf("userAgent = %q, want the visitor's", imprint.UserAgent)
	}
	if imprint.Title != "" {
		t.Errorf("title = %q, want none reported", imprint.Title)
	}

	for _, b := range bodies {
		for _, leak := range []string{"s3cret", "token", "utm_source", "some/page", "secret=", "nowhere", "?", "203.0.113.7"} {
			if strings.Contains(b, leak) {
				t.Errorf("Umami body %s carries %q", b, leak)
			}
		}
	}
}

func TestPageSetsNoCookieAndMentionsNothingOfUmami(t *testing.T) {
	// No script, no cookie, no beacon: the privacy policy's claim that
	// Umami never runs in the visitor's browser is a property of the
	// response, so it is checked on the response of a counted page.
	fake := &fakeUmami{}
	app := analyticsApp(t, fake, nil)

	rec := visit(t, app.Handler(), http.MethodGet, "/de/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "Startseite") {
		t.Errorf("body = %q, want the prerendered page", rec.Body.String())
	}
	if cookies := rec.Result().Cookies(); len(cookies) != 0 {
		t.Errorf("page set cookies %v, want none", cookies)
	}
	if strings.Contains(strings.ToLower(rec.Body.String()), "umami") {
		t.Errorf("body mentions umami: %q", rec.Body.String())
	}
	settle(t, app)
	if got := len(fake.bodies()); got != 1 {
		t.Errorf("Umami received %d views, want 1 (a counted page)", got)
	}
}

func TestHandlerCountsNothingWithoutUmami(t *testing.T) {
	fake := &fakeUmami{}
	app := analyticsApp(t, fake, map[string]string{"UMAMI_URL": "", "UMAMI_WEBSITE_ID": ""})
	if app.views.Enabled() {
		t.Fatal("views are enabled without UMAMI_URL and UMAMI_WEBSITE_ID")
	}
	if rec := visit(t, app.Handler(), http.MethodGet, "/de/", nil); rec.Code != http.StatusOK {
		t.Fatalf("GET /de/ = %d, want %d", rec.Code, http.StatusOK)
	}
	settle(t, app)
	if got := fake.bodies(); len(got) != 0 {
		t.Errorf("Umami received %v with counting off, want nothing", got)
	}
}

func TestTrustedProxyDecidesWhichAddressIsCounted(t *testing.T) {
	const (
		proxyAddr  = "10.1.2.3:5000"
		peerAddr   = "203.0.113.9:5000"
		forwarded  = "198.51.100.77"
		trustedNet = "10.0.0.0/8"
	)
	tests := map[string]struct {
		env        map[string]string
		remoteAddr string
		xff        string
		wantIP     string // empty: not counted
	}{
		"trusted proxy: the forwarded visitor is counted, masked": {
			env: map[string]string{"TRUSTED_PROXIES": trustedNet}, remoteAddr: proxyAddr, xff: forwarded, wantIP: "198.51.100.0",
		},
		"untrusted peer cannot forge the address": {
			env: map[string]string{"TRUSTED_PROXIES": trustedNet}, remoteAddr: peerAddr, xff: forwarded, wantIP: "203.0.113.0",
		},
		"trusted proxy with no header: the proxy is no visitor": {
			env: map[string]string{"TRUSTED_PROXIES": trustedNet}, remoteAddr: proxyAddr,
		},
		"no trusted proxy: a private peer counts nothing": {
			remoteAddr: proxyAddr, xff: forwarded,
		},
		"no trusted proxy: the header is ignored": {
			remoteAddr: peerAddr, xff: forwarded, wantIP: "203.0.113.0",
		},
		"forwarded private address counts nothing": {
			env: map[string]string{"TRUSTED_PROXIES": trustedNet}, remoteAddr: proxyAddr, xff: "192.168.1.5",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fake := &fakeUmami{}
			app := analyticsApp(t, fake, tc.env)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/", nil)
			req.RemoteAddr = tc.remoteAddr
			req.Header.Set("User-Agent", visitorUA)
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			rec := httptest.NewRecorder()
			app.Handler().ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("GET /de/ = %d, want %d", rec.Code, http.StatusOK)
			}
			settle(t, app)
			got := reportedViews(t, fake.bodies())
			if tc.wantIP == "" {
				if len(got) != 0 {
					t.Fatalf("Umami received %v, want nothing", fake.bodies())
				}
				return
			}
			if len(got) != 1 || got[0].IP != tc.wantIP {
				t.Fatalf("Umami received %v, want one view with ip %s", fake.bodies(), tc.wantIP)
			}
		})
	}
}

func TestHostnameSentToUmamiHasNoPort(t *testing.T) {
	fake := &fakeUmami{}
	app := analyticsApp(t, fake, map[string]string{"SITE_URL": "http://localhost:8080"})
	visit(t, app.Handler(), http.MethodGet, "/de/", nil)
	settle(t, app)
	got := reportedViews(t, fake.bodies())
	if len(got) != 1 || got[0].Hostname != "localhost" {
		t.Fatalf("Umami received %v, want one view with hostname localhost", fake.bodies())
	}
}

// droppedViews reads the counter of dropped views from /metrics.
func droppedViews(t *testing.T, app *App) string {
	t.Helper()
	rec := visit(t, app.AdminHandler(), http.MethodGet, "/metrics", nil)
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		if v, ok := strings.CutPrefix(line, viewsDroppedMetric+" "); ok {
			return v
		}
	}
	return ""
}

func TestDroppedViewsAreCounted(t *testing.T) {
	fake := &fakeUmami{block: make(chan struct{})}
	app := analyticsApp(t, fake, map[string]string{"UMAMI_QUEUE_SIZE": "1"})
	h := app.Handler()
	if got := droppedViews(t, app); got != "" && got != "0" {
		t.Fatalf("dropped counter before any view = %q, want 0", got)
	}
	for range 20 {
		visit(t, h, http.MethodGet, "/de/", nil)
	}
	if got := droppedViews(t, app); got == "" || got == "0" {
		t.Errorf("dropped counter = %q after overflowing the queue, want it above 0", got)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_ = app.Close(ctx)
}

// TestSlowUmamiNeverDelaysAResponse blocks every post to Umami and checks
// that responses still come back at once and unchanged.
func TestSlowUmamiNeverDelaysAResponse(t *testing.T) {
	fake := &fakeUmami{block: make(chan struct{})}
	app := analyticsApp(t, fake, nil)
	h := app.Handler()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// More requests than the default queue holds, so a full queue is
		// exercised too.
		for range analytics.DefaultQueueSize + 50 {
			rec := visit(t, h, http.MethodGet, "/de/", nil)
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Startseite") {
				t.Errorf("GET /de/ = %d %q, want the page unchanged", rec.Code, rec.Body.String())
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("responses were held back by a blocked Umami")
	}

	// Shutdown is bounded as well: the posts in flight are cancelled.
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	if err := app.Close(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close() error = %v", err)
	}
}
