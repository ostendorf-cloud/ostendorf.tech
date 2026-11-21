// Package envsync checks that the files which describe the environment of the website
// stay in step with the code that reads it: docker-compose.yaml and .env.example.
package envsync

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
)

const composePath = "../../docker-compose.yaml"

// interpolation matches the one compose form docker-compose.yaml may use: ${VAR} or
// ${VAR:-default}. Anything else fails the test, so a new form is looked at, not
// silently skipped.
var interpolation = regexp.MustCompile(`\$\{([A-Z][A-Z0-9_]*)(?::-([^}]*))?\}`)

// expand applies compose's ${VAR:-default} rule: unset or empty takes the default.
func expand(value string, env map[string]string) (string, error) {
	out := interpolation.ReplaceAllStringFunc(value, func(m string) string {
		sub := interpolation.FindStringSubmatch(m)
		if got := env[sub[1]]; got != "" {
			return got
		}
		return sub[2]
	})
	if strings.Contains(out, "${") {
		return "", fmt.Errorf("unsupported interpolation in %q", value)
	}
	return out, nil
}

var (
	listenAddrLine = regexp.MustCompile(`(?m)^\s+LISTEN_ADDR:\s*"([^"]*)"\s*$`)
	portLine       = regexp.MustCompile(`(?m)^\s+-\s*"([^"]+)"\s*$`)
)

// TestComposePublishedPortFollowsListenAddr proves the container's LISTEN_ADDR and the
// published host and container port come from the same value, for the default and for
// a changed one, and that the container really listens where compose publishes.
func TestComposePublishedPortFollowsListenAddr(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)

	listen := listenAddrLine.FindStringSubmatch(text)
	if listen == nil {
		t.Fatal("docker-compose.yaml does not pass LISTEN_ADDR to the container")
	}
	ports := portLine.FindAllStringSubmatch(text, -1)
	if len(ports) != 1 {
		t.Fatalf("docker-compose.yaml publishes %d ports, want exactly the one public port", len(ports))
	}

	cases := []struct {
		name     string
		env      map[string]string
		wantPort string
	}{
		{name: "default", env: map[string]string{}, wantPort: strings.TrimPrefix(config.DefaultListenAddr, ":")},
		{name: "changed", env: map[string]string{"LISTEN_PORT": "8123"}, wantPort: "8123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			addr, err := expand(listen[1], tc.env)
			if err != nil {
				t.Fatal(err)
			}
			published, err := expand(ports[0][1], tc.env)
			if err != nil {
				t.Fatal(err)
			}

			if want := ":" + tc.wantPort; addr != want {
				t.Fatalf("container LISTEN_ADDR = %q, want %q", addr, want)
			}
			if want := "127.0.0.1:" + tc.wantPort + ":" + tc.wantPort; published != want {
				t.Fatalf("published port = %q, want %q (loopback, host port equal to the container port)", published, want)
			}

			cfg, err := config.Load(func(key string) string {
				switch key {
				case "LISTEN_ADDR":
					return addr
				case "CSRF_KEY":
					return strings.Repeat("ab", 32)
				}
				return ""
			})
			if err != nil {
				t.Fatalf("config.Load with compose's environment: %v", err)
			}
			if cfg.ListenAddr != addr {
				t.Fatalf("config.ListenAddr = %q, want %q", cfg.ListenAddr, addr)
			}
		})
	}
}
