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

	"github.com/ostendorf-cloud/govue/analytics"
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
	DefaultListenAddr = ":8080"
	// DefaultAdminListenAddr is govue's own admin listener default, named
	// ADMIN_ADDR to match its example: health, readiness and metrics serve
	// from here instead of the public listener; see internal/webapp.
	DefaultAdminListenAddr = ":8081"
	DefaultSiteURL         = "https://ostendorf.tech"
	DefaultLogLevel        = "info"
	DefaultRateLimitBurst  = 120
	DefaultRateLimitPer    = time.Minute

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

	// DefaultFeedRefreshInterval is how often the website asks the feed
	// service for the Impressionen section's posts. The feed service
	// itself only reloads from Meta on its own, much longer interval
	// (internal/feedconfig.DefaultSyncInterval, an hour), so polling it
	// more often than that cannot show a visitor anything fresher; this
	// is shorter anyway because asking is now a cheap in-cluster call
	// rather than a request against Meta's own rate limits, and a shorter
	// interval bounds how long a freshly published post takes to reach
	// the website after the feed service has it.
	DefaultFeedRefreshInterval = 5 * time.Minute

	// DefaultMailRateLimitBurst and DefaultMailRateLimitPer bound every
	// contact mail this service sends in total, independent of who is
	// asking; see internal/webapp for why the per-client limit above
	// cannot stand in for this one. 50 an hour is well above anything a
	// single event technology sole trader's contact form is ever expected
	// to receive legitimately (a handful a day, generously multiplied for
	// a busy week or a marketing push), while still being orders of
	// magnitude below what an attacker spreading requests across many
	// client addresses could otherwise drive the mail account to.
	DefaultMailRateLimitBurst = 50
	DefaultMailRateLimitPer   = time.Hour
)

// Feed refresh bounds, the range FEED_REFRESH_INTERVAL is checked
// against; see DefaultFeedRefreshInterval above.
const (
	minFeedRefreshInterval = time.Minute
	maxFeedRefreshInterval = 24 * time.Hour
)

// Config is the website's whole configuration, read once in main.
type Config struct {
	ListenAddr string
	// AdminListenAddr is govue's admin listener: health, readiness and
	// metrics serve from here, never from ListenAddr, so a kubelet and a
	// scraper reach the pod directly without a public route; see
	// internal/webapp.
	AdminListenAddr string
	SiteURL         *url.URL
	LogLevel        slog.Level

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

	// ChallengeKeyPrevious verifies a challenge issued under a key that
	// was just rotated out, while ChallengeKey carries the new one. Every
	// challenge still has to be issued under ChallengeKey; this only
	// widens what Verify accepts, for the challengeTTL window where a
	// challenge handed out just before the rotation is still in flight.
	// Left unset, rotating CHALLENGE_KEY invalidates every challenge
	// issued under the old key the moment the new one takes effect.
	ChallengeKeyPrevious []byte

	// ChallengeRateLimitBurst/Per bound GET /api/v1/challenge; see
	// DefaultChallengeRateLimitBurst.
	ChallengeRateLimitBurst int
	ChallengeRateLimitPer   time.Duration
	// ContactRateLimitBurst/Per bound POST /api/v1/contact per client
	// address; see DefaultContactRateLimitBurst.
	ContactRateLimitBurst int
	ContactRateLimitPer   time.Duration

	// FeedServiceURL is the origin of the feed service that loads the home
	// page's Impressionen section: internal/feedclient asks it for the
	// post list and the picture behind each one, and this is the only
	// network dependency that section has. Empty switches the section off
	// entirely, the same way an absent Instagram token did before this
	// service existed: no syncer runs, the cache stays empty, and the
	// frontend renders no section at all. This website has no Instagram
	// or Meta credential and no OpenBao client of its own; both live in
	// the feed service (cmd/feed), reached over this URL alone.
	FeedServiceURL string
	// FeedRefreshInterval is how often the website asks the feed service
	// for the current post list; see DefaultFeedRefreshInterval.
	FeedRefreshInterval time.Duration

	// Umami is the connection to the self-hosted Umami every page view is
	// reported to, server to server, read by govue's analytics.LoadConfig
	// (UMAMI_URL, UMAMI_WEBSITE_ID, UMAMI_TIMEOUT, UMAMI_QUEUE_SIZE). The
	// zero value counts nothing, and setting exactly one of the two required
	// values is a configuration mistake rather than a half-enabled feature.
	// No script, cookie or beacon ever reaches the visitor's browser either
	// way.
	Umami analytics.Config

	// MailRateLimitBurst/Per bound every contact mail this service sends
	// in total, across every client address; see
	// DefaultMailRateLimitBurst.
	MailRateLimitBurst int
	MailRateLimitPer   time.Duration
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
		AdminListenAddr:         loader.String("ADMIN_ADDR", DefaultAdminListenAddr),
		SiteURL:                 loader.URL("SITE_URL", DefaultSiteURL),
		RateLimitBurst:          loader.Int("RATE_LIMIT_BURST", DefaultRateLimitBurst, 1, 100_000),
		RateLimitPer:            loader.Duration("RATE_LIMIT_PER", DefaultRateLimitPer, time.Second, time.Hour),
		AllowPlainHTTP:          loader.Bool("ALLOW_PLAIN_HTTP", false),
		SMTPAddr:                loader.String("SMTP_ADDR", DefaultSMTPAddr),
		ChallengeRateLimitBurst: loader.Int("CHALLENGE_RATE_LIMIT_BURST", DefaultChallengeRateLimitBurst, 1, 100_000),
		ChallengeRateLimitPer:   loader.Duration("CHALLENGE_RATE_LIMIT_PER", DefaultChallengeRateLimitPer, time.Second, time.Hour),
		ContactRateLimitBurst:   loader.Int("CONTACT_RATE_LIMIT_BURST", DefaultContactRateLimitBurst, 1, 100_000),
		ContactRateLimitPer:     loader.Duration("CONTACT_RATE_LIMIT_PER", DefaultContactRateLimitPer, time.Second, 24*time.Hour),
		FeedRefreshInterval: loader.Duration("FEED_REFRESH_INTERVAL", DefaultFeedRefreshInterval,
			minFeedRefreshInterval, maxFeedRefreshInterval),
		MailRateLimitBurst: loader.Int("MAIL_RATE_LIMIT_BURST", DefaultMailRateLimitBurst, 1, 100_000),
		MailRateLimitPer:   loader.Duration("MAIL_RATE_LIMIT_PER", DefaultMailRateLimitPer, time.Second, 24*time.Hour),
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

	cfg.SMTPUsername = loader.SecretOrFile("SMTP_USERNAME")
	cfg.SMTPPassword = loader.SecretOrFile("SMTP_PASSWORD")

	challengeKey, err := decodeKey(loader.SecretOrFile("CHALLENGE_KEY"))
	fail(wrap("CHALLENGE_KEY", err))
	cfg.ChallengeKey = challengeKey

	challengeKeyPrevious, err := decodeKey(loader.SecretOrFile("CHALLENGE_KEY_PREVIOUS"))
	fail(wrap("CHALLENGE_KEY_PREVIOUS", err))
	cfg.ChallengeKeyPrevious = challengeKeyPrevious
	if len(cfg.ChallengeKeyPrevious) > 0 && len(cfg.ChallengeKey) == 0 {
		// handleChallenge always issues a fresh challenge under
		// challengeKeys[0], the current key; webapp.go's New puts
		// ChallengeKey there when it is set, and falls back to
		// ChallengeKeyPrevious otherwise. A deployment that set only the
		// previous key would then have a retired key issuing challenges
		// again, exactly the thing rotation is meant to retire.
		fail(wrap("CHALLENGE_KEY_PREVIOUS", errors.New("must not be set without CHALLENGE_KEY, which would then be the one issuing")))
	}

	csrfKey, err := decodeKey(loader.RequiredSecretOrFile("CSRF_KEY"))
	fail(wrap("CSRF_KEY", err))
	cfg.CSRFKey = csrfKey

	// Read raw rather than through loader.URL: that helper treats an empty
	// value as a missing required one, and an empty FEED_SERVICE_URL is
	// this section switched off, not a mistake.
	cfg.FeedServiceURL = strings.TrimRight(loader.String("FEED_SERVICE_URL", ""), "/")
	fail(wrap("FEED_SERVICE_URL", validateOptionalURL(cfg.FeedServiceURL)))

	// govue's own messages name the variable at fault, so no wrap here.
	umami, err := analytics.LoadConfig(getenv)
	fail(err)
	cfg.Umami = umami

	if err := errors.Join(append(problems, loader.Err())...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// validateOptionalURL checks that raw is empty, or an absolute http or
// https URL with no credentials, query or fragment. Shared by every
// optional URL setting this package has (FEED_SERVICE_URL): a feature
// switched off when empty, not a mistake, which is why it is not read
// through loader.URL, whose "" means "required and missing".
func validateOptionalURL(raw string) error {
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	switch {
	case err != nil:
		return fmt.Errorf("must be a URL: %w", err)
	case parsed.Scheme != "http" && parsed.Scheme != "https":
		return errors.New("must start with http:// or https://")
	case parsed.Host == "":
		return errors.New("must contain a host")
	case parsed.User != nil:
		return errors.New("must not contain credentials")
	case parsed.RawQuery != "" || parsed.Fragment != "":
		return errors.New("must not contain a query or a fragment")
	}
	return nil
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
		// hex.DecodeString's own error text names the offending byte, for
		// example "encoding/hex: invalid byte: U+0058 'X'", which would put
		// one character of the key itself on stderr and into the pod log
		// if it were wrapped into this error with %w. The caller never
		// needs that detail to fix a misconfigured key, so it is dropped
		// here.
		return nil, errors.New("must be hex encoded")
	}
	if len(key) < minKeyBytes {
		return nil, fmt.Errorf("decodes to %d bytes, at least %d are required", len(key), minKeyBytes)
	}
	return key, nil
}
