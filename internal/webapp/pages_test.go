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

func TestPageHandlerServesKnownRoutes(t *testing.T) {
	p := testPageServer(t)
	handler := p.pageHandler()

	tests := map[string]struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		"german home":   {"/de/", http.StatusOK, "Startseite"},
		"german agb":    {"/de/agb/", http.StatusOK, "AGB"},
		"english home":  {"/en/", http.StatusOK, "Home"},
		"english terms": {"/en/agb/", http.StatusOK, "Terms"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			body, _ := io.ReadAll(rec.Body)
			if !strings.Contains(string(body), tc.wantBody) {
				t.Errorf("body = %q, want it to contain %q", body, tc.wantBody)
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
