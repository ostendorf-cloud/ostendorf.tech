package depsync

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// renovateConfig is the part of renovate.json these tests read.
type renovateConfig struct {
	CustomManagers []struct {
		CustomType          string   `json:"customType"`
		ManagerFilePatterns []string `json:"managerFilePatterns"`
		MatchStrings        []string `json:"matchStrings"`
	} `json:"customManagers"`
	PackageRules []struct {
		MatchPackageNames []string `json:"matchPackageNames"`
		MatchManagers     []string `json:"matchManagers"`
		GroupName         string   `json:"groupName"`
	} `json:"packageRules"`
}

func loadRenovate(t *testing.T) renovateConfig {
	t.Helper()
	raw, err := os.ReadFile("../../renovate.json")
	if err != nil {
		t.Fatal(err)
	}
	var c renovateConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("parse renovate.json: %v", err)
	}
	return c
}

// TestRenovateReadsTheDockerfilePnpm proves a custom regex manager of
// renovate.json matches the Dockerfile's PNPM_VERSION argument and extracts
// the npm datasource, the depName pnpm and the version mise.toml pins, so
// Renovate updates the Dockerfile in the same run as mise.toml. Renovate's
// regex syntax for named groups, (?<name>...), is also Go's.
func TestRenovateReadsTheDockerfilePnpm(t *testing.T) {
	t.Parallel()

	docker, err := os.ReadFile("../../Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	mise, err := os.ReadFile("../../mise.toml")
	if err != nil {
		t.Fatal(err)
	}
	pin := regexp.MustCompile(`(?m)^pnpm = "([^"]+)"`).FindStringSubmatch(string(mise))
	if pin == nil {
		t.Fatal(`mise.toml has no 'pnpm = "<version>"' line`)
	}

	found := false
	for _, m := range loadRenovate(t).CustomManagers {
		if m.CustomType != "regex" || !slices.Contains(m.ManagerFilePatterns, "/^Dockerfile$/") {
			continue
		}
		for _, s := range m.MatchStrings {
			re, err := regexp.Compile(s)
			if err != nil {
				t.Fatalf("matchString %q does not compile: %v", s, err)
			}
			got := re.FindStringSubmatch(string(docker))
			if got == nil {
				continue
			}
			named := map[string]string{}
			for i, name := range re.SubexpNames() {
				if name != "" {
					named[name] = got[i]
				}
			}
			if named["datasource"] != "npm" || named["depName"] != "pnpm" || named["currentValue"] != pin[1] {
				t.Errorf("the Dockerfile manager extracts %v, want datasource npm, depName pnpm, version %s",
					named, pin[1])
			}
			found = true
		}
	}
	if !found {
		t.Fatal("no regex manager of renovate.json matches the Dockerfile's PNPM_VERSION argument")
	}
}

// TestRenovateGroupsBothPnpmPins proves one package rule puts the mise.toml
// pin and the Dockerfile argument into one pull request, so the two never
// differ on main.
func TestRenovateGroupsBothPnpmPins(t *testing.T) {
	t.Parallel()

	for _, r := range loadRenovate(t).PackageRules {
		if !slices.Contains(r.MatchPackageNames, "pnpm") {
			continue
		}
		if strings.TrimSpace(r.GroupName) == "" {
			t.Fatal("the pnpm rule has no groupName: mise.toml and the Dockerfile would move in two pull requests")
		}
		for _, manager := range []string{"mise", "custom.regex"} {
			if !slices.Contains(r.MatchManagers, manager) {
				t.Errorf("the pnpm rule does not match the %s manager", manager)
			}
		}
		return
	}
	t.Fatal("renovate.json has no package rule for pnpm")
}
