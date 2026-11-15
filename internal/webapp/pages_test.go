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
	dist := fstest.MapFS{
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('app')")},
	}
	prerendered := fstest.MapFS{
		"de/index.html":         &fstest.MapFile{Data: []byte("<html><body>Startseite</body></html>")},
		"de/imprint/index.html": &fstest.MapFile{Data: []byte("<html><body>Impressum</body></html>")},
		"de/privacy/index.html": &fstest.MapFile{Data: []byte("<html><body>Datenschutz</body></html>")},
		"de/agb/index.html":     &fstest.MapFile{Data: []byte("<html><body>AGB</body></html>")},
		"de/404.html":           &fstest.MapFile{Data: []byte("<html><body>Nicht gefunden</body></html>")},
		"en/index.html":         &fstest.MapFile{Data: []byte("<html><body>Home</body></html>")},
		"en/imprint/index.html": &fstest.MapFile{Data: []byte("<html><body>Imprint</body></html>")},
		"en/privacy/index.html": &fstest.MapFile{Data: []byte("<html><body>Privacy</body></html>")},
		"en/agb/index.html":     &fstest.MapFile{Data: []byte("<html><body>Terms</body></html>")},
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
//
// track is the page view hook; a test that only cares about serving
// passes nil, which is also what a deployment without Umami uses.
func testMux(t *testing.T, p *pageServer, track func(*http.Request, string)) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	identity := func(h http.Handler) http.Handler { return h }
	registerRoutes(mux, p, identity, track)
	mux.Handle("/", p.pageHandler())
	return mux
}

func TestPageHandlerServesKnownRoutes(t *testing.T) {
	p := testPageServer(t)
	mux := testMux(t, p, nil)

	tests := map[string]struct {
		path        string
		wantStatus  int
		wantBody    string
		wantPattern string
	}{
		"german home":     {"/de/", http.StatusOK, "Startseite", "GET /de/{$}"},
		"german imprint":  {"/de/imprint/", http.StatusOK, "Impressum", "GET /de/imprint/{$}"},
		"german privacy":  {"/de/privacy/", http.StatusOK, "Datenschutz", "GET /de/privacy/{$}"},
		"german agb":      {"/de/agb/", http.StatusOK, "AGB", "GET /de/agb/{$}"},
		"english home":    {"/en/", http.StatusOK, "Home", "GET /en/{$}"},
		"english imprint": {"/en/imprint/", http.StatusOK, "Imprint", "GET /en/imprint/{$}"},
		"english privacy": {"/en/privacy/", http.StatusOK, "Privacy", "GET /en/privacy/{$}"},
		"english agb":     {"/en/agb/", http.StatusOK, "Terms", "GET /en/agb/{$}"},
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
	mux := testMux(t, p, nil)

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
		})
	}
}

func TestPageHandlerCanonicalizesTrailingSlash(t *testing.T) {
	p := testPageServer(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/imprint", nil)
	rec := httptest.NewRecorder()
	p.pageHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if got := rec.Header().Get("Location"); got != "/de/imprint/" {
		t.Errorf("Location = %q, want /de/imprint/", got)
	}
}

func TestPageHandlerCanonicalizesTrailingSlashPreservesQuery(t *testing.T) {
	p := testPageServer(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/imprint?utm_source=x", nil)
	rec := httptest.NewRecorder()
	p.pageHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if got := rec.Header().Get("Location"); got != "/de/imprint/?utm_source=x" {
		t.Errorf("Location = %q, want /de/imprint/?utm_source=x", got)
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

// trackedView is one call of the page view hook, as a test observes it.
type trackedView struct {
	path  string
	title string
}

// recordingTracker is the page view hook as a test fake: it records what
// the route handler reports instead of sending anything anywhere.
type recordingTracker struct {
	views []trackedView
}

func (r *recordingTracker) track(req *http.Request, title string) {
	r.views = append(r.views, trackedView{path: req.URL.Path, title: title})
}

// titledPageServer serves prerendered pages carrying the <title> element
// web/scripts/prerender.ts really writes, so the title a page view reports
// is read from the same place a visitor's browser tab reads it.
func titledPageServer(t *testing.T) *pageServer {
	t.Helper()
	dist := fstest.MapFS{"assets/app.js": &fstest.MapFile{Data: []byte("console.log('app')")}}
	prerendered := fstest.MapFS{
		"de/index.html":         &fstest.MapFile{Data: []byte("<html><head><title>Simon Ostendorf &#8211; Veranstaltungstechnik</title></head><body>Startseite</body></html>")},
		"de/imprint/index.html": &fstest.MapFile{Data: []byte("<html><head><title>Impressum – Simon Ostendorf</title></head><body>Impressum</body></html>")},
		// No title at all: a page view still counts, with none.
		"de/privacy/index.html": &fstest.MapFile{Data: []byte("<html><body>Datenschutz</body></html>")},
		// Malformed: an opening tag that never closes.
		"de/agb/index.html": &fstest.MapFile{Data: []byte("<html><head><title>AGB</head><body>AGB</body></html>")},
		"de/404.html":       &fstest.MapFile{Data: []byte("<html><body>Nicht gefunden</body></html>")},
		"en/index.html":     &fstest.MapFile{Data: []byte("<html><head><title>Simon Ostendorf &#8211; Event technology</title></head><body>Home</body></html>")},
		"en/404.html":       &fstest.MapFile{Data: []byte("<html><body>Not found</body></html>")},
	}
	p, err := newPageServer(dist, prerendered)
	if err != nil {
		t.Fatalf("newPageServer() error = %v", err)
	}
	return p
}

func TestPageTitlesAreReadFromThePrerenderedHTML(t *testing.T) {
	p := titledPageServer(t)

	tests := map[string]struct {
		file string
		want string
	}{
		// prerender.ts escapes the title it writes; the entity is read
		// back as the character a visitor's tab shows.
		"escaped entity is decoded": {"de/index.html", "Simon Ostendorf – Veranstaltungstechnik"},
		"plain title":               {"de/imprint/index.html", "Impressum – Simon Ostendorf"},
		"english title":             {"en/index.html", "Simon Ostendorf – Event technology"},
		"page without a title":      {"de/privacy/index.html", ""},
		"unclosed title element":    {"de/agb/index.html", ""},
		// Never prerendered at all in this fixture.
		"missing file": {"en/imprint/index.html", ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := p.titles[tc.file]; got != tc.want {
				t.Errorf("title of %s = %q, want %q", tc.file, got, tc.want)
			}
		})
	}
}

func TestPageViewIsTrackedWithItsTitle(t *testing.T) {
	p := titledPageServer(t)
	tracker := &recordingTracker{}
	mux := testMux(t, p, tracker.track)

	for _, path := range []string{"/de/", "/de/imprint/", "/de/privacy/"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d", path, rec.Code, http.StatusOK)
		}
	}

	want := []trackedView{
		{path: "/de/", title: "Simon Ostendorf – Veranstaltungstechnik"},
		{path: "/de/imprint/", title: "Impressum – Simon Ostendorf"},
		// A page whose HTML carries no title is still a page view; only
		// the title is missing, which Umami then leaves out.
		{path: "/de/privacy/", title: ""},
	}
	if len(tracker.views) != len(want) {
		t.Fatalf("tracked %+v, want %+v", tracker.views, want)
	}
	for i, view := range tracker.views {
		if view != want[i] {
			t.Errorf("view %d = %+v, want %+v", i, view, want[i])
		}
	}
}

func TestOnlyARealPageViewIsTracked(t *testing.T) {
	// A redirect, a 404 and a page whose file never got prerendered are
	// not page views: the visitor never received that page.
	p := titledPageServer(t)
	tracker := &recordingTracker{}
	mux := testMux(t, p, tracker.track)

	for _, path := range []string{
		"/",            // locale redirect
		"/de/imprint",  // trailing slash redirect
		"/de/kontakt/", // unknown route, 404
		"/fr/",         // unknown locale, 404
		"/en/imprint/", // known route whose file is missing, falls back to 404
	} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		mux.ServeHTTP(httptest.NewRecorder(), req)
	}

	if len(tracker.views) != 0 {
		t.Errorf("tracked %+v, want nothing", tracker.views)
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
