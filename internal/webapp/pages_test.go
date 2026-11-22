package webapp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testPageServer(t *testing.T) *pageServer {
	t.Helper()
	dist := testDist()
	prerendered := fstest.MapFS{
		"de/index.html":         &fstest.MapFile{Data: []byte("<html><body>Startseite</body></html>")},
		"de/imprint/index.html": &fstest.MapFile{Data: []byte("<html><body>Impressum</body></html>")},
		"de/privacy/index.html": &fstest.MapFile{Data: []byte("<html><body>Datenschutz</body></html>")},
		"de/terms/index.html":   &fstest.MapFile{Data: []byte("<html><body>AGB</body></html>")},
		"de/404.html":           &fstest.MapFile{Data: []byte("<html><body>Nicht gefunden</body></html>")},
		"en/index.html":         &fstest.MapFile{Data: []byte("<html><body>Home</body></html>")},
		"en/imprint/index.html": &fstest.MapFile{Data: []byte("<html><body>Imprint</body></html>")},
		"en/privacy/index.html": &fstest.MapFile{Data: []byte("<html><body>Privacy</body></html>")},
		"en/terms/index.html":   &fstest.MapFile{Data: []byte("<html><body>Terms</body></html>")},
		"en/404.html":           &fstest.MapFile{Data: []byte("<html><body>Not found</body></html>")},
	}
	p, err := newPageServer(dist, prerendered)
	if err != nil {
		t.Fatalf("newPageServer() error = %v", err)
	}
	return p
}

// testMux builds a real http.ServeMux the same way production code does:
// one pattern per locale and route slug via registerRoutes, plus the
// catch-all pageHandler under "/". Tests dispatch through it rather than
// calling pageHandler or routeHandler directly, so they observe the
// actual matched pattern and the actual route interception a bare
// handler call cannot show.
func testMux(t *testing.T, p *pageServer) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	identity := func(h http.Handler) http.Handler { return h }
	registerRoutes(mux, p, identity)
	mux.Handle("/", p.pageHandler())
	return mux
}

func TestPageHandlerServesKnownRoutes(t *testing.T) {
	p := testPageServer(t)
	mux := testMux(t, p)

	tests := map[string]struct {
		path        string
		wantStatus  int
		wantBody    string
		wantPattern string
	}{
		"german home":     {"/de/", http.StatusOK, "Startseite", "GET /de/{$}"},
		"german imprint":  {"/de/imprint", http.StatusOK, "Impressum", "GET /de/imprint"},
		"german privacy":  {"/de/privacy", http.StatusOK, "Datenschutz", "GET /de/privacy"},
		"german terms":    {"/de/terms", http.StatusOK, "AGB", "GET /de/terms"},
		"english home":    {"/en/", http.StatusOK, "Home", "GET /en/{$}"},
		"english imprint": {"/en/imprint", http.StatusOK, "Imprint", "GET /en/imprint"},
		"english privacy": {"/en/privacy", http.StatusOK, "Privacy", "GET /en/privacy"},
		"english terms":   {"/en/terms", http.StatusOK, "Terms", "GET /en/terms"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			body, _ := io.ReadAll(rec.Body)
			if !strings.Contains(string(body), tc.wantBody) {
				t.Errorf("body = %q, want it to contain %q", body, tc.wantBody)
			}
			if req.Pattern != tc.wantPattern {
				t.Errorf("matched pattern = %q, want %q", req.Pattern, tc.wantPattern)
			}
		})
	}
}

func TestPageHandlerUnknownRouteThroughMuxServesLocalNotFound(t *testing.T) {
	p := testPageServer(t)
	mux := testMux(t, p)

	tests := map[string]struct {
		path     string
		wantBody string
	}{
		"unknown german route":  {"/de/kontakt/", "Nicht gefunden"},
		"unknown english route": {"/en/contact/", "Not found"},
		"unknown locale":        {"/fr/", "Nicht gefunden"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
			body, _ := io.ReadAll(rec.Body)
			if !strings.Contains(string(body), tc.wantBody) {
				t.Errorf("body = %q, want it to contain %q", body, tc.wantBody)
			}
			if req.Pattern != "/" {
				t.Errorf("matched pattern = %q, want the catch-all %q", req.Pattern, "/")
			}
		})
	}
}

func TestPageHandlerRedirectsRoot(t *testing.T) {
	tests := map[string]struct {
		acceptLanguage string
		wantLocation   string
	}{
		"no header defaults to german":    {"", "/de/"},
		"german preference":               {"de-DE,de;q=0.9", "/de/"},
		"english preference":              {"en-US,en;q=0.9", "/en/"},
		"english ranked above german":     {"de;q=0.5,en;q=0.9", "/en/"},
		"unsupported language falls back": {"fr-FR,fr;q=0.9", "/de/"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p := testPageServer(t)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			if tc.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tc.acceptLanguage)
			}
			rec := httptest.NewRecorder()
			p.pageHandler().ServeHTTP(rec, req)

			if rec.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
			}
			if got := rec.Header().Get("Location"); got != tc.wantLocation {
				t.Errorf("Location = %q, want %q", got, tc.wantLocation)
			}
			// The target depends on Accept-Language, so a cache must key on it.
			if got := rec.Header().Get("Vary"); got != "Accept-Language" {
				t.Errorf("Vary = %q, want %q", got, "Accept-Language")
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
				t.Errorf("Cache-Control = %q, want %q", got, "no-cache")
			}
		})
	}
}

func TestPageHandlerCanonicalizesTrailingSlash(t *testing.T) {
	p := testPageServer(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/kontakt", nil)
	rec := httptest.NewRecorder()
	p.pageHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if got := rec.Header().Get("Location"); got != "/de/kontakt/" {
		t.Errorf("Location = %q, want /de/kontakt/", got)
	}
}

// TestLegalPagesRedirectTheirOldTrailingSlashForm pins the other direction:
// the legal pages are served without a trailing slash, so the old form
// redirects to it, and the former /agb/ path lands on the terms page.
func TestLegalPagesRedirectTheirOldTrailingSlashForm(t *testing.T) {
	p := testPageServer(t)
	mux := testMux(t, p)

	tests := map[string]struct {
		path         string
		wantLocation string
	}{
		"imprint":              {"/de/imprint/", "/de/imprint"},
		"privacy":              {"/en/privacy/", "/en/privacy"},
		"terms":                {"/en/terms/", "/en/terms"},
		"agb with slash":       {"/de/agb/", "/de/terms"},
		"agb without slash":    {"/de/agb", "/de/terms"},
		"query is kept":        {"/de/imprint/?utm_source=x", "/de/imprint?utm_source=x"},
		"english agb legacy":   {"/en/agb/", "/en/terms"},
		"german agb with data": {"/de/agb?a=b", "/de/terms?a=b"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusMovedPermanently {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
			}
			if got := rec.Header().Get("Location"); got != tc.wantLocation {
				t.Errorf("Location = %q, want %q", got, tc.wantLocation)
			}
		})
	}
}

func TestPageHandlerCanonicalizesTrailingSlashPreservesQuery(t *testing.T) {
	p := testPageServer(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/kontakt?utm_source=x", nil)
	rec := httptest.NewRecorder()
	p.pageHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if got := rec.Header().Get("Location"); got != "/de/kontakt/?utm_source=x" {
		t.Errorf("Location = %q, want /de/kontakt/?utm_source=x", got)
	}
}

func TestPageHandlerUnknownRouteServesLocalNotFound(t *testing.T) {
	p := testPageServer(t)

	tests := map[string]struct {
		path     string
		wantBody string
	}{
		"unknown german route":  {"/de/kontakt/", "Nicht gefunden"},
		"unknown english route": {"/en/contact/", "Not found"},
		"unknown locale":        {"/fr/", "Nicht gefunden"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			p.pageHandler().ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
			body, _ := io.ReadAll(rec.Body)
			if !strings.Contains(string(body), tc.wantBody) {
				t.Errorf("body = %q, want it to contain %q", body, tc.wantBody)
			}
		})
	}
}

func TestPageHandlerRejectsUnsafeMethods(t *testing.T) {
	p := testPageServer(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/de/", nil)
	rec := httptest.NewRecorder()
	p.pageHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestAssetHandlerServesAndCachesImmutably(t *testing.T) {
	p := testPageServer(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()
	p.assetHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q, want immutable", got)
	}
}

// TestNewPageServerFailsWhenDistHasNoAssetsDirectory reproduces the
// defect fs.Sub itself cannot catch: it builds a filesystem rooted at a
// given directory lazily, with no check that the directory exists, for
// every fs.FS that does not implement fs.SubFS itself, embed.FS and
// fstest.MapFS included. A bare fs.Sub(dist, "assets") therefore returns
// no error for a dist carrying no assets directory at all, and the
// absence only surfaces later, as an empty or 404 asset response, far
// from where newPageServer's own doc comment says it should be caught.
func TestNewPageServerFailsWhenDistHasNoAssetsDirectory(t *testing.T) {
	dist := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
	}
	prerendered := fstest.MapFS{
		"de/index.html": &fstest.MapFile{Data: []byte("<html><body>Startseite</body></html>")},
	}
	if _, err := newPageServer(dist, prerendered); err == nil {
		t.Fatal("newPageServer() error = nil, want an error naming the missing assets directory")
	}
}

func TestSplitLocale(t *testing.T) {
	tests := map[string]struct {
		path       string
		wantLocale string
		wantRest   string
		wantOK     bool
	}{
		"bare locale":        {"/de", "de", "", true},
		"locale with slash":  {"/de/", "de", "", true},
		"locale with route":  {"/en/imprint/", "en", "imprint/", true},
		"unknown locale":     {"/fr/", "", "", false},
		"root":               {"/", "", "", false},
		"locale-like prefix": {"/development/", "", "", false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			locale, rest, ok := splitLocale(tc.path)
			if ok != tc.wantOK || locale != tc.wantLocale || rest != tc.wantRest {
				t.Errorf("splitLocale(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tc.path, locale, rest, ok, tc.wantLocale, tc.wantRest, tc.wantOK)
			}
		})
	}
}
