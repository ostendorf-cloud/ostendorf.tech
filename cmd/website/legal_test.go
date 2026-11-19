package main

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/webapp"
)

var sitemapLoc = regexp.MustCompile(`<loc>https?://[^/<]+(/[^<]*)</loc>`)

func locPaths(body string) []string {
	var paths []string
	for _, m := range sitemapLoc.FindAllStringSubmatch(body, -1) {
		paths = append(paths, m[1])
	}
	slices.Sort(paths)
	return paths
}

// TestRuntimeSitemapMatchesPrerenderedSitemap ties the two sitemaps
// together. govue-check-legal reads the sitemap.xml that "task prerender"
// writes (govue's sitemapXml over the site's public routes), while the
// server answers /sitemap.xml from internal/webapp's own list under the
// configured origin. The check is only meaningful for the served sitemap if
// both list the same paths, and every listed path has to answer 200 from
// the real embedded pages.
func TestRuntimeSitemapMatchesPrerenderedSitemap(t *testing.T) {
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		t.Fatalf("fs.Sub(distFS, \"dist\") error = %v", err)
	}
	prerendered, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub(prerenderedFS, \"prerendered\") error = %v", err)
	}
	built, err := fs.ReadFile(prerendered, "sitemap.xml")
	if err != nil {
		t.Fatalf("prerendered sitemap.xml: %v; run task prerender first", err)
	}

	cfg, err := config.Load(func(key string) string {
		if key == "CSRF_KEY" {
			return testCSRFKey
		}
		return ""
	})
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	app, err := webapp.New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("webapp.New() error = %v", err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/sitemap.xml", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /sitemap.xml status = %d, want %d", rec.Code, http.StatusOK)
	}

	want, got := locPaths(string(built)), locPaths(rec.Body.String())
	if len(want) == 0 {
		t.Fatal("the prerendered sitemap lists no path")
	}
	if !slices.Equal(want, got) {
		t.Errorf("served sitemap paths = %v, prerendered sitemap paths = %v", got, want)
	}
	for _, path := range got {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		app.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, w.Code, http.StatusOK)
		}
	}
}
