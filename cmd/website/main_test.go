package main

import (
	"io/fs"
	"log/slog"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/webapp"
)

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
