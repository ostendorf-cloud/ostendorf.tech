package main

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/routes"
)

// TestPrerenderedOutputMatchesRouteTable is the check that proves
// internal/webapp's page handler, sitemap and mux registration, and
// web/scripts/prerender.ts's static renderer, cannot drift apart any
// more. Both now read the same embedded route table (see
// internal/routes), but reading the same file does not by itself prove
// prerender.ts still writes what that table says: this test checks the
// real, generated prerendered output "task prerender" wrote against that
// same table, catching exactly the two failure modes the two-list
// version of this route table used to allow silently:
//
//   - a route the table lists with no prerendered file for it under some
//     locale (the "Go only" case: a route would register and the sitemap
//     would advertise it, but a crawler following that link would hit a
//     404);
//   - a prerendered file that answers no route the table lists (the
//     "TypeScript only" case: a page prerender.ts would write under its
//     own pattern, with nothing in internal/webapp ever serving it
//     there).
func TestPrerenderedOutputMatchesRouteTable(t *testing.T) {
	table, err := routes.Load()
	if err != nil {
		t.Fatalf("routes.Load() error = %v", err)
	}
	wantFiles := make(map[string]bool, len(table))
	for _, r := range table {
		wantFiles[r.File] = true
	}

	sub, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub(prerenderedFS, \"prerendered\") error = %v", err)
	}

	localeDirs, err := fs.ReadDir(sub, ".")
	if err != nil {
		t.Fatalf("fs.ReadDir(prerendered) error = %v", err)
	}
	if len(localeDirs) == 0 {
		t.Fatal("prerendered has no locale directories; run task prerender first")
	}

	for _, entry := range localeDirs {
		if !entry.IsDir() {
			continue
		}
		locale := entry.Name()

		for _, r := range table {
			want := locale + "/" + r.File
			if _, err := fs.Stat(sub, want); err != nil {
				t.Errorf("locale %q: route %q (slug %q) has no prerendered file at %q: %v",
					locale, r.Name, r.Slug, want, err)
			}
		}

		for file := range foundFiles(t, sub, locale) {
			if file == "404.html" {
				// Not a route: the locale's own fallback page, served
				// directly by serveNotFound rather than through the
				// route table.
				continue
			}
			if !wantFiles[file] {
				t.Errorf("locale %q: prerendered file %q answers no route in routes.json", locale, file)
			}
		}
	}
}

// TestPrerenderedPagesPinTheColourMode guards the website's first paint.
// The frontend's theme composable treats a missing data-mode attribute as
// "follow the operating system", so a prerendered page that left it off
// would paint light for a visitor whose system is set to light and then
// turn dark once the bundle loaded and applied the site's own default.
// Nothing in Go reads these two attributes, which is exactly why they are
// easy to drop: they are inert markup here and the only thing that
// decides what a visitor sees before any JavaScript runs. This test reads
// the real output "task prerender" wrote, so the attributes have to
// survive in the shipped bytes rather than only in the script that writes
// them.
func TestPrerenderedPagesPinTheColourMode(t *testing.T) {
	const (
		wantTheme = `data-theme="vtech"`
		wantMode  = `data-mode="dark"`
	)

	sub, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub(prerenderedFS, \"prerendered\") error = %v", err)
	}

	pages := 0
	err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".html") {
			return nil
		}
		pages++

		data, err := fs.ReadFile(sub, p)
		if err != nil {
			return err
		}
		tag := openingHTMLTag(string(data))
		if tag == "" {
			t.Errorf("%s has no opening <html> tag", p)
			return nil
		}
		for _, want := range []string{wantTheme, wantMode} {
			if !strings.Contains(tag, want) {
				t.Errorf("%s: <html> tag %q does not carry %s", p, tag, want)
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
}

// TestPrerenderedThemeBootScriptIsByteIdenticalAcrossPages guards the
// assumption internal/webapp's themeBootScriptHash relies on: it computes
// one CSP script-src 'sha256-' source from whichever prerendered page it
// finds the <script id="theme-boot"> tag on first, so that hash only
// covers every page's own copy of the script if every page's copy is
// exactly the same bytes. web/scripts/prerender.ts builds the script from
// two build-time constants with no per-page interpolation, which this
// test checks against the real output "task prerender" wrote, rather than
// trusting that description to still hold.
func TestPrerenderedThemeBootScriptIsByteIdenticalAcrossPages(t *testing.T) {
	sub, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub(prerenderedFS, \"prerendered\") error = %v", err)
	}

	var first string
	var firstPath string
	pages := 0
	err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".html") {
			return nil
		}
		data, err := fs.ReadFile(sub, p)
		if err != nil {
			return err
		}
		script, ok := themeBootScriptOf(string(data))
		if !ok {
			t.Errorf("%s has no <script id=\"theme-boot\"> tag", p)
			return nil
		}
		pages++
		if first == "" {
			first, firstPath = script, p
			return nil
		}
		if script != first {
			t.Errorf("%s's theme-boot script differs from %s's; the CSP hash can only cover one", p, firstPath)
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

// themeBootScriptOf returns the text between the theme boot script's open
// and close tags in one page's HTML, and whether it was found at all. A
// plain cut, like openingHTMLTag above, over the same real build output
// rather than a parse, since the exact bytes are what the CSP hash covers.
func themeBootScriptOf(html string) (string, bool) {
	const open = `<script id="theme-boot">`
	const close = `</script>`
	start := strings.Index(html, open)
	if start == -1 {
		return "", false
	}
	start += len(open)
	rest := html[start:]
	end := strings.Index(rest, close)
	if end == -1 {
		return "", false
	}
	return rest[:end], true
}

// openingHTMLTag is the document's <html ...> tag, attributes and all, or
// an empty string when there is none. It cuts on the raw markup rather
// than parsing the document, because the attributes are the thing under
// test and a parser would be free to normalise them.
func openingHTMLTag(document string) string {
	start := strings.Index(document, "<html")
	if start == -1 {
		return ""
	}
	end := strings.Index(document[start:], ">")
	if end == -1 {
		return ""
	}
	return document[start : start+end+1]
}

// foundFiles lists every file actually prerendered under locale,
// relative to that locale's own directory.
func foundFiles(t *testing.T, sub fs.FS, locale string) map[string]bool {
	t.Helper()
	found := make(map[string]bool)
	err := fs.WalkDir(sub, locale, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		found[strings.TrimPrefix(p, locale+"/")] = true
		return nil
	})
	if err != nil {
		t.Fatalf("fs.WalkDir(%q) error = %v", locale, err)
	}
	return found
}
