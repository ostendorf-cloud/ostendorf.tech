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
	tests := map[string]map[string]string{
		"kubernetes": {
			"OPENBAO_ADDR":     "https://openbao.example.test",
			"OPENBAO_ROLE":     "feed",
			"OPENBAO_JWT_FILE": "/var/run/secrets/kubernetes.io/serviceaccount/token",
		},
		"token": {
			"OPENBAO_ADDR":        "http://127.0.0.1:8200",
			"OPENBAO_AUTH_METHOD": "token",
			"OPENBAO_TOKEN":       "s.fake-cmd-feed-token",
		},
	}
	for name, values := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := feedconfig.Load(func(key string) string { return values[key] })
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
		})
	}
}
