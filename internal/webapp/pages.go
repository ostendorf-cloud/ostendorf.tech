package webapp

import (
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/embedfs"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/routes"
)

// locales are the only two the site serves. German is the source of truth
// and the default; see preferredLocale for how a visitor without one yet
// is sent to one of them.
var locales = []string{"de", "en"}

// routeFiles maps a route slug, the path segment after the locale, to the
// prerendered file that answers it. The empty slug is the locale's home
// page. Every value is relative to one locale's own directory.
//
// Built from internal/routes' embedded route table, the single list
// web/scripts/prerender.ts also reads to decide what it writes: adding a
// page means editing that one file, routes.json, not this map. A
// malformed routes.json is a build bug, caught by internal/routes' own
// tests before it ever reaches this package; see mustRouteFiles.
var routeFiles = mustRouteFiles()

// mustRouteFiles loads the embedded route table and indexes it by slug.
// A load failure here can only mean routes.json itself is malformed: it
// is checked in, not a runtime input, so treating that the way
// contact.germanTime treats a failed zoneinfo load (a panic, since
// package webapp cannot start correctly either way) is the right
// response rather than threading an error through every caller of this
// package level table.
func mustRouteFiles() map[string]string {
	table, err := routes.Load()
	if err != nil {
		panic("webapp: " + err.Error())
	}
	files := make(map[string]string, len(table))
	for _, r := range table {
		files[r.Slug] = r.File
	}
	return files
}

// routeSlugs lists routeFiles' keys in a deterministic, sorted order, so
// registering one http.ServeMux pattern per slug never depends on Go's
// unspecified map iteration order.
func routeSlugs() []string {
	slugs := make([]string, 0, len(routeFiles))
	for slug := range routeFiles {
		slugs = append(slugs, slug)
	}
	slices.Sort(slugs)
	return slugs
}

// routePattern builds the http.ServeMux pattern one locale and route slug
// register under, with {$} anchoring to the exact end of the path so the
// pattern never matches a child path or a request missing its trailing
// slash.
//
// The locale is a literal, not a wildcard segment: a generic {locale}
// wildcard here would syntactically conflict with the existing
// "GET /assets/" registration, because net/http's ServeMux rejects two
// patterns at registration time unless one matches a strict subset of the
// other's requests (see "Precedence" in `go doc net/http ServeMux`), and
// neither "GET /{locale}/{$}" nor "GET /assets/" is a subset of the other
// ("GET /{locale}/{$}" also matches "/assets/" itself, with "assets"
// bound to {locale}; "GET /assets/" also matches "/assets/app.js", which
// "GET /{locale}/{$}" does not). Registering one literal pattern per
// known locale avoids that ambiguity entirely, since "de" and "en" never
// overlap with any other top-level route this mux serves, and still
// yields one distinct, attributable pattern per page per locale.
func routePattern(locale, slug string) string {
	if slug == "" {
		return "GET /" + locale + "/{$}"
	}
	return "GET /" + locale + "/" + slug + "/{$}"
}

// registerRoutes registers one http.ServeMux pattern per known locale and
// route slug on mux, each handler wrapped by wrap (the middleware chain a
// caller wants applied, route in production). It is the one place that
// loops over locales and routeSlugs, so a test exercising the real
// registration never has to hand-duplicate a route pattern string.
//
// track is called once per page actually served, with that page's own
// title, and is how a page view reaches Umami (see (*App).trackPageView).
// A nil track counts nothing, which is both the default for a deployment
// with no Umami configured and what a test that cares about serving alone
// passes.
func registerRoutes(mux *http.ServeMux, p *pageServer, wrap func(http.Handler) http.Handler, track func(*http.Request, string)) {
	for _, locale := range locales {
		for _, slug := range routeSlugs() {
			mux.Handle(routePattern(locale, slug), wrap(p.routeHandler(locale, slug, track)))
		}
	}
}

// pageServer answers every document and static asset request out of the
// built SPA and the prerendered HTML, both read only filesystems.
type pageServer struct {
	assets      fs.FS
	prerendered fs.FS
	// titles is each prerendered file's own <title>, read once at
	// startup, keyed the same way serveFile names a file. It exists for
	// the page view reported to Umami, which the privacy policy says
	// carries "aufgerufene Seite (ohne angehängte Parameter) und ihr
	// Titel": this is that title, the exact text the visitor's own tab
	// shows, rather than a second copy of the i18n catalogue in Go.
	titles map[string]string
}

// newPageServer validates that dist carries the assets/ directory the SPA
// build writes, so a missing build is reported once at startup rather than
// as a 404 on the first request. embedfs.Subtree is what actually does the
// validating; see its own doc comment for why a bare fs.Sub would not
// catch this.
func newPageServer(dist, prerendered fs.FS) (*pageServer, error) {
	assets, err := embedfs.Subtree(dist, "assets")
	if err != nil {
		return nil, fmt.Errorf("webapp: dist has no assets directory: %w", err)
	}
	return &pageServer{assets: assets, prerendered: prerendered, titles: readTitles(prerendered)}, nil
}

// readTitles reads every known route's prerendered <title> once. A file
// that cannot be read, or carries no title, simply has none here: a page
// view then reports an empty title, which Umami leaves out of the record
// entirely. Nothing about this may fail startup or a request, because a
// statistic is never worth a page.
func readTitles(prerendered fs.FS) map[string]string {
	titles := make(map[string]string, len(locales)*len(routeFiles))
	for _, locale := range locales {
		for _, slug := range routeSlugs() {
			file := locale + "/" + routeFiles[slug]
			data, err := fs.ReadFile(prerendered, file)
			if err != nil {
				continue
			}
			if title := TitleOf(data); title != "" {
				titles[file] = title
			}
		}
	}
	return titles
}

// TitleOf is the text of the one <title> element web/scripts/prerender.ts
// writes into every page it generates. It is a plain cut rather than an
// HTML parse because the input is this repository's own build output, not
// anything a visitor sends, and prerender.ts escapes the title before
// writing it, which html.UnescapeString here reverses.
func TitleOf(data []byte) string {
	_, rest, ok := strings.Cut(string(data), "<title>")
	if !ok {
		return ""
	}
	title, _, ok := strings.Cut(rest, "</title>")
	if !ok {
		return ""
	}
	return html.UnescapeString(title)
}

// assetHandler serves the SPA's built, content hashed files as immutable:
// the filename changes with the content, so a cached copy can never go
// stale.
func (p *pageServer) assetHandler() http.Handler {
	fileServer := http.FileServerFS(p.assets)
	return http.StripPrefix("/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	}))
}

// routeHandler answers one known route slug for one locale, registered
// once at startup under routePattern(locale, slug) so routeFiles[slug]
// can be captured directly: the pattern itself pins both the locale and
// the slug literally, so no per-request lookup or validation is needed.
//
// track, when set, is called after the page was written, so only a page a
// visitor actually received is ever counted: no redirect, no 404, no
// asset and no API call reaches it, the same rule
// old_simon-ostendorf.de's own page() applied.
func (p *pageServer) routeHandler(locale, slug string, track func(*http.Request, string)) http.Handler {
	file := locale + "/" + routeFiles[slug]
	title := p.titles[file]
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served := p.serveFile(w, http.StatusOK, file)
		if served && track != nil {
			track(r, title)
		}
	})
}

// pageHandler is the catch-all document handler: the locale redirect at
// "/", trailing slash canonicalisation, and the locale's own 404 page for
// anything else. Serving a known route's prerendered HTML is handled by
// the per-slug patterns routeHandler registers; those intercept every
// known route in real serving, so this handler only ever reaches the 404
// branch for a genuinely unknown slug.
func (p *pageServer) pageHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := r.URL.Path
		if path == "/" {
			locale := preferredLocale(r.Header.Get("Accept-Language"))
			http.Redirect(w, r, "/"+locale+"/", http.StatusFound)
			return
		}

		locale, _, ok := splitLocale(path)
		if !ok {
			p.serveNotFound(w, defaultLocale)
			return
		}
		if !strings.HasSuffix(path, "/") {
			target := path + "/"
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}

		p.serveNotFound(w, locale)
	})
}

// serveFile writes the prerendered file at name, or falls back to the
// locale's own 404 page when it is missing: a route this handler believes
// exists but whose file never got prerendered must still answer something
// rather than crash. It reports whether the requested file itself was
// served, so a caller counting page views counts the page and not the
// fallback.
func (p *pageServer) serveFile(w http.ResponseWriter, status int, name string) bool {
	data, err := fs.ReadFile(p.prerendered, name)
	if err != nil {
		locale, _, _ := strings.Cut(name, "/")
		p.serveNotFound(w, locale)
		return false
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	w.WriteHeader(status)
	_, _ = w.Write(data)
	return true
}

// serveNotFound answers 404 with the given locale's own prerendered page,
// falling back to a plain text line when even that cannot be read.
func (p *pageServer) serveNotFound(w http.ResponseWriter, locale string) {
	data, err := fs.ReadFile(p.prerendered, locale+"/404.html")
	if err != nil {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(data)
}

// defaultLocale is German, the source of truth for this site's content.
const defaultLocale = "de"

// splitLocale reports the locale a path starts with and the remainder
// after it, or false when the path does not start with a known locale.
func splitLocale(path string) (locale, rest string, ok bool) {
	for _, candidate := range locales {
		prefix := "/" + candidate
		if path == prefix {
			return candidate, "", true
		}
		if rest, ok := strings.CutPrefix(path, prefix+"/"); ok {
			return candidate, rest, true
		}
	}
	return "", "", false
}

// preferredLocale picks a locale from an Accept-Language header: English
// when the visitor's highest weighted language is English, German
// otherwise, which covers both a missing header and a language this site
// has no translation for. German is the deliberate fallback: it is this
// site's source of truth, the same rule applied to a missing translation
// key within a page.
func preferredLocale(header string) string {
	best, bestWeight := "", -1.0
	for _, entry := range strings.Split(header, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		tag, weight := entry, 1.0
		if name, params, found := strings.Cut(entry, ";"); found {
			tag = strings.TrimSpace(name)
			for _, param := range strings.Split(params, ";") {
				if value, ok := strings.CutPrefix(strings.TrimSpace(param), "q="); ok {
					if parsed, err := strconv.ParseFloat(value, 64); err == nil {
						weight = parsed
					}
				}
			}
		}
		if weight > bestWeight {
			best, bestWeight = tag, weight
		}
	}

	primary, _, _ := strings.Cut(best, "-")
	if strings.EqualFold(primary, "en") {
		return "en"
	}
	return defaultLocale
}
