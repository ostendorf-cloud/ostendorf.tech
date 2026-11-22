package config

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	govuemail "github.com/ostendorf-cloud/govue/mail"
)

// env builds a getenv function from a map, so a test can set exactly the
// variables it cares about and leave everything else unset.
func env(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

// validHexKey is a 32 byte key, hex encoded: 64 hex characters, long enough
// for CHALLENGE_KEY.
var validHexKey = strings.Repeat("ab", minKeyBytes)

// TestLoadDefaults proves the website starts with no variable set at all:
// no key and no SMTP server is required, the contact form then answers 503.
func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.ListenAddr != DefaultListenAddr {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, DefaultListenAddr)
	}
	if cfg.AdminListenAddr != DefaultAdminListenAddr {
		t.Errorf("AdminListenAddr = %q, want %q", cfg.AdminListenAddr, DefaultAdminListenAddr)
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
	if cfg.SMTP != nil {
		t.Errorf("SMTP = %+v, want nil without SMTP_HOST", cfg.SMTP)
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
	if cfg.ChallengeRateLimitBurst != DefaultChallengeRateLimitBurst || cfg.ChallengeRateLimitPer != DefaultChallengeRateLimitPer {
		t.Errorf("challenge rate limit = %d/%v, want %d/%v",
			cfg.ChallengeRateLimitBurst, cfg.ChallengeRateLimitPer, DefaultChallengeRateLimitBurst, DefaultChallengeRateLimitPer)
	}
	if cfg.ContactRateLimitBurst != DefaultContactRateLimitBurst || cfg.ContactRateLimitPer != DefaultContactRateLimitPer {
		t.Errorf("contact rate limit = %d/%v, want %d/%v",
			cfg.ContactRateLimitBurst, cfg.ContactRateLimitPer, DefaultContactRateLimitBurst, DefaultContactRateLimitPer)
	}
	if cfg.FeedServiceURL != "" {
		t.Error("FeedServiceURL is set by default, want the Impressionen section switched off")
	}
	if cfg.FeedRefreshInterval != DefaultFeedRefreshInterval {
		t.Errorf("FeedRefreshInterval = %v, want %v", cfg.FeedRefreshInterval, DefaultFeedRefreshInterval)
	}
	if cfg.Umami.Enabled() || cfg.Umami.URL != "" || cfg.Umami.WebsiteID != "" {
		t.Errorf("umami = %q/%q, want both empty by default", cfg.Umami.URL, cfg.Umami.WebsiteID)
	}
}

// TestLoadKeepsTheProtectionOfTheFormerSiteDefaults pins the contact limits
// to the values the site ran with before it moved to govue's handlers: 10
// challenges per 20 seconds and 5 submissions per 10 minutes per client,
// and 50 mails an hour in total.
func TestLoadKeepsTheProtectionOfTheFormerSiteDefaults(t *testing.T) {
	if DefaultChallengeRateLimitBurst != 10 || DefaultChallengeRateLimitPer != 20*time.Second {
		t.Errorf("challenge default = %d/%v, want 10/20s", DefaultChallengeRateLimitBurst, DefaultChallengeRateLimitPer)
	}
	if DefaultContactRateLimitBurst != 5 || DefaultContactRateLimitPer != 10*time.Minute {
		t.Errorf("contact default = %d/%v, want 5/10m", DefaultContactRateLimitBurst, DefaultContactRateLimitPer)
	}
	if DefaultMailRateLimitBurst != 50 || DefaultMailRateLimitPer != time.Hour {
		t.Errorf("mail default = %d/%v, want 50/1h", DefaultMailRateLimitBurst, DefaultMailRateLimitPer)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"LISTEN_ADDR":      ":9090",
		"ADMIN_ADDR":       ":9091",
		"SITE_URL":         "https://example.test",
		"LOG_LEVEL":        "debug",
		"TRUSTED_PROXIES":  "10.0.0.0/8, 192.168.1.1",
		"RATE_LIMIT_BURST": "30",
		"RATE_LIMIT_PER":   "30s",
		"ALLOW_PLAIN_HTTP": "true",
		"SMTP_HOST":        "smtp.example.test",
		"SMTP_PORT":        "465",
		"SMTP_USERNAME":    "no-reply@example.test",
		"SMTP_PASSWORD":    "s3cret",
		"CONTACT_TO":       "someone@example.test",
		"CONTACT_FROM":     "Example <no-reply@example.test>",
	}))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q, want :9090", cfg.ListenAddr)
	}
	if cfg.AdminListenAddr != ":9091" {
		t.Errorf("AdminListenAddr = %q, want :9091", cfg.AdminListenAddr)
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
	if cfg.SMTP == nil {
		t.Fatal("SMTP = nil, want the server SMTP_HOST names")
	}
	if cfg.SMTP.Host != "smtp.example.test" || cfg.SMTP.Port != 465 || cfg.SMTP.Security != govuemail.ImplicitTLS {
		t.Errorf("SMTP = %s:%d %s, want smtp.example.test:465 implicit-tls", cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Security)
	}
	if cfg.SMTP.Username != "no-reply@example.test" || cfg.SMTP.Password.Reveal() != "s3cret" {
		t.Errorf("SMTP credentials = %q/%v", cfg.SMTP.Username, cfg.SMTP.Password)
	}
	// The EHLO name is the site's host name, not govue's localhost default.
	if cfg.SMTP.HeloName != "example.test" {
		t.Errorf("SMTP.HeloName = %q, want example.test", cfg.SMTP.HeloName)
	}
	if cfg.ContactTo.Address != "someone@example.test" {
		t.Errorf("ContactTo = %v, want someone@example.test", cfg.ContactTo)
	}
	if cfg.ContactFrom.Address != "no-reply@example.test" || cfg.ContactFrom.Name != "Example" {
		t.Errorf("ContactFrom = %v, want Example <no-reply@example.test>", cfg.ContactFrom)
	}
}

// TestLoadSMTPDefaultsToSubmissionWithStartTLS proves SMTP_PORT defaults to
// 587 with STARTTLS, the port the former SMTP_ADDR default named.
func TestLoadSMTPDefaultsToSubmissionWithStartTLS(t *testing.T) {
	cfg, err := Load(env(map[string]string{"SMTP_HOST": "mail.example.test"}))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.SMTP == nil || cfg.SMTP.Port != govuemail.DefaultPort || cfg.SMTP.Security != govuemail.StartTLS {
		t.Fatalf("SMTP = %+v, want port %d with STARTTLS", cfg.SMTP, govuemail.DefaultPort)
	}
}

func TestLoadSecretFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "smtp-password")
	if err := os.WriteFile(path, []byte("s3cret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(env(map[string]string{
		"SMTP_HOST":          "mail.example.test",
		"SMTP_USERNAME":      "no-reply@example.test",
		"SMTP_PASSWORD_FILE": path,
	}))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.SMTP.Password.Reveal() != "s3cret" {
		t.Errorf("SMTP password = %q, want %q (trailing newline trimmed)", cfg.SMTP.Password.Reveal(), "s3cret")
	}
}

func TestLoadSecretRefusesBothVariableAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "smtp-password")
	if err := os.WriteFile(path, []byte("s3cret"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(env(map[string]string{
		"SMTP_HOST":          "mail.example.test",
		"SMTP_USERNAME":      "no-reply@example.test",
		"SMTP_PASSWORD":      "also-set",
		"SMTP_PASSWORD_FILE": path,
	}))
	if err == nil || !strings.Contains(err.Error(), "SMTP_PASSWORD") {
		t.Fatalf("Load() error = %v, want a problem naming SMTP_PASSWORD", err)
	}
}

// TestLoadSecretFileMustNotBeEmpty is govue's envconf.SecretOrFile
// contract: a file resolving to an empty value is a problem, not a silent
// "not configured".
func TestLoadSecretFileMustNotBeEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "smtp-password")
	if err := os.WriteFile(path, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(env(map[string]string{
		"SMTP_HOST":          "mail.example.test",
		"SMTP_PASSWORD_FILE": path,
	}))
	if err == nil || !strings.Contains(err.Error(), "SMTP_PASSWORD") {
		t.Fatalf("Load() error = %v, want a problem naming SMTP_PASSWORD", err)
	}
}

// TestLoadRefusesAHalfConfiguredSMTPServer proves a username without a
// password is a startup error, govue's mail.Config.Validate rule, rather
// than a form that fails on the first visitor.
func TestLoadRefusesAHalfConfiguredSMTPServer(t *testing.T) {
	_, err := Load(env(map[string]string{
		"SMTP_HOST":     "mail.example.test",
		"SMTP_USERNAME": "no-reply@example.test",
	}))
	if err == nil || !strings.Contains(err.Error(), "username and password") {
		t.Fatalf("Load() error = %v, want the username and password problem", err)
	}
	if strings.Contains(err.Error(), "no-reply@example.test") {
		t.Errorf("Load() error = %q, must not name the username", err)
	}
}

func TestLoadRejectsShortOrNonHexKeys(t *testing.T) {
	tests := map[string]string{
		"not hex":   "not-hex-at-all",
		"too short": strings.Repeat("ab", 8),
	}
	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Load(env(map[string]string{"CHALLENGE_KEY": value}))
			if err == nil || !strings.Contains(err.Error(), "CHALLENGE_KEY") {
				t.Fatalf("Load() error = %v, want a problem naming CHALLENGE_KEY", err)
			}
		})
	}
}

func TestLoadChallengeKeyOptional(t *testing.T) {
	validKey := strings.Repeat("cd", minKeyBytes)
	cfg, err := Load(env(map[string]string{"CHALLENGE_KEY": validKey}))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if len(cfg.ChallengeKey) != minKeyBytes {
		t.Errorf("ChallengeKey = %d bytes, want %d", len(cfg.ChallengeKey), minKeyBytes)
	}
	if cfg.ChallengeKeyPrevious != nil {
		t.Errorf("ChallengeKeyPrevious = %v, want nil when CHALLENGE_KEY_PREVIOUS is unset", cfg.ChallengeKeyPrevious)
	}
}

// TestLoadChallengeKeyPreviousOptional proves the rotation key loads the
// same way the current one does, and independently of it: a deployment can
// carry both at once while a rotation is in flight, see the README.
func TestLoadChallengeKeyPreviousOptional(t *testing.T) {
	currentKey := strings.Repeat("cd", minKeyBytes)
	previousKey := strings.Repeat("ef", minKeyBytes)
	cfg, err := Load(env(map[string]string{
		"CHALLENGE_KEY":          currentKey,
		"CHALLENGE_KEY_PREVIOUS": previousKey,
	}))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if len(cfg.ChallengeKey) != minKeyBytes {
		t.Errorf("ChallengeKey = %d bytes, want %d", len(cfg.ChallengeKey), minKeyBytes)
	}
	if len(cfg.ChallengeKeyPrevious) != minKeyBytes {
		t.Errorf("ChallengeKeyPrevious = %d bytes, want %d", len(cfg.ChallengeKeyPrevious), minKeyBytes)
	}
	if bytes.Equal(cfg.ChallengeKey, cfg.ChallengeKeyPrevious) {
		t.Error("ChallengeKey and ChallengeKeyPrevious decoded to the same bytes, want the two distinct keys given")
	}
}

func TestLoadRejectsShortOrNonHexChallengeKeyPrevious(t *testing.T) {
	_, err := Load(env(map[string]string{"CHALLENGE_KEY_PREVIOUS": "not-hex-at-all"}))
	if err == nil || !strings.Contains(err.Error(), "CHALLENGE_KEY_PREVIOUS") {
		t.Fatalf("Load() error = %v, want a problem naming CHALLENGE_KEY_PREVIOUS", err)
	}
}

// TestLoadRejectsChallengeKeyPreviousWithoutCurrent proves a retired key can
// never end up issuing fresh challenges: govue's pow signs with its first
// key, so setting only the previous key would hand it the role it was
// rotated out of.
func TestLoadRejectsChallengeKeyPreviousWithoutCurrent(t *testing.T) {
	_, err := Load(env(map[string]string{"CHALLENGE_KEY_PREVIOUS": validHexKey}))
	if err == nil || !strings.Contains(err.Error(), "CHALLENGE_KEY_PREVIOUS") {
		t.Fatalf("Load() error = %v, want a problem naming CHALLENGE_KEY_PREVIOUS", err)
	}
}

// TestDecodeKeyErrorNeverQuotesTheInvalidInput: hex.DecodeString's own error
// text names the offending byte, which would put one character of a
// misconfigured key on stderr and into the pod log if Load's error wrapped
// it with %w.
func TestDecodeKeyErrorNeverQuotesTheInvalidInput(t *testing.T) {
	const invalid = "not-hex-at-all-XYZ"
	_, err := Load(env(map[string]string{"CHALLENGE_KEY": invalid}))
	if err == nil {
		t.Fatal("Load() = nil error, want one naming CHALLENGE_KEY")
	}
	if strings.Contains(err.Error(), invalid) {
		t.Errorf("Load() error = %q, must not contain the invalid key value %q", err.Error(), invalid)
	}
}

func TestLoadMailRateLimitDefaultsAndOverrides(t *testing.T) {
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.MailRateLimitBurst != DefaultMailRateLimitBurst || cfg.MailRateLimitPer != DefaultMailRateLimitPer {
		t.Errorf("mail rate limit = %d/%v, want %d/%v",
			cfg.MailRateLimitBurst, cfg.MailRateLimitPer, DefaultMailRateLimitBurst, DefaultMailRateLimitPer)
	}

	cfg, err = Load(env(map[string]string{
		"MAIL_RATE_LIMIT_BURST": "10",
		"MAIL_RATE_LIMIT_PER":   "30m",
	}))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.MailRateLimitBurst != 10 || cfg.MailRateLimitPer != 30*time.Minute {
		t.Errorf("mail rate limit = %d/%v, want 10/30m", cfg.MailRateLimitBurst, cfg.MailRateLimitPer)
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
			env:  map[string]string{"TRUSTED_PROXIES": "not-a-cidr"},
			want: "TRUSTED_PROXIES",
		},
		"rate limit burst out of range": {
			env:  map[string]string{"RATE_LIMIT_BURST": "0"},
			want: "RATE_LIMIT_BURST",
		},
		"rate limit per not a duration": {
			env:  map[string]string{"RATE_LIMIT_PER": "soon"},
			want: "RATE_LIMIT_PER",
		},
		"smtp host with a port": {
			env:  map[string]string{"SMTP_HOST": "mail.example.test:587"},
			want: "host is not a host name",
		},
		"smtp port out of range": {
			env:  map[string]string{"SMTP_HOST": "mail.example.test", "SMTP_PORT": "70000"},
			want: "SMTP_PORT",
		},
		"feed refresh interval too short": {
			env:  map[string]string{"FEED_REFRESH_INTERVAL": "10s"},
			want: "FEED_REFRESH_INTERVAL",
		},
		"feed refresh interval not a duration": {
			env:  map[string]string{"FEED_REFRESH_INTERVAL": "often"},
			want: "FEED_REFRESH_INTERVAL",
		},
		"feed service url not a url": {
			env:  map[string]string{"FEED_SERVICE_URL": "not a url"},
			want: "FEED_SERVICE_URL",
		},
		"feed service url with credentials": {
			env:  map[string]string{"FEED_SERVICE_URL": "http://a:b@feed.internal"},
			want: "FEED_SERVICE_URL",
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

// validUmamiWebsiteID is the lowercase UUID shape Umami names a website
// by, the only shape Load accepts.
const validUmamiWebsiteID = "0f1e2d3c-4b5a-4968-8776-655443322110"

func TestLoadFeedAndUmamiOverrides(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"FEED_SERVICE_URL":      "http://feed.ostendorf-tech.svc.cluster.local/",
		"FEED_REFRESH_INTERVAL": "30m",
		"UMAMI_URL":             "http://umami.umami.svc.cluster.local/",
		"UMAMI_WEBSITE_ID":      validUmamiWebsiteID,
	}))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	// The trailing slash is trimmed, so joining /api/v1/posts onto it
	// cannot produce a double slash.
	if cfg.FeedServiceURL != "http://feed.ostendorf-tech.svc.cluster.local" {
		t.Errorf("FeedServiceURL = %q, want the trailing slash trimmed", cfg.FeedServiceURL)
	}
	if cfg.FeedRefreshInterval != 30*time.Minute {
		t.Errorf("FeedRefreshInterval = %v, want 30m", cfg.FeedRefreshInterval)
	}
	if cfg.Umami.URL != "http://umami.umami.svc.cluster.local" {
		t.Errorf("Umami.URL = %q, want the trailing slash trimmed", cfg.Umami.URL)
	}
	if cfg.Umami.WebsiteID != validUmamiWebsiteID {
		t.Errorf("Umami.WebsiteID = %q, want %q", cfg.Umami.WebsiteID, validUmamiWebsiteID)
	}
	if !cfg.Umami.Enabled() {
		t.Error("Umami.Enabled() = false, want true with both values set")
	}
}

func TestLoadRejectsInvalidUmamiPairs(t *testing.T) {
	tests := map[string]map[string]string{
		"URL alone":        {"UMAMI_URL": "http://umami"},
		"ID alone":         {"UMAMI_WEBSITE_ID": validUmamiWebsiteID},
		"relative URL":     {"UMAMI_URL": "umami:3000", "UMAMI_WEBSITE_ID": validUmamiWebsiteID},
		"URL with query":   {"UMAMI_URL": "http://umami/?x=1", "UMAMI_WEBSITE_ID": validUmamiWebsiteID},
		"URL with user":    {"UMAMI_URL": "http://a:b@umami", "UMAMI_WEBSITE_ID": validUmamiWebsiteID},
		"ftp URL":          {"UMAMI_URL": "ftp://umami", "UMAMI_WEBSITE_ID": validUmamiWebsiteID},
		"ID that is no ID": {"UMAMI_URL": "http://umami", "UMAMI_WEBSITE_ID": "ostendorf.tech"},
		"plain http host":  {"UMAMI_URL": "http://umami.example.test", "UMAMI_WEBSITE_ID": validUmamiWebsiteID},
	}

	for name, values := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Load(env(values))
			if err == nil {
				t.Fatal("Load() = nil error, want one naming UMAMI_URL or UMAMI_WEBSITE_ID")
			}
			if !strings.Contains(err.Error(), "UMAMI_") {
				t.Errorf("Load() error = %q, want it to mention the UMAMI_ variable at fault", err.Error())
			}
		})
	}

	// govue accepts an upper case id and lowers it: a UUID is case
	// insensitive. The site's own check used to refuse it.
	cfg, err := Load(env(map[string]string{
		"UMAMI_URL": "http://umami", "UMAMI_WEBSITE_ID": strings.ToUpper(validUmamiWebsiteID),
	}))
	if err != nil {
		t.Fatalf("Load() with an upper case id = %v, want nil", err)
	}
	if cfg.Umami.WebsiteID != validUmamiWebsiteID {
		t.Errorf("Umami.WebsiteID = %q, want it lowered to %q", cfg.Umami.WebsiteID, validUmamiWebsiteID)
	}

	// Both unset is the feature switched off, not a problem.
	if _, err := Load(env(nil)); err != nil {
		t.Errorf("Load() with no Umami configuration = %v, want nil", err)
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
