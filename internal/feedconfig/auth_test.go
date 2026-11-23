package feedconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/secrettest"
)

// fakeToken is a distinctive stand in for an OpenBao token: every test
// below checks that it never reaches an error, under any verb, at any depth.
const fakeToken = "s.fake-feedconfig-token-must-never-leak-41d2"

// tokenValues is a valid token method configuration plus values.
func tokenValues(values map[string]string) map[string]string {
	merged := map[string]string{
		"OPENBAO_ADDR":        "http://127.0.0.1:8200",
		"OPENBAO_AUTH_METHOD": "token",
		"OPENBAO_TOKEN":       fakeToken,
	}
	for k, v := range values {
		merged[k] = v
	}
	return merged
}

func writeTokenFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadTokenMethod(t *testing.T) {
	t.Parallel()

	tests := map[string]map[string]string{
		"token directly": tokenValues(nil),
		"token from a file with a trailing newline": tokenValues(map[string]string{
			"OPENBAO_TOKEN":      "",
			"OPENBAO_TOKEN_FILE": writeTokenFile(t, fakeToken+"\n"),
		}),
	}
	for name, values := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg, err := Load(env(values))
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.OpenBaoAuthMethod != AuthMethodToken {
				t.Errorf("OpenBaoAuthMethod = %q, want token", cfg.OpenBaoAuthMethod)
			}
			// Compared, never printed: a failure message must not show it.
			if string(cfg.OpenBaoToken) != fakeToken {
				t.Error("OpenBaoToken is not the configured token, trimmed")
			}
			if cfg.OpenBaoRole != "" || cfg.OpenBaoJWTFile != "" || cfg.OpenBaoAuthPath != "" {
				t.Error("a Kubernetes field is set with the token method")
			}
			secrettest.AssertFormatHidden(t, cfg, fakeToken)
			secrettest.AssertFormatHidden(t, &cfg, fakeToken)
		})
	}
}

// TestLoadRefusals covers every refused configuration around the auth
// method. Each must name the variables at fault and never carry the token
// or the refused value itself.
func TestLoadRefusals(t *testing.T) {
	t.Parallel()

	const badMethod = "approle-not-allowed-value-5b7"
	tests := map[string]struct {
		env    map[string]string
		want   []string
		hidden []string
	}{
		"unknown auth method": {
			env:    requiredValues(map[string]string{"OPENBAO_AUTH_METHOD": badMethod}),
			want:   []string{"OPENBAO_AUTH_METHOD", "kubernetes", "token"},
			hidden: []string{badMethod},
		},
		"unknown auth method with a token set": {
			env:    tokenValues(map[string]string{"OPENBAO_AUTH_METHOD": badMethod}),
			want:   []string{"OPENBAO_AUTH_METHOD"},
			hidden: []string{badMethod, fakeToken},
		},
		"auth method in another case": {
			env:  requiredValues(map[string]string{"OPENBAO_AUTH_METHOD": "Kubernetes"}),
			want: []string{"OPENBAO_AUTH_METHOD"},
		},
		"token method without a token": {
			env:  tokenValues(map[string]string{"OPENBAO_TOKEN": ""}),
			want: []string{"OPENBAO_TOKEN"},
		},
		"token and its file twin both set": {
			env:    tokenValues(map[string]string{"OPENBAO_TOKEN_FILE": "/run/secrets/openbao-token"}),
			want:   []string{"OPENBAO_TOKEN", "OPENBAO_TOKEN_FILE"},
			hidden: []string{fakeToken},
		},
		"token file that is empty": {
			env: tokenValues(map[string]string{
				"OPENBAO_TOKEN":      "",
				"OPENBAO_TOKEN_FILE": writeTokenFile(t, "\n"),
			}),
			want: []string{"OPENBAO_TOKEN_FILE"},
		},
		"token method with every Kubernetes variable": {
			env: tokenValues(map[string]string{
				"OPENBAO_ROLE":      "feed-role-value",
				"OPENBAO_JWT_FILE":  "/jwt-file-value",
				"OPENBAO_AUTH_PATH": "auth-path-value",
			}),
			want:   []string{"OPENBAO_ROLE", "OPENBAO_JWT_FILE", "OPENBAO_AUTH_PATH"},
			hidden: []string{fakeToken, "feed-role-value", "/jwt-file-value", "auth-path-value"},
		},
		"token method with OPENBAO_ROLE alone": {
			env:    tokenValues(map[string]string{"OPENBAO_ROLE": "feed"}),
			want:   []string{"OPENBAO_ROLE"},
			hidden: []string{fakeToken},
		},
		"token method with OPENBAO_JWT_FILE alone": {
			env:    tokenValues(map[string]string{"OPENBAO_JWT_FILE": "/var/run/secrets/openbao/token"}),
			want:   []string{"OPENBAO_JWT_FILE"},
			hidden: []string{fakeToken},
		},
		"token method with OPENBAO_AUTH_PATH alone": {
			env:    tokenValues(map[string]string{"OPENBAO_AUTH_PATH": "kubernetes"}),
			want:   []string{"OPENBAO_AUTH_PATH"},
			hidden: []string{fakeToken},
		},
		"kubernetes method with a token": {
			env:    requiredValues(map[string]string{"OPENBAO_TOKEN": fakeToken}),
			want:   []string{"OPENBAO_TOKEN"},
			hidden: []string{fakeToken},
		},
		"default method with a token": {
			env:    requiredValues(map[string]string{"OPENBAO_AUTH_METHOD": "", "OPENBAO_TOKEN": fakeToken}),
			want:   []string{"OPENBAO_TOKEN"},
			hidden: []string{fakeToken},
		},
		"kubernetes method with a token file": {
			env:  requiredValues(map[string]string{"OPENBAO_TOKEN_FILE": writeTokenFile(t, fakeToken)}),
			want: []string{"OPENBAO_TOKEN_FILE"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := Load(env(tc.env))
			if err == nil {
				t.Fatal("Load() = nil error, want a refusal")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Load() error does not name %s", want)
				}
			}
			// The fake token is checked in every case, set or not.
			secrettest.AssertHidden(t, err, fakeToken)
			for _, hidden := range tc.hidden {
				secrettest.AssertHidden(t, err, hidden)
			}
		})
	}
}

// TestLoadTokenMethodNeedsNoKubernetesVariables proves OPENBAO_ROLE and
// OPENBAO_JWT_FILE are not required with the token method.
func TestLoadTokenMethodNeedsNoKubernetesVariables(t *testing.T) {
	t.Parallel()

	if _, err := Load(env(tokenValues(nil))); err != nil {
		t.Fatalf("Load() error = %v, want none without OPENBAO_ROLE and OPENBAO_JWT_FILE", err)
	}
}

// TestLoadIgnoresTokenPath proves the removed TOKEN_PATH is not read: no
// alias, no migration.
func TestLoadIgnoresTokenPath(t *testing.T) {
	t.Parallel()

	cfg, err := Load(env(requiredValues(map[string]string{"TOKEN_PATH": "old/data/path"})))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.OpenBaoSecretPath != DefaultSecretPath {
		t.Errorf("OpenBaoSecretPath = %q, want the default: TOKEN_PATH is removed", cfg.OpenBaoSecretPath)
	}
}
