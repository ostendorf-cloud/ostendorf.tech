package envsync

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const (
	dockerfilePath = "../../Dockerfile"
	workflowsDir   = "../../.github/workflows"
	fakefeedMain   = "../../cmd/fakefeed/main.go"
)

var (
	fakefeedDefaultAddr = regexp.MustCompile(`defaultListenAddr\s*=\s*"(:\d+)"`)
	exampleFakeURL      = regexp.MustCompile(`(?m)^# FEED_SERVICE_URL=(http://fakefeed:\d+)$`)
)

// TestComposeFakeFeed proves the development only fake feed stays behind its profile,
// off the host, and on the one port the website is told to use.
func TestComposeFakeFeed(t *testing.T) {
	t.Parallel()

	services := loadCompose(t).Services
	fake, ok := services["fakefeed"]
	if !ok {
		t.Fatal("docker-compose.yaml has no fakefeed service")
	}
	if !slices.Equal(fake.Profiles, []string{"feed"}) {
		t.Errorf("fakefeed profiles = %v, want [feed] so a plain compose up never starts it", fake.Profiles)
	}
	if len(fake.Ports) != 0 {
		t.Errorf("fakefeed publishes %v, want no host port", fake.Ports)
	}
	if fake.Build.Target != "fakefeed" {
		t.Errorf("fakefeed builds target %q, want fakefeed", fake.Build.Target)
	}
	if got := services["website"].Profiles; len(got) != 0 {
		t.Errorf("website has profiles %v, want none: it must start with a plain compose up", got)
	}

	// Without the profile the website gets no feed URL: the section stays off.
	website := services["website"]
	raw, ok := website.Environment["FEED_SERVICE_URL"]
	if !ok {
		t.Fatal("docker-compose.yaml does not pass FEED_SERVICE_URL to the website")
	}
	if got, err := expand(raw, map[string]string{}); err != nil || got != "" {
		t.Errorf("FEED_SERVICE_URL without .env = %q, %v; want empty", got, err)
	}

	// The one port: the service's LISTEN_ADDR, the program's default and the URL
	// .env.example tells the owner to set.
	listen := fake.Environment["LISTEN_ADDR"]
	code, err := os.ReadFile(fakefeedMain)
	if err != nil {
		t.Fatal(err)
	}
	m := fakefeedDefaultAddr.FindSubmatch(code)
	if m == nil {
		t.Fatalf("no defaultListenAddr in %s: the pattern no longer matches the code", fakefeedMain)
	}
	if listen != string(m[1]) {
		t.Errorf("fakefeed LISTEN_ADDR = %q, but cmd/fakefeed defaults to %q", listen, m[1])
	}
	example, err := os.ReadFile(envExamplePath)
	if err != nil {
		t.Fatal(err)
	}
	u := exampleFakeURL.FindSubmatch(example)
	if u == nil {
		t.Fatal(".env.example has no line '# FEED_SERVICE_URL=http://fakefeed:<port>'")
	}
	if want := "http://fakefeed" + listen; string(u[1]) != want {
		t.Errorf(".env.example sets FEED_SERVICE_URL=%s, want %s to match fakefeed's LISTEN_ADDR", u[1], want)
	}
}

// stage is one Dockerfile build stage with its comments removed.
type stage struct {
	name, text string
	deps       []string
}

var (
	fromLine     = regexp.MustCompile(`(?mi)^FROM\s+(?:--\S+\s+)*(\S+)(?:\s+AS\s+(\S+))?`)
	copyFromFlag = regexp.MustCompile(`--from=(\S+)`)
)

// parseStages splits the Dockerfile into its stages, in file order.
func parseStages(t *testing.T) []stage {
	t.Helper()
	raw, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatal(err)
	}
	code := commentLine.ReplaceAllString(string(raw), "")
	idx := fromLine.FindAllStringSubmatchIndex(code, -1)
	var stages []stage
	for i, loc := range idx {
		end := len(code)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		s := stage{text: code[loc[0]:end]}
		base := code[loc[2]:loc[3]]
		if loc[4] >= 0 {
			s.name = code[loc[4]:loc[5]]
		}
		s.deps = append(s.deps, base)
		for _, c := range copyFromFlag.FindAllStringSubmatch(s.text, -1) {
			s.deps = append(s.deps, c[1])
		}
		stages = append(stages, s)
	}
	return stages
}

// closure returns the named stage and every stage it depends on, through FROM and
// COPY --from: exactly what BuildKit evaluates for that target.
func closure(stages []stage, target string) []stage {
	byName := map[string]stage{}
	for _, s := range stages {
		byName[s.name] = s
	}
	var out []stage
	seen := map[string]bool{}
	var walk func(name string)
	walk = func(name string) {
		s, ok := byName[name]
		if !ok || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, s)
		for _, d := range s.deps {
			walk(d)
		}
	}
	walk(target)
	return out
}

// TestFakeFeedNotInProductionImages proves neither production target evaluates a
// stage that mentions the fake, that the fake has a target of its own, and that
// "frontend" stays the last stage a bare build resolves to.
func TestFakeFeedNotInProductionImages(t *testing.T) {
	t.Parallel()

	stages := parseStages(t)
	if len(stages) == 0 || stages[len(stages)-1].name != "frontend" {
		t.Fatal("the last Dockerfile stage must be frontend, what a build without --target resolves to")
	}
	if !slices.ContainsFunc(stages, func(s stage) bool { return s.name == "fakefeed" }) {
		t.Fatal("the Dockerfile has no fakefeed target")
	}
	for _, target := range []string{"frontend", "api", "feed"} {
		got := closure(stages, target)
		if len(got) < 2 {
			t.Fatalf("target %s resolved to %d stages: the parser no longer understands the Dockerfile", target, len(got))
		}
		for _, s := range got {
			if strings.Contains(strings.ToLower(s.name+"\n"+s.text), "fakefeed") {
				t.Errorf("target %s evaluates stage %q, which references the fake feed", target, s.name)
			}
		}
	}
}

// TestFakeFeedNotBuiltByWorkflows proves no workflow builds or publishes the fake.
func TestFakeFeedNotBuiltByWorkflows(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.y*ml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no workflow files found in %s: %v", workflowsDir, err)
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(raw)), "fakefeed") {
			t.Errorf("%s references fakefeed: the fake must never be built or published", f)
		}
	}
}
