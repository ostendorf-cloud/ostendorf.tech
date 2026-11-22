package envsync

import (
	"os"
	"regexp"
	"testing"

	"github.com/ostendorf-cloud/govue/analytics"
	"github.com/ostendorf-cloud/govue/mail"
)

const envExamplePath = "../../.env.example"

// envRead matches the name of every variable internal/config and internal/feedconfig
// read through govue's envconf loader (loader.String("X", ...), loader.SecretOrFile("X")
// and so on) or through the resolved lookup.
var envRead = regexp.MustCompile(`(?:loader\.(\w+)|resolved)\("([A-Z][A-Z0-9_]*)"`)

// govueEnv lists the variables govue's analytics.LoadConfig and mail.LoadConfig read on
// the website's behalf. They never appear as a literal in this repository's own Go code,
// so a govue upgrade that adds or renames one shows up here as a compile error or a
// failed test. mail.LoadConfig reads SMTP_PASSWORD through a _FILE twin, SMTP_USERNAME
// without one.
var govueEnv = []string{
	analytics.EnvURL,
	analytics.EnvWebsiteID,
	analytics.EnvTimeout,
	analytics.EnvQueueSize,
	mail.EnvHost,
	mail.EnvPort,
	mail.EnvUsername,
	mail.EnvPassword,
	mail.EnvPassword + "_FILE",
}

// commentLine matches a whole YAML comment line, which may name ${VAR} forms as prose.
var commentLine = regexp.MustCompile(`(?m)^\s*#.*$`)

// documented reports whether .env.example has a line for name, as an assignment or as a
// commented out assignment.
func documented(example, name string) bool {
	return regexp.MustCompile(`(?m)^#? ?` + regexp.QuoteMeta(name) + `=`).MatchString(example)
}

// TestEnvExampleDocumentsEveryVariable fails when a variable read by internal/config or
// internal/feedconfig, or one docker-compose.yaml forwards, has no line in .env.example.
func TestEnvExampleDocumentsEveryVariable(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(envExamplePath)
	if err != nil {
		t.Fatal(err)
	}
	example := string(raw)

	type variable struct{ name, source string }
	var vars []variable
	for _, src := range []string{"../config/config.go", "../feedconfig/config.go"} {
		code, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		matches := envRead.FindAllStringSubmatch(string(code), -1)
		if len(matches) == 0 {
			t.Fatalf("found no variable read in %s: the pattern no longer matches the code", src)
		}
		for _, m := range matches {
			vars = append(vars, variable{m[2], src})
			if m[1] == "SecretOrFile" || m[1] == "RequiredSecretOrFile" {
				vars = append(vars, variable{m[2] + "_FILE", src})
			}
		}
	}
	for _, name := range govueEnv {
		vars = append(vars, variable{name, "govue analytics.LoadConfig"})
	}

	compose, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatal(err)
	}
	code := commentLine.ReplaceAllString(string(compose), "")
	for _, m := range interpolation.FindAllStringSubmatch(code, -1) {
		vars = append(vars, variable{m[1], "docker-compose.yaml"})
	}

	for _, v := range vars {
		if !documented(example, v.name) {
			t.Errorf(".env.example has no line for %s, read by %s", v.name, v.source)
		}
	}
}
