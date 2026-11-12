package webapp

import (
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
)

// locales are the only two the site serves. German is the source of truth
// and the default; see preferredLocale for how a visitor without one yet
// is sent to one of them.
var locales = []string{"de", "en"}

// routeFiles maps a route slug, the path segment after the locale, to the
// prerendered file that answers it. The empty slug is the locale's home
// page. Every value is relative to one locale's own directory.
var routeFiles = map[string]string{
	"":        "index.html",
	"imprint": "imprint/index.html",
	"privacy": "privacy/index.html",
	"agb":     "agb/index.html",
}

// pageServer answers every document and static asset request out of the
// built SPA and the prerendered HTML, both read only filesystems.
type pageServer struct {
	assets      fs.FS
	prerendered fs.FS
}

// newPageServer validates that dist carries the assets/ directory the SPA
// build writes, so a missing build is reported once at startup rather than
// as a 404 on the first request.
func newPageServer(dist, prerendered fs.FS) (*pageServer, error) {
	assets, err := fs.Sub(dist, "assets")
	if err != nil {
		return nil, fmt.Errorf("webapp: dist has no assets directory: %w", err)
	}
	return &pageServer{assets: assets, prerendered: prerendered}, nil
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

// pageHandler is the catch-all document handler: the locale redirect at
// "/", trailing slash canonicalisation, and serving the prerendered HTML
// for every known route, or the locale's own 404 page for anything else.
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

		locale, rest, ok := splitLocale(path)
		if !ok {
			p.serveNotFound(w, defaultLocale)
			return
		}
		if !strings.HasSuffix(path, "/") {
			http.Redirect(w, r, path+"/", http.StatusMovedPermanently)
			return
		}

		slug := strings.Trim(rest, "/")
		file, ok := routeFiles[slug]
		if !ok {
			p.serveNotFound(w, locale)
			return
		}
		p.serveFile(w, http.StatusOK, locale+"/"+file)
	})
}

// serveFile writes the prerendered file at name, or falls back to the
// locale's own 404 page when it is missing: a route this handler believes
// exists but whose file never got prerendered must still answer something
// rather than crash.
func (p *pageServer) serveFile(w http.ResponseWriter, status int, name string) {
	data, err := fs.ReadFile(p.prerendered, name)
	if err != nil {
		locale, _, _ := strings.Cut(name, "/")
		p.serveNotFound(w, locale)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	w.WriteHeader(status)
	_, _ = w.Write(data)
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
