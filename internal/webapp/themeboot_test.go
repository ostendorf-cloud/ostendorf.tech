package webapp

import (
	"testing"
	"testing/fstest"
)

func TestExtractThemeBootScript(t *testing.T) {
	tests := map[string]struct {
		html     string
		wantText string
		wantOK   bool
	}{
		"finds the script between the tags": {
			html:     `<html><head><script id="theme-boot">console.log(1)</script></head></html>`,
			wantText: "console.log(1)",
			wantOK:   true,
		},
		"no marker at all": {
			html:   `<html><head></head></html>`,
			wantOK: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := extractThemeBootScript(tc.html)
			if ok != tc.wantOK || got != tc.wantText {
				t.Errorf("extractThemeBootScript(...) = (%q, %v), want (%q, %v)", got, ok, tc.wantText, tc.wantOK)
			}
		})
	}
}

func TestFirstThemeBootScript(t *testing.T) {
	t.Run("finds it across several files", func(t *testing.T) {
		fsys := fstest.MapFS{
			"de/404.html":   &fstest.MapFile{Data: []byte(`<html><body>not found</body></html>`)},
			"de/index.html": &fstest.MapFile{Data: []byte(`<html><head><script id="theme-boot">theScript()</script></head></html>`)},
		}
		got, err := firstThemeBootScript(fsys)
		if err != nil {
			t.Fatalf("firstThemeBootScript() error = %v", err)
		}
		if got != "theScript()" {
			t.Errorf("firstThemeBootScript() = %q, want %q", got, "theScript()")
		}
	})

	t.Run("errors when no page carries it", func(t *testing.T) {
		fsys := fstest.MapFS{
			"de/404.html": &fstest.MapFile{Data: []byte(`<html><body>not found</body></html>`)},
		}
		if _, err := firstThemeBootScript(fsys); err == nil {
			t.Error("firstThemeBootScript() error = nil, want an error")
		}
	})
}

func TestThemeBootScriptHash(t *testing.T) {
	fixture := func(script string) fstest.MapFS {
		return fstest.MapFS{
			"de/index.html": &fstest.MapFile{Data: []byte(`<html><head><script id="theme-boot">` + script + `</script></head></html>`)},
		}
	}

	t.Run("is stable and non-empty for a known fixture", func(t *testing.T) {
		fsys := fixture("console.log(1)")
		first, err := themeBootScriptHash(fsys)
		if err != nil {
			t.Fatalf("themeBootScriptHash() error = %v", err)
		}
		if first == "" {
			t.Fatal("themeBootScriptHash() = \"\", want a non-empty hash")
		}
		second, err := themeBootScriptHash(fsys)
		if err != nil {
			t.Fatalf("themeBootScriptHash() error = %v", err)
		}
		if first != second {
			t.Errorf("themeBootScriptHash() = %q then %q, want the same hash both times", first, second)
		}
	})

	t.Run("differs for a different script", func(t *testing.T) {
		a, err := themeBootScriptHash(fixture("console.log(1)"))
		if err != nil {
			t.Fatalf("themeBootScriptHash() error = %v", err)
		}
		b, err := themeBootScriptHash(fixture("console.log(2)"))
		if err != nil {
			t.Fatalf("themeBootScriptHash() error = %v", err)
		}
		if a == b {
			t.Errorf("themeBootScriptHash() = %q for two different scripts, want them to differ", a)
		}
	})
}
