package webapp

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// legalApp is the full App with the legal pages prerendered, so a request
// goes through the real handler chain and the real mux.
func legalApp(t *testing.T) *App {
	t.Helper()
	cfg := testConfig(t, map[string]string{
		"RATE_LIMIT_BURST": "100",
		"RATE_LIMIT_PER":   "1s",
	})
	dist, prerendered := testFS()
	for _, locale := range locales {
		for _, slug := range []string{"imprint", "privacy", "terms"} {
			prerendered[locale+"/"+slug+"/index.html"] = &fstest.MapFile{Data: []byte("<html><body>" + locale + " " + slug + "</body></html>")}
		}
	}
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return app
}

// TestAppLegalRoutesRedirectToTheLocalePage pins the govue legal routes as
// the site mounts them: the short paths follow the visitor's language, the
// German aliases always go to the German page, every answer is a 302 that
// is never cached, and every target answers 200.
func TestAppLegalRoutesRedirectToTheLocalePage(t *testing.T) {
	app := legalApp(t)

	tests := map[string]struct {
		path           string
		acceptLanguage string
		wantLocation   string
	}{
		"imprint german visitor":       {"/imprint", "de-DE,de;q=0.9", "/de/imprint"},
		"imprint english visitor":      {"/imprint", "en-US,en;q=0.9", "/en/imprint"},
		"imprint without header":       {"/imprint", "", "/de/imprint"},
		"privacy english visitor":      {"/privacy", "en", "/en/privacy"},
		"terms english visitor":        {"/terms", "en", "/en/terms"},
		"impressum is german":          {"/impressum", "en", "/de/imprint"},
		"datenschutz is german":        {"/datenschutz", "en", "/de/privacy"},
		"datenschutzerklaerung german": {"/datenschutzerklaerung", "en", "/de/privacy"},
		"agb is german":                {"/agb", "en", "/de/terms"},
		"trailing slash impressum":     {"/impressum/", "en", "/de/imprint"},
		"trailing slash privacy":       {"/privacy/", "en", "/en/privacy"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			if tc.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tc.acceptLanguage)
			}
			rec := httptest.NewRecorder()
			app.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
			}
			if got := rec.Header().Get("Location"); got != tc.wantLocation {
				t.Errorf("Location = %q, want %q", got, tc.wantLocation)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store", got)
			}

			follow := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.wantLocation, nil)
			followRec := httptest.NewRecorder()
			app.Handler().ServeHTTP(followRec, follow)
			if followRec.Code != http.StatusOK {
				t.Errorf("GET %s status = %d, want %d", tc.wantLocation, followRec.Code, http.StatusOK)
			}
		})
	}
}

func TestAppLegalRoutesRefuseOtherMethods(t *testing.T) {
	app := legalApp(t)
	for _, path := range []string{"/imprint", "/impressum", "/agb"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s status = %d, want %d", path, rec.Code, http.StatusMethodNotAllowed)
		}
	}
}
