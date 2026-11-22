package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/webapp"
)

func TestCloseResult(t *testing.T) {
	other := errors.New("boom")
	tests := map[string]struct {
		err      error
		want     error
		wantWarn bool
	}{
		"nil":               {},
		"deadline exceeded": {err: context.DeadlineExceeded, wantWarn: true},
		"canceled":          {err: context.Canceled, wantWarn: true},
		"wrapped deadline":  {err: fmt.Errorf("close: %w", context.DeadlineExceeded), wantWarn: true},
		"other error":       {err: other, want: other},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			got := closeResult(logger, tc.err)
			if !errors.Is(got, tc.want) || (tc.want == nil && got != nil) {
				t.Errorf("closeResult() = %v, want %v", got, tc.want)
			}
			if warned := strings.Contains(buf.String(), `"level":"WARN"`); warned != tc.wantWarn {
				t.Errorf("warn logged = %v, want %v (log %q)", warned, tc.wantWarn, buf.String())
			}
		})
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// TestEmbeddedAssetsBuildAWorkingApp proves the embedded dist and
// prerendered filesystems are shaped the way webapp.New expects, without
// starting a server or binding a port: a build with an empty or
// mis-shaped frontend build fails here instead of at a request.
func TestEmbeddedAssetsBuildAWorkingApp(t *testing.T) {
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		t.Fatalf("fs.Sub(distFS, \"dist\") error = %v", err)
	}
	prerendered, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub(prerenderedFS, \"prerendered\") error = %v", err)
	}

	cfg, err := config.Load(func(string) string { return "" })
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	if _, err := webapp.New(cfg, discardLogger(), dist, prerendered); err != nil {
		t.Fatalf("webapp.New() error = %v", err)
	}
}

// TestContentSecurityPolicyAllowsTheThemeBootScript proves the wiring in
// internal/webapp's New, not just that it compiles: a response from the
// real app, built from the real embedded dist and prerendered output, has
// to carry a script-src source naming the theme boot script's hash, or
// the inline script web/scripts/prerender.ts writes into every page's
// <head> (see internal/webapp/themeboot.go) would be silently blocked by
// this site's own Content-Security-Policy, and a returning visitor's
// stored light preference would flash dark again with nothing here to
// catch it.
func TestContentSecurityPolicyAllowsTheThemeBootScript(t *testing.T) {
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		t.Fatalf("fs.Sub(distFS, \"dist\") error = %v", err)
	}
	prerendered, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		t.Fatalf("fs.Sub(prerenderedFS, \"prerendered\") error = %v", err)
	}
	cfg, err := config.Load(func(string) string { return "" })
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	app, err := webapp.New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("webapp.New() error = %v", err)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src") || !strings.Contains(csp, "'sha256-") {
		t.Errorf("Content-Security-Policy = %q, want a script-src source naming a sha256 hash", csp)
	}
}
