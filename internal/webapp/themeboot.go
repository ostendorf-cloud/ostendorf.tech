package webapp

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

// themeBootScriptOpen and themeBootScriptClose bracket the inline script
// web/scripts/prerender.ts writes into the <head> of every prerendered
// page, so a visitor's stored theme choice corrects data-mode (and the
// theme-color meta) before first paint, before main.ts's deferred
// useTheme call ever runs. See web/scripts/prerender.ts's own comment for
// what the script does.
//
// The site's Content-Security-Policy carries no 'unsafe-inline' for
// script-src, and the prerendered HTML is served byte for byte with no
// per-request templating (pageServer.serveFile), so there is no per-
// response nonce this static script could carry. The one way CSP lets an
// inline script with fixed, build-time-known text run is a 'sha256-'
// source naming its exact text. themeBootScriptHash computes that hash by
// reading the script straight out of the real prerendered output this
// package already embeds, rather than keeping a second copy of its text
// in Go that could silently drift from what prerender.ts actually writes.
const (
	themeBootScriptOpen  = `<script id="theme-boot">`
	themeBootScriptClose = `</script>`
)

// themeBootScriptHash returns the base64 standard encoded SHA-256 digest
// CSP's script-src 'sha256-' source expects, computed over the first
// <script id="theme-boot"> tag's exact text found while walking
// prerendered. Every prerendered page carries the same byte-for-byte
// script (web/scripts/prerender.ts builds it once, with no per-page
// interpolation), so finding it on any one page is enough.
func themeBootScriptHash(prerendered fs.FS) (string, error) {
	script, err := firstThemeBootScript(prerendered)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(script))
	return base64.StdEncoding.EncodeToString(sum[:]), nil
}

// firstThemeBootScript walks prerendered and returns the inner text of the
// first <script id="theme-boot"> tag it finds in any .html file.
func firstThemeBootScript(prerendered fs.FS) (string, error) {
	var script string
	found := false
	err := fs.WalkDir(prerendered, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if found || d.IsDir() || !strings.HasSuffix(p, ".html") {
			return nil
		}
		data, err := fs.ReadFile(prerendered, p)
		if err != nil {
			return err
		}
		if s, ok := extractThemeBootScript(string(data)); ok {
			script = s
			found = true
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("webapp: walk prerendered for the theme boot script: %w", err)
	}
	if !found {
		return "", errors.New("webapp: no prerendered page carries the theme boot script")
	}
	return script, nil
}

// extractThemeBootScript returns the text between the theme boot script's
// open and close tags in one page's HTML, and whether it was found at all.
func extractThemeBootScript(html string) (string, bool) {
	start := strings.Index(html, themeBootScriptOpen)
	if start == -1 {
		return "", false
	}
	start += len(themeBootScriptOpen)
	rest := html[start:]
	end := strings.Index(rest, themeBootScriptClose)
	if end == -1 {
		return "", false
	}
	return rest[:end], true
}
