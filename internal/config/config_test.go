package config

import (
	"log/slog"
	"strings"
	"testing"
)

// env builds a getenv function from a map, so a test can set exactly the
// variables it cares about and leave everything else unset.
func env(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.ListenAddr != DefaultListenAddr {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, DefaultListenAddr)
	}
	if cfg.SiteURL == nil || cfg.SiteURL.String() != DefaultSiteURL {
		t.Errorf("SiteURL = %v, want %q", cfg.SiteURL, DefaultSiteURL)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.RateLimitBurst != DefaultRateLimitBurst {
		t.Errorf("RateLimitBurst = %d, want %d", cfg.RateLimitBurst, DefaultRateLimitBurst)
	}
	if cfg.RateLimitPer != DefaultRateLimitPer {
		t.Errorf("RateLimitPer = %v, want %v", cfg.RateLimitPer, DefaultRateLimitPer)
	}
	if len(cfg.TrustedProxies) != 0 {
		t.Errorf("TrustedProxies = %v, want empty", cfg.TrustedProxies)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"LISTEN_ADDR":         ":9090",
		"SITE_URL":            "https://example.test",
		"LOG_LEVEL":           "debug",
		"TRUSTED_PROXY_CIDRS": "10.0.0.0/8, 192.168.1.1",
		"RATE_LIMIT_BURST":    "30",
		"RATE_LIMIT_PER":      "30s",
	}))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q, want :9090", cfg.ListenAddr)
	}
	if cfg.SiteURL.String() != "https://example.test" {
		t.Errorf("SiteURL = %v, want https://example.test", cfg.SiteURL)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want debug", cfg.LogLevel)
	}
	if len(cfg.TrustedProxies) != 2 {
		t.Errorf("TrustedProxies = %v, want 2 entries", cfg.TrustedProxies)
	}
	if cfg.RateLimitBurst != 30 {
		t.Errorf("RateLimitBurst = %d, want 30", cfg.RateLimitBurst)
	}
}

func TestLoadRejectsEachInvalidField(t *testing.T) {
	tests := map[string]struct {
		env  map[string]string
		want string
	}{
		"invalid site url": {
			env:  map[string]string{"SITE_URL": "not a url"},
			want: "SITE_URL",
		},
		"invalid log level": {
			env:  map[string]string{"LOG_LEVEL": "verbose"},
			want: "LOG_LEVEL",
		},
		"invalid trusted proxy cidr": {
			env:  map[string]string{"TRUSTED_PROXY_CIDRS": "not-a-cidr"},
			want: "TRUSTED_PROXY_CIDRS",
		},
		"rate limit burst out of range": {
			env:  map[string]string{"RATE_LIMIT_BURST": "0"},
			want: "RATE_LIMIT_BURST",
		},
		"rate limit per not a duration": {
			env:  map[string]string{"RATE_LIMIT_PER": "soon"},
			want: "RATE_LIMIT_PER",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Load(env(tc.env))
			if err == nil {
				t.Fatalf("Load() = nil error, want one mentioning %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Load() error = %q, want it to mention %q", err.Error(), tc.want)
			}
		})
	}
}

func TestLoadAggregatesEveryProblem(t *testing.T) {
	_, err := Load(env(map[string]string{
		"SITE_URL":  "not a url",
		"LOG_LEVEL": "verbose",
	}))
	if err == nil {
		t.Fatal("Load() = nil error, want one aggregating both problems")
	}
	for _, want := range []string{"SITE_URL", "LOG_LEVEL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Load() error = %q, want it to also mention %q", err.Error(), want)
		}
	}
}
