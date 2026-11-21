package envsync

import (
	"os"
	"regexp"
	"testing"
)

var (
	dockerfilePnpm = regexp.MustCompile(`corepack prepare pnpm@(\S+) --activate`)
	misePnpm       = regexp.MustCompile(`(?m)^pnpm = "([^"]+)"`)
)

// TestDockerfilePnpmMatchesMise fails when the Dockerfile's frontend stage installs a
// different pnpm than mise.toml pins. mise.toml says the two must match: the lockfile the
// image reads with --frozen-lockfile has to come from the same pnpm a local install uses.
func TestDockerfilePnpmMatchesMise(t *testing.T) {
	t.Parallel()

	docker, err := os.ReadFile("../../Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	mise, err := os.ReadFile("../../mise.toml")
	if err != nil {
		t.Fatal(err)
	}

	d := dockerfilePnpm.FindStringSubmatch(string(docker))
	if d == nil {
		t.Fatal("Dockerfile has no 'corepack prepare pnpm@<version> --activate' line")
	}
	m := misePnpm.FindStringSubmatch(string(mise))
	if m == nil {
		t.Fatal(`mise.toml has no 'pnpm = "<version>"' line`)
	}
	if d[1] != m[1] {
		t.Fatalf("Dockerfile installs pnpm %s but mise.toml pins pnpm %s; they must match", d[1], m[1])
	}
}
