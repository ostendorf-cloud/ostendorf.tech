package main

import (
	"bytes"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/govue/favicon"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/webapp"
)

func realFS(t *testing.T) (dist, prerendered fs.FS) {
	t.Helper()
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		t.Fatalf("fs.Sub(distFS, \"dist\") error = %v", err)
	}
	prerendered, err = fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub(prerenderedFS, \"prerendered\") error = %v", err)
	}
	return dist, prerendered
}

// TestPrerenderedPagesCarryTheFavicon runs govue's own favicon check over
// the real output of "task prerender" in both languages: the SVG icon, the
// PNG fallback and the touch icon are linked from the head of every page,
// and each linked file exists in the real dist, is not empty and has the
// right signature.
func TestPrerenderedPagesCarryTheFavicon(t *testing.T) {
	dist, prerendered := realFS(t)
	if err := favicon.CheckPages(prerendered, dist); err != nil {
		t.Fatalf("favicon.CheckPages() error = %v", err)
	}
}

// TestAppServesTheLinkedIconsFromTheEmbeddedDist requests each linked icon
// and the preview image through the real handler built from the real
// embedded dist, and compares the answer with the embedded file.
func TestAppServesTheLinkedIconsFromTheEmbeddedDist(t *testing.T) {
	dist, prerendered := realFS(t)
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

	files := map[string]string{
		"/" + favicon.SVGFile:   "image/svg+xml",
		"/" + favicon.PNGFile:   "image/png",
		"/" + favicon.TouchFile: "image/png",
		"/og-image.png":         "image/png",
	}
	for path, wantType := range files {
		t.Run(path, func(t *testing.T) {
			want, err := fs.ReadFile(dist, path[1:])
			if err != nil {
				t.Fatalf("fs.ReadFile(dist, %q) error = %v", path[1:], err)
			}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			app.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Content-Type"); got != wantType {
				t.Errorf("Content-Type = %q, want %q", got, wantType)
			}
			if got := rec.Header().Get("Cache-Control"); got != "public, max-age=86400" {
				t.Errorf("Cache-Control = %q, want one day", got)
			}
			if !bytes.Equal(rec.Body.Bytes(), want) {
				t.Errorf("body differs from the embedded %s (%d bytes, want %d)", path, rec.Body.Len(), len(want))
			}
		})
	}
}

// TestPrerenderedPagesCarryTheSocialImage checks that every page links the
// preview image with an absolute URL built from the configured site URL,
// for og:image and twitter:image, and that the file is served from dist.
func TestPrerenderedPagesCarryTheSocialImage(t *testing.T) {
	dist, prerendered := realFS(t)
	want := config.DefaultSiteURL + "/og-image.png"
	alts := map[string]string{
		"de": "Simon Ostendorf Veranstaltungstechnik: das Fader-Zeichen und der Schriftzug auf dunklem Hintergrund",
		"en": "Simon Ostendorf Event Technology: the fader mark and the wordmark on a dark background",
	}
	pages := 0
	err := fs.WalkDir(prerendered, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".html") {
			return nil
		}
		data, err := fs.ReadFile(prerendered, p)
		if err != nil {
			return err
		}
		pages++
		locale, _, _ := strings.Cut(p, "/")
		alt, ok := alts[locale]
		if !ok {
			t.Fatalf("%s is in no known locale directory", p)
		}
		for _, tag := range []string{
			`<meta property="og:image" content="` + want + `">`,
			`<meta property="og:image:width" content="1200">`,
			`<meta property="og:image:height" content="630">`,
			`<meta property="og:image:alt" content="` + alt + `">`,
			`<meta name="twitter:image" content="` + want + `">`,
			`<meta name="twitter:image:alt" content="` + alt + `">`,
		} {
			if !strings.Contains(string(data), tag) {
				t.Errorf("%s does not carry %s", p, tag)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("fs.WalkDir(prerendered) error = %v", err)
	}
	if pages == 0 {
		t.Fatal("prerendered has no pages; run task prerender first")
	}

	data, err := fs.ReadFile(dist, "og-image.png")
	if err != nil {
		t.Fatalf("og-image.png missing from dist: %v", err)
	}
	// IHDR width and height sit at bytes 16 to 24, big endian.
	if len(data) < 24 || string(data[1:4]) != "PNG" {
		t.Fatal("og-image.png is not a PNG")
	}
	w := int(data[16])<<24 | int(data[17])<<16 | int(data[18])<<8 | int(data[19])
	h := int(data[20])<<24 | int(data[21])<<16 | int(data[22])<<8 | int(data[23])
	if w != 1200 || h != 630 {
		t.Errorf("og-image.png is %dx%d, want 1200x630", w, h)
	}
}
