package feedconfig

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/secrettest"
)

// env builds a getenv function from a map, so a test can set exactly the
// variables it cares about and leave everything else unset.
func env(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

// requiredValues carries the three required variables Load cannot start
// without, so a test about something else does not want that failure
// mixed in.
func requiredValues(values map[string]string) map[string]string {
	merged := map[string]string{
		"OPENBAO_ADDR":     "https://openbao.example.test",
		"OPENBAO_ROLE":     "feed",
		"OPENBAO_JWT_FILE": "/var/run/secrets/kubernetes.io/serviceaccount/token",
	}
	for k, v := range values {
		merged[k] = v
	}
	return merged
}

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(requiredValues(nil)))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.ListenAddr != DefaultListenAddr {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, DefaultListenAddr)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.OpenBaoAddr == nil || cfg.OpenBaoAddr.String() != "https://openbao.example.test" {
		t.Errorf("OpenBaoAddr = %v, want https://openbao.example.test", cfg.OpenBaoAddr)
	}
	if cfg.OpenBaoAuthPath != DefaultOpenBaoAuthPath {
		t.Errorf("OpenBaoAuthPath = %q, want %q", cfg.OpenBaoAuthPath, DefaultOpenBaoAuthPath)
	}
	if cfg.OpenBaoRole != "feed" {
		t.Errorf("OpenBaoRole = %q, want feed", cfg.OpenBaoRole)
	}
	if cfg.OpenBaoJWTFile != "/var/run/secrets/kubernetes.io/serviceaccount/token" {
		t.Errorf("OpenBaoJWTFile = %q, want the required value", cfg.OpenBaoJWTFile)
	}
	if cfg.OpenBaoAuthMethod != AuthMethodKubernetes {
		t.Errorf("OpenBaoAuthMethod = %q, want %q", cfg.OpenBaoAuthMethod, AuthMethodKubernetes)
	}
	if cfg.OpenBaoToken != "" {
		t.Error("OpenBaoToken is set with the Kubernetes method")
	}
	// The same path old_simon-ostendorf.de used, spelled out rather than
	// read back from the constant, so a changed default fails here.
	if cfg.OpenBaoSecretPath != "instagram.com/api-tokens/data/simonostendorf.vtech" {
		t.Errorf("OpenBaoSecretPath = %q, want the old default", cfg.OpenBaoSecretPath)
	}
	if cfg.SyncInterval != DefaultSyncInterval {
		t.Errorf("SyncInterval = %v, want %v", cfg.SyncInterval, DefaultSyncInterval)
	}
	if cfg.TokenRefreshAge != DefaultTokenRefreshAge {
		t.Errorf("TokenRefreshAge = %v, want %v", cfg.TokenRefreshAge, DefaultTokenRefreshAge)
	}
	if cfg.APIVersion != DefaultAPIVersion {
		t.Errorf("APIVersion = %q, want %q", cfg.APIVersion, DefaultAPIVersion)
	}
	// Spelled out rather than read back from the constants, so a changed
	// default fails here.
	if cfg.PostCount != 200 {
		t.Errorf("PostCount = %d, want 200", cfg.PostCount)
	}
	if cfg.PostsPerRequest != 50 {
		t.Errorf("PostsPerRequest = %d, want 50", cfg.PostsPerRequest)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(requiredValues(map[string]string{
		"LISTEN_ADDR":         ":9091",
		"LOG_LEVEL":           "debug",
		"OPENBAO_AUTH_PATH":   "custom-kubernetes",
		"OPENBAO_SECRET_PATH": "custom/path",
		"SYNC_INTERVAL":       "30m",
		"TOKEN_REFRESH_AGE":   "14d",
		"API_VERSION":         "v24.0",
		"POST_COUNT":          "8",
		"POSTS_PER_REQUEST":   "100",
	})))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.ListenAddr != ":9091" {
		t.Errorf("ListenAddr = %q, want :9091", cfg.ListenAddr)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want debug", cfg.LogLevel)
	}
	if cfg.OpenBaoAuthPath != "custom-kubernetes" {
		t.Errorf("OpenBaoAuthPath = %q, want custom-kubernetes", cfg.OpenBaoAuthPath)
	}
	if cfg.OpenBaoSecretPath != "custom/path" {
		t.Errorf("OpenBaoSecretPath = %q, want custom/path", cfg.OpenBaoSecretPath)
	}
	if cfg.SyncInterval != 30*time.Minute {
		t.Errorf("SyncInterval = %v, want 30m", cfg.SyncInterval)
	}
	if cfg.TokenRefreshAge != 14*24*time.Hour {
		t.Errorf("TokenRefreshAge = %v, want 14d", cfg.TokenRefreshAge)
	}
	if cfg.APIVersion != "v24.0" {
		t.Errorf("APIVersion = %q, want v24.0", cfg.APIVersion)
	}
	if cfg.PostCount != 8 {
		t.Errorf("PostCount = %d, want 8", cfg.PostCount)
	}
	if cfg.PostsPerRequest != 100 {
		t.Errorf("PostsPerRequest = %d, want 100", cfg.PostsPerRequest)
	}
}

// TestLoadPostCountZeroIsAccepted proves POST_COUNT=0, the maximum of
// 10000 posts, is accepted rather than refused, and POSTS_PER_REQUEST=1
// with it: 10000 posts at one per request stay within the client's page
// bound, see internal/instagram's TestMediaZeroMeansTheMaximumAtOnePostPerRequest.
func TestLoadPostCountZeroIsAccepted(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(requiredValues(map[string]string{"POST_COUNT": "0", "POSTS_PER_REQUEST": "1"})))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.PostCount != 0 || cfg.PostsPerRequest != 1 {
		t.Errorf("PostCount, PostsPerRequest = %d, %d; want 0, 1", cfg.PostCount, cfg.PostsPerRequest)
	}
}

// TestLoadRefusesPostsPerRequestOutOfRangeWithoutTheValue proves a value
// outside 1 to 100 is a startup error that names the variable and never
// quotes the value, over the whole unwrap tree.
func TestLoadRefusesPostsPerRequestOutOfRangeWithoutTheValue(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"0", "101", "-7", "987654321", "fifty"} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			_, err := Load(env(requiredValues(map[string]string{"POSTS_PER_REQUEST": raw})))
			if err == nil {
				t.Fatal("Load() = nil error, want one naming POSTS_PER_REQUEST")
			}
			if !strings.Contains(err.Error(), "POSTS_PER_REQUEST") {
				t.Errorf("Load() error does not name POSTS_PER_REQUEST")
			}
			if len(raw) > 2 {
				// One and two digit values are too short to search for
				// without matching the bounds the message names.
				secrettest.AssertHidden(t, err, raw)
			}
		})
	}
}

func TestLoadRequiresEachRequiredField(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"OPENBAO_ADDR":     "OPENBAO_ADDR",
		"OPENBAO_ROLE":     "OPENBAO_ROLE",
		"OPENBAO_JWT_FILE": "OPENBAO_JWT_FILE",
	}
	for missing, want := range tests {
		t.Run(missing, func(t *testing.T) {
			t.Parallel()

			values := requiredValues(nil)
			delete(values, missing)
			_, err := Load(env(values))
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("Load() error = %v, want a problem naming %q", err, want)
			}
		})
	}
}

// TestLoadAggregatesEveryMissingRequiredField proves every missing required
// field is named in the one aggregated error, not just the first one Load
// happens to read.
func TestLoadAggregatesEveryMissingRequiredField(t *testing.T) {
	t.Parallel()

	_, err := Load(env(nil))
	if err == nil {
		t.Fatal("Load() = nil error, want one aggregating every missing field")
	}
	for _, want := range []string{"OPENBAO_ADDR", "OPENBAO_ROLE", "OPENBAO_JWT_FILE"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Load() error = %q, want it to also mention %q", err.Error(), want)
		}
	}
}

func TestLoadRejectsEachInvalidField(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		env  map[string]string
		want string
	}{
		"invalid log level": {
			env:  map[string]string{"LOG_LEVEL": "verbose"},
			want: "LOG_LEVEL",
		},
		"openbao addr without a scheme": {
			env:  map[string]string{"OPENBAO_ADDR": "openbao.example.test"},
			want: "OPENBAO_ADDR",
		},
		"openbao addr without a host": {
			env:  map[string]string{"OPENBAO_ADDR": "https:///path"},
			want: "OPENBAO_ADDR",
		},
		"openbao addr with credentials": {
			env:  map[string]string{"OPENBAO_ADDR": "https://user:pass@openbao.example.test"},
			want: "OPENBAO_ADDR",
		},
		"openbao addr with a query": {
			env:  map[string]string{"OPENBAO_ADDR": "https://openbao.example.test?x=1"},
			want: "OPENBAO_ADDR",
		},
		"sync interval too short": {
			env:  map[string]string{"SYNC_INTERVAL": "10s"},
			want: "SYNC_INTERVAL",
		},
		"sync interval too long": {
			env:  map[string]string{"SYNC_INTERVAL": "48h"},
			want: "SYNC_INTERVAL",
		},
		"sync interval not a duration": {
			env:  map[string]string{"SYNC_INTERVAL": "often"},
			want: "SYNC_INTERVAL",
		},
		// Meta refuses to refresh a token younger than 24 hours.
		"token refresh age below one day": {
			env:  map[string]string{"TOKEN_REFRESH_AGE": "12h"},
			want: "TOKEN_REFRESH_AGE",
		},
		// A long lived token expires after 60 days.
		"token refresh age past expiry": {
			env:  map[string]string{"TOKEN_REFRESH_AGE": "60d"},
			want: "TOKEN_REFRESH_AGE",
		},
		"negative post count": {
			env:  map[string]string{"POST_COUNT": "-1"},
			want: "POST_COUNT",
		},
		"post count above the ceiling": {
			env:  map[string]string{"POST_COUNT": "10001"},
			want: "POST_COUNT",
		},
		"posts per request of zero": {
			env:  map[string]string{"POSTS_PER_REQUEST": "0"},
			want: "POSTS_PER_REQUEST",
		},
		"posts per request above the API maximum": {
			env:  map[string]string{"POSTS_PER_REQUEST": "101"},
			want: "POSTS_PER_REQUEST",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := Load(env(requiredValues(tc.env)))
			if err == nil {
				t.Fatalf("Load() = nil error, want one mentioning %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Load() error = %q, want it to mention %q", err.Error(), tc.want)
			}
		})
	}
}

func TestLoadAggregatesInvalidFieldsTogetherWithMissingOnes(t *testing.T) {
	t.Parallel()

	values := requiredValues(nil)
	delete(values, "OPENBAO_ROLE")
	values["LOG_LEVEL"] = "verbose"

	_, err := Load(env(values))
	if err == nil {
		t.Fatal("Load() = nil error, want one aggregating both problems")
	}
	for _, want := range []string{"OPENBAO_ROLE", "LOG_LEVEL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Load() error = %q, want it to also mention %q", err.Error(), want)
		}
	}
}
