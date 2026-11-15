package main

import (
	"embed"
	"fmt"
	"io/fs"
)

// dist is the built SPA, written by "task web:build" (vite build, with
// outDir pointed at this directory). embed resolves relative to this
// file, which is why the directory sits under cmd/ rather than internal/;
// see AGENTS.md.
//
//go:embed dist
var distFS embed.FS

// prerendered is the static HTML for every route of every locale, written
// by "task prerender" after dist exists.
//
//go:embed prerendered
var prerenderedFS embed.FS

// embeddedSubtree returns the subtree of fsys rooted at dir.
//
// fs.Sub alone would not catch a missing dir: it builds a filesystem
// rooted there lazily, with no check that it exists, for every fs.FS that
// does not implement fs.SubFS itself, embed.FS included. It returns no
// error at all for a directory that is not there and only fails on the
// first read against the result, which in this program means the first
// request for a page or an asset, far from where the real problem is.
// fs.Stat against the original filesystem is what actually does the
// validating, the same shape internal/webapp's newPageServer applies to
// "assets" within dist.
func embeddedSubtree(fsys fs.FS, dir string) (fs.FS, error) {
	if _, err := fs.Stat(fsys, dir); err != nil {
		return nil, fmt.Errorf("embedded %s: %w", dir, err)
	}
	return fs.Sub(fsys, dir)
}
