package routes

import (
	"testing"
)

func TestLoadParsesTheCheckedInRouteTable(t *testing.T) {
	t.Parallel()

	table, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := []Route{
		{Name: "home", Slug: "", File: "index.html"},
		{Name: "imprint", Slug: "imprint", File: "imprint/index.html"},
		{Name: "privacy", Slug: "privacy", File: "privacy/index.html"},
		{Name: "agb", Slug: "agb", File: "agb/index.html"},
	}
	if len(table) != len(want) {
		t.Fatalf("Load() returned %d routes, want %d: %+v", len(table), len(want), table)
	}
	for i, r := range want {
		if table[i] != r {
			t.Errorf("Load()[%d] = %+v, want %+v", i, table[i], r)
		}
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input   string
		want    []Route
		wantErr bool
	}{
		"valid table": {
			input: `[{"name":"home","slug":"","file":"index.html"}]`,
			want:  []Route{{Name: "home", Slug: "", File: "index.html"}},
		},
		"empty table": {
			input: `[]`,
			want:  []Route{},
		},
		"malformed json": {
			input:   `{not json`,
			wantErr: true,
		},
		"entry missing name": {
			input:   `[{"name":"","slug":"imprint","file":"imprint/index.html"}]`,
			wantErr: true,
		},
		"entry missing file": {
			input:   `[{"name":"imprint","slug":"imprint","file":""}]`,
			wantErr: true,
		},
		"duplicate slug": {
			input: `[
				{"name":"home","slug":"dup","file":"a.html"},
				{"name":"other","slug":"dup","file":"b.html"}
			]`,
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse([]byte(tc.input))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) error = nil, want an error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tc.input, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("Parse(%q) = %+v, want %+v", tc.input, got, tc.want)
			}
			for i, r := range tc.want {
				if got[i] != r {
					t.Errorf("Parse(%q)[%d] = %+v, want %+v", tc.input, i, got[i], r)
				}
			}
		})
	}
}

func TestParseRejectsNilData(t *testing.T) {
	t.Parallel()

	if _, err := Parse(nil); err == nil {
		t.Fatal("Parse(nil) error = nil, want an error")
	}
}
