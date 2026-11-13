package main

import (
	"log/slog"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/feed"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/feedconfig"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// TestConfigBuildsAWorkingApp proves feedconfig.Load's result is exactly
// what feed.New expects, without starting a server or binding a port: a
// change to either package that breaks the other fails here instead of at
// a real deployment.
func TestConfigBuildsAWorkingApp(t *testing.T) {
	cfg, err := feedconfig.Load(func(key string) string {
		switch key {
		case "OPENBAO_ADDR":
			return "https://openbao.example.test"
		case "OPENBAO_ROLE":
			return "feed"
		case "OPENBAO_JWT_FILE":
			return "/var/run/secrets/kubernetes.io/serviceaccount/token"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("feedconfig.Load() error = %v", err)
	}

	app := feed.New(cfg, discardLogger())
	if app.Handler() == nil {
		t.Error("app.Handler() = nil, want a working handler")
	}
	if app.Health() == nil {
		t.Error("app.Health() = nil, want a working health registry")
	}
}
