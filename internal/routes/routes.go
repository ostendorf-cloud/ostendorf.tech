// Package routes is the website's one route table. internal/webapp's page
// handler, sitemap and per-page mux registration, web/scripts/prerender.ts's
// static renderer, and the frontend's own router (web/src/router/index.ts)
// and SEO metadata (web/src/lib/seo.ts), through the shared
// web/src/lib/routeTable.ts, each build their own list from routes.json,
// embedded here, instead of restating it: a page added to one side and not
// the other is no longer possible, because there is only one list left to
// edit. See AGENTS.md for how each side reads it, and cmd/website's own test
// for the check that the prerendered output this table describes actually
// exists.
package routes

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed routes.json
var tableJSON []byte

// Route is one page the site serves under every locale.
type Route struct {
	// Name identifies the route for a caller that needs a stable key
	// distinct from Slug, such as the frontend router's route name or its
	// SEO metadata lookup: Slug alone cannot serve that purpose, because
	// the home page's slug is empty.
	Name string `json:"name"`
	// Slug is the path segment after the locale; the home page's slug is
	// empty.
	Slug string `json:"slug"`
	// File is the prerendered file that answers Slug, relative to one
	// locale's own directory under cmd/website/prerendered.
	File string `json:"file"`
}

// Load parses the embedded route table.
func Load() ([]Route, error) {
	return Parse(tableJSON)
}

// Parse decodes data into the route table, failing on anything that would
// make the table ambiguous: an entry missing a name or a file, or two
// entries sharing a slug. A caller that only ever parses the checked-in
// routes.json should treat a failure here as a build bug, not a runtime
// one.
func Parse(data []byte) ([]Route, error) {
	var table []Route
	if err := json.Unmarshal(data, &table); err != nil {
		return nil, fmt.Errorf("routes: parse route table: %w", err)
	}

	slugs := make(map[string]bool, len(table))
	for _, r := range table {
		if r.Name == "" {
			return nil, fmt.Errorf("routes: route table: entry with slug %q has no name", r.Slug)
		}
		if r.File == "" {
			return nil, fmt.Errorf("routes: route table: route %q has no file", r.Name)
		}
		if slugs[r.Slug] {
			return nil, fmt.Errorf("routes: route table: duplicate slug %q", r.Slug)
		}
		slugs[r.Slug] = true
	}
	return table, nil
}
