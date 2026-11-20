package main

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/govue/favicon"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
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

// TestPrerenderedPagesCarryTheSocialImage checks that every page links the
// preview image with an absolute URL built from the configured site URL,
// for og:image and twitter:image, and that the file is served from dist.
func TestPrerenderedPagesCarryTheSocialImage(t *testing.T) {
	dist, prerendered := realFS(t)
	want := config.DefaultSiteURL + "/og-image.png"
	for _, tag := range []string{
		`<meta property="og:image" content="` + want + `">`,
		`<meta name="twitter:image" content="` + want + `">`,
	} {
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
			if !strings.Contains(string(data), tag) {
				t.Errorf("%s does not carry %s", p, tag)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("fs.WalkDir(prerendered) error = %v", err)
		}
		if pages == 0 {
			t.Fatal("prerendered has no pages; run task prerender first")
		}
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
