// Package feedconfig loads and validates the feed service's configuration
// from the environment, once, in main. Every problem is collected and
// returned together, so a misconfigured deployment reports all of its
// mistakes on the first start instead of one per restart.
//
// One field is a secret: OPENBAO_TOKEN, the OpenBao token of the token auth
// method, which also accepts an OPENBAO_TOKEN_FILE twin the same way
// internal/config's secrets do. It is held as an openbao.Token, which never
// prints its value. The Instagram access token itself lives in OpenBao,
// read and written through internal/openbao and internal/instagram, never
// here. No error this package returns quotes a refused value.
package feedconfig

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/ostendorf-cloud/govue/envconf"
	"github.com/ostendorf-cloud/govue/logging"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/instagram"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/openbao"
)

// The values OPENBAO_AUTH_METHOD accepts.
const (
	// AuthMethodKubernetes logs in with the pod's service account token. It
	// is the default and the method for a cluster.
	AuthMethodKubernetes = "kubernetes"
	// AuthMethodToken uses a token handed to the process, for a local run.
	// The token is never renewed.
	AuthMethodToken = "token"
)

// Defaults applied to a field left unset in the environment.
const (
	// DefaultListenAddr is distinct from the website's ":8080" public
	// default. It is the same port as the website's default ADMIN_ADDR
	// (":8081"), so both on one machine need one of the two changed.
	DefaultListenAddr        = ":8081"
	DefaultLogLevel          = "info"
	DefaultOpenBaoAuthMethod = AuthMethodKubernetes
	DefaultOpenBaoAuthPath   = "kubernetes"
	// DefaultSecretPath is the same OpenBao path old_simon-ostendorf.de used.
	DefaultSecretPath = "instagram.com/api-tokens/data/simonostendorf.vtech"
	// DefaultSecretKeyAccessToken and DefaultSecretKeyRefreshedAt name the
	// two keys inside that secret: the Instagram access token and the RFC
	// 3339 time it was last refreshed.
	DefaultSecretKeyAccessToken = "access_token"
	DefaultSecretKeyRefreshedAt = "refreshed_at"
	DefaultSyncInterval         = time.Hour
	DefaultTokenRefreshAge      = 7 * 24 * time.Hour
	// DefaultAPIVersion reads internal/instagram's own pinned version
	// rather than repeating it, so the two can never drift.
	DefaultAPIVersion = instagram.DefaultVersion
	// DefaultPostCount caps how many posts one sync collects across all
	// pages; 0 means the maximum of 10000 (instagram.MaxPosts). Every post
	// held costs the bytes of its picture,
	// see the chart's README for the memory this takes.
	DefaultPostCount = 200
	// DefaultPostsPerRequest is how many posts one call to Meta asks for.
	DefaultPostsPerRequest = instagram.DefaultPostsPerRequest
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
	// POST_COUNT is 0 (the maximum) or at most maxPostCount, the same bound
	// internal/feedclient applies to the list the website reads.
	minPostCount       = 0
	maxPostCount       = instagram.MaxPosts
	minPostsPerRequest = 1
	maxPostsPerRequest = instagram.MaxPostsPerRequest
)

// Config is the feed service's whole configuration, read once in main.
type Config struct {
	ListenAddr string
	LogLevel   slog.Level

	// OpenBaoAddr is the OpenBao address, for every auth method.
	OpenBaoAddr *url.URL
	// OpenBaoAuthMethod is AuthMethodKubernetes or AuthMethodToken.
	OpenBaoAuthMethod string

	// OpenBaoAuthPath, OpenBaoRole and OpenBaoJWTFile are set with
	// AuthMethodKubernetes only: everything openbao.Kubernetes needs.
	OpenBaoAuthPath string
	OpenBaoRole     string
	// OpenBaoJWTFile is the path to the Kubernetes service account token,
	// not a secret value itself: the file it points at is read later by
	// internal/openbao, not here.
	OpenBaoJWTFile string

	// OpenBaoToken is set with AuthMethodToken only. It is a secret and
	// prints as "[redacted]".
	OpenBaoToken openbao.Token

	// OpenBaoSecretPath is the OpenBao KV v2 data path of the secret that
	// holds the Instagram access token; see internal/instagram.TokenStore.
	OpenBaoSecretPath string
	// SecretKeyAccessToken and SecretKeyRefreshedAt name the two keys
	// inside that secret; see instagram.TokenStore.
	SecretKeyAccessToken string
	SecretKeyRefreshedAt string
	// SyncInterval is how often the feed is reloaded from Meta.
	SyncInterval time.Duration
	// TokenRefreshAge is how old the token may get before TokenStore tries
	// to refresh it.
	TokenRefreshAge time.Duration

	// APIVersion pins the Graph API version calls are made against; see
	// internal/instagram.DefaultVersion.
	APIVersion string
	// PostCount caps the posts one sync collects across all pages; 0 means
	// the maximum of 10000.
	PostCount int
	// PostsPerRequest is how many posts one call to Meta asks for, 1 to
	// instagram.MaxPostsPerRequest.
	PostsPerRequest int
}

// Load reads every setting from the environment through getenv and
// validates the result. A nil getenv reads the real process environment.
// Every problem is reported together through the returned error, never one
// at a time.
func Load(getenv func(string) string) (Config, error) {
	loader := envconf.New(getenv)

	cfg := Config{
		ListenAddr:        loader.String("LISTEN_ADDR", DefaultListenAddr),
		OpenBaoAddr:       loader.URL("OPENBAO_ADDR", ""),
		OpenBaoSecretPath: loader.String("OPENBAO_SECRET_PATH", DefaultSecretPath),
		SyncInterval:      loader.Duration("SYNC_INTERVAL", DefaultSyncInterval, minSyncInterval, maxSyncInterval),
		TokenRefreshAge:   loader.Duration("TOKEN_REFRESH_AGE", DefaultTokenRefreshAge, minTokenRefreshAge, maxTokenRefreshAge),
		APIVersion:        loader.String("API_VERSION", DefaultAPIVersion),
		PostCount:         loader.Int("POST_COUNT", DefaultPostCount, minPostCount, maxPostCount),
		PostsPerRequest:   loader.Int("POSTS_PER_REQUEST", DefaultPostsPerRequest, minPostsPerRequest, maxPostsPerRequest),
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

	resolved := resolvedGetenv(getenv)
	fail(loadAuth(&cfg, loader, resolved))

	cfg.SecretKeyAccessToken = loader.String("OPENBAO_SECRET_KEY_ACCESS_TOKEN", DefaultSecretKeyAccessToken)
	fail(validateKey("OPENBAO_SECRET_KEY_ACCESS_TOKEN", resolved, cfg.SecretKeyAccessToken))
	cfg.SecretKeyRefreshedAt = loader.String("OPENBAO_SECRET_KEY_REFRESHED_AT", DefaultSecretKeyRefreshedAt)
	fail(validateKey("OPENBAO_SECRET_KEY_REFRESHED_AT", resolved, cfg.SecretKeyRefreshedAt))
	if cfg.SecretKeyAccessToken == cfg.SecretKeyRefreshedAt {
		fail(wrap("OPENBAO_SECRET_KEY_REFRESHED_AT", errors.New("must differ from OPENBAO_SECRET_KEY_ACCESS_TOKEN")))
	}

	if err := errors.Join(append(problems, loader.Err())...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// kubernetesOnly are the variables only the Kubernetes auth method reads;
// tokenOnly the ones only the token method reads. Each set is refused next
// to the other method, so a mixed configuration fails loudly instead of
// silently ignoring half of itself.
var (
	kubernetesOnly = []string{"OPENBAO_AUTH_PATH", "OPENBAO_ROLE", "OPENBAO_JWT_FILE"}
	tokenOnly      = []string{"OPENBAO_TOKEN", "OPENBAO_TOKEN_FILE"}
)

// loadAuth reads OPENBAO_AUTH_METHOD and the variables of the method it
// names into cfg. Problems of the loader land in loader.Err; the method's
// own refusals are returned joined. No message carries a given value: an
// unknown method names the variable and the allowed values only, and a
// refused variable is named, never quoted.
func loadAuth(cfg *Config, loader *envconf.Loader, getenv func(string) string) error {
	set := func(name string) bool { return strings.TrimSpace(getenv(name)) != "" }
	refuse := func(names []string, method string) error {
		var errs []error
		for _, name := range names {
			if set(name) {
				errs = append(errs, wrap(name, fmt.Errorf("must not be set with OPENBAO_AUTH_METHOD=%s", method)))
			}
		}
		return errors.Join(errs...)
	}

	cfg.OpenBaoAuthMethod = loader.String("OPENBAO_AUTH_METHOD", DefaultOpenBaoAuthMethod)
	switch cfg.OpenBaoAuthMethod {
	case AuthMethodKubernetes:
		cfg.OpenBaoAuthPath = loader.String("OPENBAO_AUTH_PATH", DefaultOpenBaoAuthPath)
		cfg.OpenBaoRole = loader.Required("OPENBAO_ROLE")
		cfg.OpenBaoJWTFile = loader.Required("OPENBAO_JWT_FILE")
		return refuse(tokenOnly, AuthMethodKubernetes)
	case AuthMethodToken:
		cfg.OpenBaoToken = openbao.Token(loader.RequiredSecretOrFile("OPENBAO_TOKEN"))
		return refuse(kubernetesOnly, AuthMethodToken)
	default:
		cfg.OpenBaoAuthMethod = ""
		return wrap("OPENBAO_AUTH_METHOD", fmt.Errorf("must be %s or %s (the value is not shown)",
			AuthMethodKubernetes, AuthMethodToken))
	}
}

// validateKey checks a key name inside the secret: not blank, no "/" and no
// control character. Set to nothing but whitespace counts as empty, not as
// unset. The control character check reads the raw value, so a newline the
// loader would trim is refused too. The message names the variable only,
// never the refused value.
func validateKey(name string, getenv func(string) string, value string) error {
	raw := getenv(name)
	switch {
	case raw != "" && strings.TrimSpace(raw) == "", value == "":
		return wrap(name, errors.New("must not be empty"))
	case strings.Contains(value, "/"):
		return wrap(name, errors.New(`must not contain "/" (the value is not shown)`))
	case strings.ContainsFunc(raw, unicode.IsControl), strings.ContainsFunc(value, unicode.IsControl):
		return wrap(name, errors.New("must not contain a control character (the value is not shown)"))
	}
	return nil
}

// resolvedGetenv defaults a nil lookup function to the real process
// environment, the same default envconf.New applies.
func resolvedGetenv(getenv func(string) string) func(string) string {
	if getenv == nil {
		return os.Getenv
	}
	return getenv
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
