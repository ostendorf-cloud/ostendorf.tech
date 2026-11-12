package config

import (
	"log/slog"
	"os"
	"path/filepath"
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

// validHexKey is a 32 byte key, hex encoded: 64 hex characters, long enough
// for CSRF_KEY and CHALLENGE_KEY.
var validHexKey = strings.Repeat("ab", minKeyBytes)

// withCSRFKey adds a valid CSRF_KEY to values, since Load fails without
// one: most tests about something else do not want that failure mixed in.
func withCSRFKey(values map[string]string) map[string]string {
	merged := map[string]string{"CSRF_KEY": validHexKey}
	for k, v := range values {
		merged[k] = v
	}
	return merged
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(withCSRFKey(nil)))
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
	if cfg.AllowPlainHTTP {
		t.Error("AllowPlainHTTP = true, want false by default")
	}
	if cfg.SMTPAddr != DefaultSMTPAddr {
		t.Errorf("SMTPAddr = %q, want %q", cfg.SMTPAddr, DefaultSMTPAddr)
	}
	if cfg.SMTPUsername != "" || cfg.SMTPPassword != "" {
		t.Errorf("SMTP credentials = %q/%q, want both empty by default", cfg.SMTPUsername, cfg.SMTPPassword)
	}
	if cfg.ContactTo == nil || cfg.ContactTo.Address != "simon@ostendorf.tech" {
		t.Errorf("ContactTo = %v, want simon@ostendorf.tech", cfg.ContactTo)
	}
	if cfg.ContactFrom == nil || cfg.ContactFrom.Address != "no-reply@ostendorf.cloud" {
		t.Errorf("ContactFrom = %v, want no-reply@ostendorf.cloud", cfg.ContactFrom)
	}
	if cfg.ChallengeKey != nil {
		t.Errorf("ChallengeKey = %v, want nil when CHALLENGE_KEY is unset", cfg.ChallengeKey)
	}
	if len(cfg.CSRFKey) != minKeyBytes {
		t.Errorf("CSRFKey = %d bytes, want %d", len(cfg.CSRFKey), minKeyBytes)
	}
	if cfg.ChallengeRateLimitBurst != DefaultChallengeRateLimitBurst || cfg.ChallengeRateLimitPer != DefaultChallengeRateLimitPer {
		t.Errorf("challenge rate limit = %d/%v, want %d/%v",
			cfg.ChallengeRateLimitBurst, cfg.ChallengeRateLimitPer, DefaultChallengeRateLimitBurst, DefaultChallengeRateLimitPer)
	}
	if cfg.ContactRateLimitBurst != DefaultContactRateLimitBurst || cfg.ContactRateLimitPer != DefaultContactRateLimitPer {
		t.Errorf("contact rate limit = %d/%v, want %d/%v",
			cfg.ContactRateLimitBurst, cfg.ContactRateLimitPer, DefaultContactRateLimitBurst, DefaultContactRateLimitPer)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(env(withCSRFKey(map[string]string{
		"LISTEN_ADDR":         ":9090",
		"SITE_URL":            "https://example.test",
		"LOG_LEVEL":           "debug",
		"TRUSTED_PROXY_CIDRS": "10.0.0.0/8, 192.168.1.1",
		"RATE_LIMIT_BURST":    "30",
		"RATE_LIMIT_PER":      "30s",
		"ALLOW_PLAIN_HTTP":    "true",
		"SMTP_ADDR":           "smtp.example.test:587",
		"SMTP_USERNAME":       "no-reply@example.test",
		"SMTP_PASSWORD":       "s3cret",
		"CONTACT_TO":          "someone@example.test",
		"CONTACT_FROM":        "Example <no-reply@example.test>",
	})))
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
	if !cfg.AllowPlainHTTP {
		t.Error("AllowPlainHTTP = false, want true")
	}
	if cfg.SMTPAddr != "smtp.example.test:587" {
		t.Errorf("SMTPAddr = %q, want smtp.example.test:587", cfg.SMTPAddr)
	}
	if cfg.SMTPUsername != "no-reply@example.test" || cfg.SMTPPassword != "s3cret" {
		t.Errorf("SMTP credentials = %q/%q", cfg.SMTPUsername, cfg.SMTPPassword)
	}
	if cfg.ContactTo.Address != "someone@example.test" {
		t.Errorf("ContactTo = %v, want someone@example.test", cfg.ContactTo)
	}
	if cfg.ContactFrom.Address != "no-reply@example.test" || cfg.ContactFrom.Name != "Example" {
		t.Errorf("ContactFrom = %v, want Example <no-reply@example.test>", cfg.ContactFrom)
	}
}

func TestLoadSecretFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "smtp-password")
	if err := os.WriteFile(path, []byte("s3cret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(env(withCSRFKey(map[string]string{
		"SMTP_PASSWORD_FILE": path,
	})))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.SMTPPassword != "s3cret" {
		t.Errorf("SMTPPassword = %q, want %q (trailing newline trimmed)", cfg.SMTPPassword, "s3cret")
	}
}

func TestLoadSecretRefusesBothVariableAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "smtp-password")
	if err := os.WriteFile(path, []byte("s3cret"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(env(withCSRFKey(map[string]string{
		"SMTP_PASSWORD":      "also-set",
		"SMTP_PASSWORD_FILE": path,
	})))
	if err == nil || !strings.Contains(err.Error(), "SMTP_PASSWORD") {
		t.Fatalf("Load() error = %v, want a problem naming SMTP_PASSWORD", err)
	}
}

func TestLoadRequiresCSRFKey(t *testing.T) {
	_, err := Load(env(nil))
	if err == nil || !strings.Contains(err.Error(), "CSRF_KEY") {
		t.Fatalf("Load() error = %v, want a problem naming CSRF_KEY", err)
	}
}

func TestLoadRejectsShortOrNonHexKeys(t *testing.T) {
	tests := map[string]string{
		"not hex":   "not-hex-at-all",
		"too short": strings.Repeat("ab", 8),
	}
	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Load(env(map[string]string{"CSRF_KEY": value}))
			if err == nil || !strings.Contains(err.Error(), "CSRF_KEY") {
				t.Fatalf("Load() error = %v, want a problem naming CSRF_KEY", err)
			}
		})
	}
}

func TestLoadChallengeKeyOptional(t *testing.T) {
	validKey := strings.Repeat("cd", minKeyBytes)
	cfg, err := Load(env(withCSRFKey(map[string]string{"CHALLENGE_KEY": validKey})))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if len(cfg.ChallengeKey) != minKeyBytes {
		t.Errorf("ChallengeKey = %d bytes, want %d", len(cfg.ChallengeKey), minKeyBytes)
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
			_, err := Load(env(withCSRFKey(tc.env)))
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
	_, err := Load(env(withCSRFKey(map[string]string{
		"SITE_URL":  "not a url",
		"LOG_LEVEL": "verbose",
	})))
	if err == nil {
		t.Fatal("Load() = nil error, want one aggregating both problems")
	}
	for _, want := range []string{"SITE_URL", "LOG_LEVEL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Load() error = %q, want it to also mention %q", err.Error(), want)
		}
	}
}
