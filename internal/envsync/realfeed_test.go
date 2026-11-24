package envsync

import (
	"os"
	"regexp"
	"slices"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/feedconfig"
)

var exampleRealURL = regexp.MustCompile(`(?m)^# FEED_SERVICE_URL=(http://realfeed:\d+)$`)

// TestComposeRealFeed proves the real feed service stays behind its own profile, off the
// host, on the one port the website is told to use, that the secret token is only ever a
// pass-through, and that cmd/feed's own configuration accepts exactly what compose hands it.
func TestComposeRealFeed(t *testing.T) {
	t.Parallel()

	services := loadCompose(t).Services
	real, ok := services["realfeed"]
	if !ok {
		t.Fatal("docker-compose.yaml has no realfeed service")
	}
	fake := services["fakefeed"]
	if !slices.Equal(real.Profiles, []string{"realfeed"}) {
		t.Errorf("realfeed profiles = %v, want [realfeed] so a plain compose up never starts it", real.Profiles)
	}
	if slices.Equal(real.Profiles, fake.Profiles) {
		t.Errorf("realfeed and fakefeed share the profiles %v: one choice must start one feed", real.Profiles)
	}
	if len(real.Ports) != 0 {
		t.Errorf("realfeed publishes %v, want no host port", real.Ports)
	}
	if real.Build.Target != "feed" {
		t.Errorf("realfeed builds target %q, want feed", real.Build.Target)
	}

	// The token is a pass-through from .env or the shell, never a literal.
	if got := real.Environment["OPENBAO_TOKEN"]; got != "${OPENBAO_TOKEN:-}" {
		t.Errorf("realfeed OPENBAO_TOKEN = %q, want the pass-through ${OPENBAO_TOKEN:-}", got)
	}

	// Compose's environment with .env empty, and with the two required values given.
	for name, given := range map[string]map[string]string{
		"empty .env": {},
		"token set":  {"OPENBAO_ADDR": "http://host.containers.internal:8200", "OPENBAO_TOKEN": "s.test"},
		"paging set": {
			"OPENBAO_ADDR": "http://host.containers.internal:8200", "OPENBAO_TOKEN": "s.test",
			"POST_COUNT": "0", "POSTS_PER_REQUEST": "100",
		},
	} {
		env := map[string]string{}
		for k, v := range real.Environment {
			got, err := expand(v, given)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			env[k] = got
		}
		if env["OPENBAO_AUTH_METHOD"] != "token" {
			t.Errorf("%s: OPENBAO_AUTH_METHOD = %q, want the default token", name, env["OPENBAO_AUTH_METHOD"])
		}
		cfg, err := feedconfig.Load(func(key string) string { return env[key] })
		switch name {
		case "token set":
			if err != nil {
				t.Errorf("feedconfig.Load rejects compose's environment: %v", err)
			} else if cfg.ListenAddr != real.Environment["LISTEN_ADDR"] {
				t.Errorf("feed listens on %q, compose says %q", cfg.ListenAddr, real.Environment["LISTEN_ADDR"])
			}
		case "paging set":
			// Both paging values reach the feed service from .env.
			if err != nil {
				t.Errorf("feedconfig.Load rejects compose's environment with paging values: %v", err)
			} else if cfg.PostCount != 0 || cfg.PostsPerRequest != 100 {
				t.Errorf("POST_COUNT, POSTS_PER_REQUEST arrive as %d, %d; want 0, 100 from .env", cfg.PostCount, cfg.PostsPerRequest)
			}
		default:
			if err == nil {
				t.Errorf("%s: feedconfig.Load accepts an environment with no OPENBAO_ADDR and no token", name)
			}
		}
	}

	// The one port: the service's LISTEN_ADDR and the URL .env.example tells the owner to set.
	example, err := os.ReadFile(envExamplePath)
	if err != nil {
		t.Fatal(err)
	}
	u := exampleRealURL.FindSubmatch(example)
	if u == nil {
		t.Fatal(".env.example has no line '# FEED_SERVICE_URL=http://realfeed:<port>'")
	}
	if want := "http://realfeed" + real.Environment["LISTEN_ADDR"]; string(u[1]) != want {
		t.Errorf(".env.example sets FEED_SERVICE_URL=%s, want %s to match realfeed's LISTEN_ADDR", u[1], want)
	}
	if !regexp.MustCompile(`(?m)^OPENBAO_TOKEN=$`).Match(example) {
		t.Error(".env.example must carry an empty OPENBAO_TOKEN= line, never a value")
	}
}
