package frontendconfig

import (
	"strings"
	"testing"
	"time"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ListenAddr != DefaultListenAddr || cfg.AdminListenAddr != DefaultAdminListenAddr {
		t.Errorf("listeners = %q/%q", cfg.ListenAddr, cfg.AdminListenAddr)
	}
	if cfg.SiteURL.String() != DefaultSiteURL {
		t.Errorf("SiteURL = %v", cfg.SiteURL)
	}
	if cfg.RateLimitBurst != DefaultRateLimitBurst || cfg.RateLimitPer != DefaultRateLimitPer {
		t.Errorf("rate limit = %d/%v", cfg.RateLimitBurst, cfg.RateLimitPer)
	}
	if cfg.Umami.Enabled() {
		t.Error("Umami enabled by default")
	}
	if cfg.Development() || cfg.DevAPIProxy != "" {
		t.Error("development is on by default, want off: production never sets it")
	}
}

// TestDevelopmentFollowsTheProxySwitchOnly proves Development is true
// exactly when FRONTEND_DEV_API_PROXY is set, and no other variable turns
// it on.
func TestDevelopmentFollowsTheProxySwitchOnly(t *testing.T) {
	for _, name := range []string{"DEVELOPMENT", "DEV_API_PROXY", "ALLOW_PLAIN_HTTP"} {
		cfg, err := Load(env(map[string]string{name: "true"}))
		if err != nil {
			t.Fatalf("Load(%s) error = %v", name, err)
		}
		if cfg.Development() {
			t.Errorf("%s turned development on", name)
		}
	}
	cfg, err := Load(env(map[string]string{"FRONTEND_DEV_API_PROXY": "http://api:8080"}))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Development() || cfg.DevAPIProxy != "http://api:8080" {
		t.Errorf("Development() = %v, DevAPIProxy = %q", cfg.Development(), cfg.DevAPIProxy)
	}
}

// validUmamiWebsiteID is the lowercase UUID shape Umami names a website by.
const validUmamiWebsiteID = "0f1e2d3c-4b5a-4968-8776-655443322110"

func TestLoadUmami(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"UMAMI_URL":        "http://umami.umami.svc.cluster.local/",
		"UMAMI_WEBSITE_ID": validUmamiWebsiteID,
		"UMAMI_TIMEOUT":    "2s",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Umami.Enabled() || cfg.Umami.URL != "http://umami.umami.svc.cluster.local" ||
		cfg.Umami.WebsiteID != validUmamiWebsiteID || cfg.Umami.Timeout != 2*time.Second {
		t.Errorf("Umami = %+v", cfg.Umami)
	}
}

func TestLoadRejectsInvalidUmamiPairs(t *testing.T) {
	for name, values := range map[string]map[string]string{
		"URL alone":       {"UMAMI_URL": "http://umami"},
		"ID alone":        {"UMAMI_WEBSITE_ID": validUmamiWebsiteID},
		"plain http host": {"UMAMI_URL": "http://umami.example.test", "UMAMI_WEBSITE_ID": validUmamiWebsiteID},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Load(env(values))
			if err == nil || !strings.Contains(err.Error(), "UMAMI_") {
				t.Errorf("Load() error = %v, want one naming the UMAMI_ variable", err)
			}
		})
	}
}

func TestLoadAggregatesEveryProblem(t *testing.T) {
	_, err := Load(env(map[string]string{
		"LOG_LEVEL":       "verbose",
		"TRUSTED_PROXIES": "nope",
		"ADMIN_ADDR":      "",
		"SITE_URL":        "not a url",
	}))
	if err == nil {
		t.Fatal("Load() = nil error")
	}
	for _, want := range []string{"LOG_LEVEL", "TRUSTED_PROXIES", "SITE_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %s", err, want)
		}
	}
}
