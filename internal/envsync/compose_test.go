// Package envsync checks that the files which describe the environment of the website
// stay in step with the code that reads it: docker-compose.yaml and .env.example.
package envsync

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/frontendconfig"
)

const composePath = "../../docker-compose.yaml"

// interpolation matches every compose substitution form that takes a variable name:
// ${VAR}, ${VAR:-x}, ${VAR-x}, ${VAR:?msg}, ${VAR?msg}, ${VAR:+x} and ${VAR+x}. Group 1 is
// the name, group 2 the operator and group 3 its operand.
var interpolation = regexp.MustCompile(`\$\{([A-Z][A-Z0-9_]*)(?:(:?[-?+])([^}]*))?\}`)

// expand applies compose's ${VAR:-default} rule: unset or empty takes the default. It is
// the only form the port and address values may use, so any other form is an error here,
// looked at rather than silently skipped.
func expand(value string, env map[string]string) (string, error) {
	var bad error
	out := interpolation.ReplaceAllStringFunc(value, func(m string) string {
		sub := interpolation.FindStringSubmatch(m)
		if sub[2] != ":-" {
			bad = fmt.Errorf("unsupported interpolation %q in %q", m, value)
			return m
		}
		if got := env[sub[1]]; got != "" {
			return got
		}
		return sub[3]
	})
	if bad != nil {
		return "", bad
	}
	if strings.Contains(out, "${") {
		return "", fmt.Errorf("unsupported interpolation in %q", value)
	}
	return out, nil
}

// composeFile is the part of docker-compose.yaml this package reads.
type composeFile struct {
	Services map[string]struct {
		Build struct {
			Target string `yaml:"target"`
		} `yaml:"build"`
		Environment map[string]string `yaml:"environment"`
		Ports       []string          `yaml:"ports"`
		Profiles    []string          `yaml:"profiles"`
	} `yaml:"services"`
}

// loadCompose reads and unmarshals docker-compose.yaml.
func loadCompose(t *testing.T) composeFile {
	t.Helper()
	raw, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatal(err)
	}
	var c composeFile
	if err := yaml.Unmarshal(raw, &c); err != nil {
		t.Fatalf("parse %s: %v", composePath, err)
	}
	return c
}

// TestComposePublishedPortFollowsListenAddr proves the container's LISTEN_ADDR and the
// published host and container port come from the same value, for the default and for
// a changed one, and that the container really listens where compose publishes.
func TestComposePublishedPortFollowsListenAddr(t *testing.T) {
	t.Parallel()

	website, ok := loadCompose(t).Services["frontend"]
	if !ok {
		t.Fatal("docker-compose.yaml has no frontend service")
	}
	listen, ok := website.Environment["LISTEN_ADDR"]
	if !ok {
		t.Fatal("docker-compose.yaml does not pass LISTEN_ADDR to the container")
	}
	if len(website.Ports) != 1 {
		t.Fatalf("docker-compose.yaml publishes %d ports, want exactly the one public port", len(website.Ports))
	}

	cases := []struct {
		name     string
		env      map[string]string
		wantPort string
	}{
		{name: "default", env: map[string]string{}, wantPort: strings.TrimPrefix(frontendconfig.DefaultListenAddr, ":")},
		{name: "changed", env: map[string]string{"LISTEN_PORT": "8123"}, wantPort: "8123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			addr, err := expand(listen, tc.env)
			if err != nil {
				t.Fatal(err)
			}
			published, err := expand(website.Ports[0], tc.env)
			if err != nil {
				t.Fatal(err)
			}

			if want := ":" + tc.wantPort; addr != want {
				t.Fatalf("container LISTEN_ADDR = %q, want %q", addr, want)
			}
			if want := "127.0.0.1:" + tc.wantPort + ":" + tc.wantPort; published != want {
				t.Fatalf("published port = %q, want %q (loopback, host port equal to the container port)", published, want)
			}

			cfg, err := frontendconfig.Load(func(key string) string {
				v, err := expand(website.Environment[key], tc.env)
				if err != nil {
					t.Fatal(err)
				}
				return v
			})
			if err != nil {
				t.Fatalf("frontendconfig.Load with compose's environment: %v", err)
			}
			if cfg.ListenAddr != addr {
				t.Fatalf("frontendconfig.ListenAddr = %q, want %q", cfg.ListenAddr, addr)
			}
		})
	}
}

// TestComposeOneOriginThroughTheDevProxy proves the stack has one browser
// origin: only the frontend publishes a port, its development proxy points
// at the api service on the port the API listens on, the API trusts the
// frontend's origin, and the switch is accepted by the frontend config.
func TestComposeOneOriginThroughTheDevProxy(t *testing.T) {
	t.Parallel()

	services := loadCompose(t).Services
	front, api := services["frontend"], services["api"]
	if len(api.Ports) != 0 {
		t.Errorf("api publishes %v, want no host port: the browser reaches it through the frontend", api.Ports)
	}
	if len(front.Profiles) != 0 || len(api.Profiles) != 0 {
		t.Error("frontend and api must start with a plain compose up, without a profile")
	}
	proxy := front.Environment[frontendconfig.EnvDevAPIProxy]
	if want := "http://api" + api.Environment["LISTEN_ADDR"]; proxy != want {
		t.Errorf("%s = %q, want %q, the api service's LISTEN_ADDR", frontendconfig.EnvDevAPIProxy, proxy, want)
	}
	if api.Build.Target != "api" || front.Build.Target != "frontend" {
		t.Errorf("targets = %q/%q, want api/frontend", api.Build.Target, front.Build.Target)
	}
	for _, env := range []map[string]string{{}, {"LISTEN_PORT": "8123"}} {
		frontSite, err := expand(front.Environment["SITE_URL"], env)
		if err != nil {
			t.Fatal(err)
		}
		apiSite, err := expand(api.Environment["SITE_URL"], env)
		if err != nil {
			t.Fatal(err)
		}
		port, _ := expand(front.Ports[0], env)
		if apiSite != frontSite || !strings.HasSuffix(port, ":"+strings.TrimPrefix(frontSite, "http://localhost:")) {
			t.Errorf("SITE_URL frontend %q, api %q, port %q: the API must trust the one origin", frontSite, apiSite, port)
		}
	}
	cfg, err := config.Load(func(key string) string {
		v, _ := expand(api.Environment[key], map[string]string{})
		return v
	})
	if err != nil {
		t.Fatalf("config.Load with compose's api environment: %v", err)
	}
	if cfg.ListenAddr != api.Environment["LISTEN_ADDR"] {
		t.Errorf("api ListenAddr = %q", cfg.ListenAddr)
	}
}
