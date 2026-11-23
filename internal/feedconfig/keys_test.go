package feedconfig

import (
	"strings"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/secrettest"
)

func TestLoadSecretKeyDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(requiredValues(nil)))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// Spelled out rather than read back from the constants, so a changed
	// default fails here: these are the keys the secret holds.
	if cfg.SecretKeyAccessToken != "access_token" {
		t.Errorf("SecretKeyAccessToken = %q, want access_token", cfg.SecretKeyAccessToken)
	}
	if cfg.SecretKeyRefreshedAt != "refreshed_at" {
		t.Errorf("SecretKeyRefreshedAt = %q, want refreshed_at", cfg.SecretKeyRefreshedAt)
	}
}

func TestLoadSecretKeyOverrides(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(requiredValues(map[string]string{
		"OPENBAO_SECRET_KEY_ACCESS_TOKEN": "ig.token",
		"OPENBAO_SECRET_KEY_REFRESHED_AT": "ig-refreshed",
	})))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.SecretKeyAccessToken != "ig.token" || cfg.SecretKeyRefreshedAt != "ig-refreshed" {
		t.Errorf("keys = %q, %q, want ig.token, ig-refreshed", cfg.SecretKeyAccessToken, cfg.SecretKeyRefreshedAt)
	}
}

func TestLoadRefusesBadSecretKeys(t *testing.T) {
	t.Parallel()

	const marker = "keymarker"
	tests := map[string]string{
		"blank":                     "   ",
		"slash":                     marker + "/x",
		"leading slash":             "/" + marker,
		"newline inside":            marker + "\nx",
		"trailing newline":          marker + "\n",
		"tab inside":                marker + "\tx",
		"escape character":          marker + "\x1b[31m",
		"delete character":          marker + "\x7f",
		"unicode control character": marker + "\u0085",
	}
	for _, name := range []string{"OPENBAO_SECRET_KEY_ACCESS_TOKEN", "OPENBAO_SECRET_KEY_REFRESHED_AT"} {
		for label, value := range tests {
			t.Run(name+" "+label, func(t *testing.T) {
				t.Parallel()

				_, err := Load(env(requiredValues(map[string]string{name: value})))
				if err == nil {
					t.Fatal("Load() = nil error, want a refusal")
				}
				if !strings.Contains(err.Error(), name) {
					t.Errorf("Load() error does not name %s", name)
				}
				secrettest.AssertHidden(t, err, marker)
			})
		}
	}
}

func TestLoadRefusesTheSameKeyTwice(t *testing.T) {
	t.Parallel()

	_, err := Load(env(requiredValues(map[string]string{"OPENBAO_SECRET_KEY_REFRESHED_AT": "access_token"})))
	if err == nil || !strings.Contains(err.Error(), "OPENBAO_SECRET_KEY_REFRESHED_AT") {
		t.Fatalf("Load() error = %v, want a refusal naming OPENBAO_SECRET_KEY_REFRESHED_AT", err)
	}
}
