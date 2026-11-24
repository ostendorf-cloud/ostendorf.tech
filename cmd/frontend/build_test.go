package main

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ostendorf-cloud/govue/favicon"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/routes"
)

// pages walks every prerendered HTML page of the real build, under the
// language directories, and calls fn with its path and markup.
func pages(t *testing.T, fn func(path, html string)) {
	t.Helper()
	b := build(t)
	n := 0
	for _, code := range languages {
		err := fs.WalkDir(b, code, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(p, ".html") {
				return nil
			}
			data, err := fs.ReadFile(b, p)
			if err != nil {
				return err
			}
			n++
			fn(p, string(data))
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", code, err)
		}
	}
	if n == 0 {
		t.Fatal("the build has no pages; run task prerender first")
	}
}

// TestPrerenderedOutputMatchesRouteTable checks the real prerender output
// against internal/routes' table: every route has a page in every language,
// and every page but the 404 page answers a route.
func TestPrerenderedOutputMatchesRouteTable(t *testing.T) {
	table, err := routes.Load()
	if err != nil {
		t.Fatalf("routes.Load() error = %v", err)
	}
	want := map[string]bool{}
	for _, r := range table {
		want[r.File] = true
	}
	b := build(t)
	for _, code := range languages {
		for _, r := range table {
			if _, err := fs.Stat(b, code+"/"+r.File); err != nil {
				t.Errorf("%s: route %q has no page %s/%s", code, r.Name, code, r.File)
			}
		}
	}
	pages(t, func(p, _ string) {
		_, file, _ := strings.Cut(p, "/")
		if file != "404.html" && !want[file] {
			t.Errorf("%s answers no route in routes.json", p)
		}
	})
}

// TestPrerenderedPagesPinTheColourMode guards the first paint: every page
// ships data-theme="vtech" and data-mode="dark" on its html element.
func TestPrerenderedPagesPinTheColourMode(t *testing.T) {
	pages(t, func(p, html string) {
		start := strings.Index(html, "<html")
		end := strings.Index(html[start:], ">")
		tag := html[start : start+end+1]
		for _, want := range []string{`data-theme="vtech"`, `data-mode="dark"`} {
			if !strings.Contains(tag, want) {
				t.Errorf("%s: <html> tag %q lacks %s", p, tag, want)
			}
		}
	})
}

// TestPrerenderedIndexingTags: a 404 page is noindex with no canonical,
// og:url or hreflang, every other page indexable with its canonical link.
func TestPrerenderedIndexingTags(t *testing.T) {
	pages(t, func(p, html string) {
		has := func(s string) bool { return strings.Contains(html, s) }
		if strings.HasSuffix(p, "404.html") {
			if !has(`<meta name="robots" content="noindex">`) {
				t.Errorf("%s: want noindex", p)
			}
			for _, bad := range []string{`rel="canonical"`, `property="og:url"`, `rel="alternate"`} {
				if has(bad) {
					t.Errorf("%s: a 404 page must not carry %s", p, bad)
				}
			}
			return
		}
		if !has(`<meta name="robots" content="index, follow">`) || !has(`rel="canonical"`) {
			t.Errorf("%s: want index, follow and a canonical link", p)
		}
	})
}

// TestPrerenderedPagesLeaveTheHeadToGovue proves the prerender writes none
// of what govue's frontend injects and refuses twice: icon links, a
// theme-color tag and a boot script of the site's own.
func TestPrerenderedPagesLeaveTheHeadToGovue(t *testing.T) {
	pages(t, func(p, html string) {
		for _, bad := range []string{`rel="icon"`, `rel="apple-touch-icon"`, `name="theme-color"`, `id="theme-boot"`} {
			if strings.Contains(html, bad) {
				t.Errorf("%s carries %s", p, bad)
			}
		}
	})
}

// TestServedPagesCarryTheFaviconAndSocialImage runs govue's
// favicon.CheckPages over the pages as the server sends them, with the
// links it injected, and checks the social preview image of every page.
func TestServedPagesCarryTheFaviconAndSocialImage(t *testing.T) {
	h := server(t, nil, options{}).Handler()
	served := fstest.MapFS{}
	for _, path := range []string{
		"/de/", "/en/", "/de/imprint", "/en/imprint", "/de/privacy",
		"/en/privacy", "/de/terms", "/en/terms",
	} {
		body := visit(t, h, path, nil).Body.String()
		served[strings.Trim(path, "/")+"/index.html"] = &fstest.MapFile{Data: []byte(body)}
		if !strings.Contains(body, `<meta property="og:image" content="https://ostendorf.tech/og-image.png">`) {
			t.Errorf("%s lacks the social preview image", path)
		}
	}
	if err := favicon.CheckPages(served, build(t)); err != nil {
		t.Errorf("favicon.CheckPages() = %v", err)
	}
}
