// Package config loads and validates the website's configuration from the
// environment, once, in main. Every problem is collected and returned
// together, so a misconfigured deployment reports all of its mistakes on
// the first start instead of one per restart.
package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ostendorf-cloud/govue/clientip"
	"github.com/ostendorf-cloud/govue/envconf"
	"github.com/ostendorf-cloud/govue/logging"
)

// minKeyBytes is the shortest decoded key this package accepts for a signing
// secret: a key shorter than this is a deployment mistake, not a value worth
// running with, the same floor govue's own example applies to its keys.
const minKeyBytes = 32

// Defaults applied to a field left unset in the environment.
const (
	DefaultListenAddr     = ":8080"
	DefaultSiteURL        = "https://ostendorf.tech"
	DefaultLogLevel       = "info"
	DefaultRateLimitBurst = 120
	DefaultRateLimitPer   = time.Minute

	DefaultSMTPAddr  = "mail.ostendorf.cloud:587"
	DefaultContactTo = "simon@ostendorf.tech"
	// DefaultContactFrom is flagged in the report this milestone ends with:
	// it may belong on ostendorf.tech itself rather than ostendorf.cloud,
	// and that is a decision for Simon, not this change.
	DefaultContactFrom = "ostendorf.tech <no-reply@ostendorf.cloud>"

	// DefaultChallengeRateLimitBurst and DefaultChallengeRateLimitPer bound
	// GET /api/v1/challenge per client address. The browser fetches one
	// automatically whenever a visitor focuses the contact form, so the
	// burst is generous; see old_simon-ostendorf.de's cmd/website/main.go
	// for the values this ports.
	DefaultChallengeRateLimitBurst = 10
	DefaultChallengeRateLimitPer   = 20 * time.Second
	// DefaultContactRateLimitBurst and DefaultContactRateLimitPer bound
	// POST /api/v1/contact per client address, tighter than the challenge
	// limit because every one of these actually tries to send a mail.
	DefaultContactRateLimitBurst = 5
	DefaultContactRateLimitPer   = 10 * time.Minute
)

// Config is the website's whole configuration, read once in main.
type Config struct {
	ListenAddr string
	SiteURL    *url.URL
	LogLevel   slog.Level

	// TrustedProxies are the TCP peers whose X-Forwarded-For entry clientip
	// trusts. Empty means every request is attributed to its TCP peer.
	TrustedProxies []netip.Prefix

	// RateLimitBurst and RateLimitPer bound how many requests one client
	// address may make in a row before it has to wait; see the ratelimit
	// package for the exact token bucket behaviour.
	RateLimitBurst int
	RateLimitPer   time.Duration

	// AllowPlainHTTP drops the session cookie's Secure attribute, for a
	// plain HTTP development run (docker compose, task run). Never set on
	// anything reachable from outside the machine it runs on.
	AllowPlainHTTP bool

	// CSRFKey signs the contact form's per-session CSRF token. Required:
	// the contact form is the only mutating route this service has, and an
	// application that started without a way to protect it would otherwise
	// have to disable that protection outright, which is not a tradeoff
	// this config makes silently.
	CSRFKey []byte

	// SMTPAddr, SMTPUsername and SMTPPassword reach the mail submission
	// server the contact form sends through. Username and password are
	// optional at the config level: a deployment without them yet still
	// serves every other page, and the contact form answers unavailable
	// instead of failing to start; see internal/webapp.
	SMTPAddr     string
	SMTPUsername string
	SMTPPassword string

	// ContactTo and ContactFrom are the fixed addresses of every contact
	// mail: the recipient (always Simon) and the envelope sender.
	ContactTo   *mail.Address
	ContactFrom *mail.Address

	// ChallengeKey signs the proof of work challenge the contact form's
	// Web Worker solves. Optional at the config level, like the SMTP
	// credentials: without it the contact form answers unavailable rather
	// than the service refusing to start.
	ChallengeKey []byte

	// ChallengeRateLimitBurst/Per bound GET /api/v1/challenge; see
	// DefaultChallengeRateLimitBurst.
	ChallengeRateLimitBurst int
	ChallengeRateLimitPer   time.Duration
	// ContactRateLimitBurst/Per bound POST /api/v1/contact; see
	// DefaultContactRateLimitBurst.
	ContactRateLimitBurst int
	ContactRateLimitPer   time.Duration
}

// Load reads every setting from the environment through getenv and
// validates the result. A nil getenv reads the real process environment.
// Every problem is reported together through the returned error, never one
// at a time.
func Load(getenv func(string) string) (Config, error) {
	loader := envconf.New(getenv)
	resolved := resolvedGetenv(getenv)

	cfg := Config{
		ListenAddr:              loader.String("LISTEN_ADDR", DefaultListenAddr),
		SiteURL:                 loader.URL("SITE_URL", DefaultSiteURL),
		RateLimitBurst:          loader.Int("RATE_LIMIT_BURST", DefaultRateLimitBurst, 1, 100_000),
		RateLimitPer:            loader.Duration("RATE_LIMIT_PER", DefaultRateLimitPer, time.Second, time.Hour),
		AllowPlainHTTP:          loader.Bool("ALLOW_PLAIN_HTTP", false),
		SMTPAddr:                loader.String("SMTP_ADDR", DefaultSMTPAddr),
		ChallengeRateLimitBurst: loader.Int("CHALLENGE_RATE_LIMIT_BURST", DefaultChallengeRateLimitBurst, 1, 100_000),
		ChallengeRateLimitPer:   loader.Duration("CHALLENGE_RATE_LIMIT_PER", DefaultChallengeRateLimitPer, time.Second, time.Hour),
		ContactRateLimitBurst:   loader.Int("CONTACT_RATE_LIMIT_BURST", DefaultContactRateLimitBurst, 1, 100_000),
		ContactRateLimitPer:     loader.Duration("CONTACT_RATE_LIMIT_PER", DefaultContactRateLimitPer, time.Second, 24*time.Hour),
	}

	var problems []error
	fail := func(err error) {
		if err != nil {
			problems = append(problems, err)
		}
	}

	level, err := logging.Level(loader.String("LOG_LEVEL", DefaultLogLevel))
	fail(wrap("LOG_LEVEL", err))
	cfg.LogLevel = level

	prefixes, err := clientip.ParseCIDRs(resolved("TRUSTED_PROXY_CIDRS"))
	fail(wrap("TRUSTED_PROXY_CIDRS", err))
	cfg.TrustedProxies = prefixes

	cfg.ContactTo, err = mail.ParseAddress(loader.String("CONTACT_TO", DefaultContactTo))
	fail(wrap("CONTACT_TO", err))

	cfg.ContactFrom, err = mail.ParseAddress(loader.String("CONTACT_FROM", DefaultContactFrom))
	fail(wrap("CONTACT_FROM", err))

	username, err := secretValue(resolved, "SMTP_USERNAME")
	fail(wrap("SMTP_USERNAME", err))
	cfg.SMTPUsername = username

	password, err := secretValue(resolved, "SMTP_PASSWORD")
	fail(wrap("SMTP_PASSWORD", err))
	cfg.SMTPPassword = password

	challengeKeyRaw, err := secretValue(resolved, "CHALLENGE_KEY")
	fail(wrap("CHALLENGE_KEY", err))
	challengeKey, err := decodeKey(challengeKeyRaw)
	fail(wrap("CHALLENGE_KEY", err))
	cfg.ChallengeKey = challengeKey

	csrfKeyRaw, err := secretValue(resolved, "CSRF_KEY")
	fail(wrap("CSRF_KEY", err))
	if err == nil && csrfKeyRaw == "" {
		fail(wrap("CSRF_KEY", errors.New("is required")))
	}
	csrfKey, err := decodeKey(csrfKeyRaw)
	fail(wrap("CSRF_KEY", err))
	cfg.CSRFKey = csrfKey

	if err := errors.Join(append(problems, loader.Err())...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// wrap names the variable a problem came from, or returns nil for no
// problem at all. Every config.Load failure goes through it, so every
// error in the aggregate names the variable to fix.
func wrap(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("config: %s: %w", name, err)
}

// resolvedGetenv defaults a nil lookup function to the real process
// environment, the same default envconf.New applies, so a caller of
// clientip.ParseCIDRs never has to special case a nil getenv.
func resolvedGetenv(getenv func(string) string) func(string) string {
	if getenv == nil {
		return os.Getenv
	}
	return getenv
}

// secretValue resolves one configuration value that may be a secret: either
// name's own value, or the trimmed content of the file name+"_FILE" points
// at. Setting both is an error, so a deployment is never in doubt about
// which one wins. This is what lets OpenBao, External Secrets, a Kubernetes
// secret mount and a plain .env file all supply the same variable without
// this package, or anything above it, knowing which one is in use.
func secretValue(resolved func(string) string, name string) (string, error) {
	direct := strings.TrimSpace(resolved(name))
	filePath := strings.TrimSpace(resolved(name + "_FILE"))
	switch {
	case direct != "" && filePath != "":
		return "", fmt.Errorf("%s and %s_FILE are both set, set only one", name, name)
	case filePath != "":
		raw, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("%s_FILE: read %s: %w", name, filePath, err)
		}
		return strings.TrimRight(string(raw), "\r\n"), nil
	default:
		return direct, nil
	}
}

// decodeKey decodes raw as hex and checks its length. An empty raw means the
// key was simply not configured, not a problem by itself: whether that is
// acceptable is the caller's decision, CSRFKey requiring one and
// ChallengeKey not. The caller names the variable through wrap, so the
// message here never repeats it.
func decodeKey(raw string) ([]byte, error) {
	if raw == "" {
		return nil, nil //nolint:nilnil // empty means "not configured"; the caller decides whether that is a problem
	}
	key, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("must be hex encoded: %w", err)
	}
	if len(key) < minKeyBytes {
		return nil, fmt.Errorf("decodes to %d bytes, at least %d are required", len(key), minKeyBytes)
	}
	return key, nil
}
