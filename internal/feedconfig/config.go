// Package feedconfig loads and validates the feed service's configuration
// from the environment, once, in main. Every problem is collected and
// returned together, so a misconfigured deployment reports all of its
// mistakes on the first start instead of one per restart.
//
// None of the ten fields below are secret-shaped: the one actual secret,
// the Instagram access token, lives in OpenBao, read and written through
// internal/openbao and internal/instagram, never here. That is deliberately
// why there is no "_FILE" twin for any of these, unlike internal/config's
// SMTP credentials or signing keys.
package feedconfig

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/ostendorf-cloud/govue/envconf"
	"github.com/ostendorf-cloud/govue/logging"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/instagram"
)

// Defaults applied to a field left unset in the environment.
const (
	// DefaultListenAddr is distinct from the website's ":8080" public
	// default. It is the same port as the website's default ADMIN_ADDR
	// (":8081"), so both on one machine need one of the two changed.
	DefaultListenAddr      = ":8081"
	DefaultLogLevel        = "info"
	DefaultOpenBaoAuthPath = "kubernetes"
	// DefaultTokenPath is the same OpenBao path old_simon-ostendorf.de used.
	DefaultTokenPath       = "instagram.com/api-tokens/data/simonostendorf.vtech"
	DefaultSyncInterval    = time.Hour
	DefaultTokenRefreshAge = 7 * 24 * time.Hour
	// DefaultAPIVersion reads internal/instagram's own pinned version
	// rather than repeating it, so the two can never drift.
	DefaultAPIVersion = instagram.DefaultVersion
	// DefaultPostCount is how many posts the feed asks Meta for; the old
	// site asked for the same 20.
	DefaultPostCount = 20
)

// Sync interval and token refresh age bounds, the range SYNC_INTERVAL and
// TOKEN_REFRESH_AGE are checked against; see the defaults above. They
// mirror internal/config's own Instagram bounds for the same reasons: Meta
// refreshes only tokens older than 24 hours, and a long lived token expires
// after 60 days.
const (
	minSyncInterval    = time.Minute
	maxSyncInterval    = 24 * time.Hour
	minTokenRefreshAge = 24 * time.Hour
	maxTokenRefreshAge = 50 * 24 * time.Hour
	minPostCount       = 1
	maxPostCount       = 100
)

// Config is the feed service's whole configuration, read once in main.
type Config struct {
	ListenAddr string
	LogLevel   slog.Level

	// OpenBaoAddr, OpenBaoAuthPath, OpenBaoRole and OpenBaoJWTFile are
	// everything internal/openbao.New needs to log in with the Kubernetes
	// auth method; see that package for what each one means.
	OpenBaoAddr     *url.URL
	OpenBaoAuthPath string
	OpenBaoRole     string
	// OpenBaoJWTFile is the path to the Kubernetes service account token,
	// not a secret value itself: the file it points at is read later by
	// internal/openbao, not here.
	OpenBaoJWTFile string

	// TokenPath is the OpenBao KV v2 data path the Instagram access token
	// is stored at; see internal/instagram.TokenStore.
	TokenPath string
	// SyncInterval is how often the feed is reloaded from Meta.
	SyncInterval time.Duration
	// TokenRefreshAge is how old the token may get before TokenStore tries
	// to refresh it.
	TokenRefreshAge time.Duration

	// APIVersion pins the Graph API version calls are made against; see
	// internal/instagram.DefaultVersion.
	APIVersion string
	// PostCount is how many posts are asked for per sync.
	PostCount int
}

// Load reads every setting from the environment through getenv and
// validates the result. A nil getenv reads the real process environment.
// Every problem is reported together through the returned error, never one
// at a time.
func Load(getenv func(string) string) (Config, error) {
	loader := envconf.New(getenv)

	cfg := Config{
		ListenAddr:      loader.String("LISTEN_ADDR", DefaultListenAddr),
		OpenBaoAddr:     loader.URL("OPENBAO_ADDR", ""),
		OpenBaoAuthPath: loader.String("OPENBAO_AUTH_PATH", DefaultOpenBaoAuthPath),
		OpenBaoRole:     loader.Required("OPENBAO_ROLE"),
		OpenBaoJWTFile:  loader.Required("OPENBAO_JWT_FILE"),
		TokenPath:       loader.String("TOKEN_PATH", DefaultTokenPath),
		SyncInterval:    loader.Duration("SYNC_INTERVAL", DefaultSyncInterval, minSyncInterval, maxSyncInterval),
		TokenRefreshAge: loader.Duration("TOKEN_REFRESH_AGE", DefaultTokenRefreshAge, minTokenRefreshAge, maxTokenRefreshAge),
		APIVersion:      loader.String("API_VERSION", DefaultAPIVersion),
		PostCount:       loader.Int("POST_COUNT", DefaultPostCount, minPostCount, maxPostCount),
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

	if err := errors.Join(append(problems, loader.Err())...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// wrap names the variable a problem came from, or returns nil for no
// problem at all. Every feedconfig.Load failure that is not already
// reported by envconf's own loader methods goes through it, so every error
// in the aggregate names the variable to fix.
func wrap(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("feedconfig: %s: %w", name, err)
}
