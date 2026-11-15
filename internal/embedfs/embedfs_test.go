package embedfs

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

// TestSubtreeFailsWhenTheDirectoryIsMissing reproduces the defect
// fs.Sub itself cannot catch: it builds a filesystem rooted at a given
// directory lazily, with no check that the directory actually exists, for
// every fs.FS that does not implement fs.SubFS itself (embed.FS and
// fstest.MapFS included). A bare fs.Sub(fsys, dir) therefore returns no
// error here, and only fails on the first read against the result, far
// from where the real problem is. Subtree exists to catch this at the
// call site instead.
func TestSubtreeFailsWhenTheDirectoryIsMissing(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		fsys fs.FS
		dir  string
	}{
		"dist directory missing": {
			fsys: fstest.MapFS{
				"prerendered/de/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
			},
			dir: "dist",
		},
		"prerendered directory missing": {
			fsys: fstest.MapFS{
				"dist/assets/app.js": &fstest.MapFile{Data: []byte("console.log(1)")},
			},
			dir: "prerendered",
		},
		"fsys entirely empty": {
			fsys: fstest.MapFS{},
			dir:  "dist",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Subtree(tc.fsys, tc.dir); err == nil {
				t.Fatalf("Subtree(fsys, %q) error = nil, want an error naming the missing directory", tc.dir)
			}
		})
	}
}

// TestSubtreeNamesTheMissingDirectory covers the acceptance criterion
// that the failure names which directory is missing, not only that
// something is missing.
func TestSubtreeNamesTheMissingDirectory(t *testing.T) {
	t.Parallel()
	_, err := Subtree(fstest.MapFS{}, "prerendered")
	if err == nil {
		t.Fatal("Subtree() error = nil, want an error naming \"prerendered\"")
	}
	if !strings.Contains(err.Error(), "prerendered") {
		t.Errorf("error = %q, want it to name %q", err.Error(), "prerendered")
	}
}

// TestSubtreeSucceedsForAnExistingDirectory proves the check adds no
// false positive: a directory that is actually there still resolves to a
// usable subtree, with its own content readable.
func TestSubtreeSucceedsForAnExistingDirectory(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"dist/assets/app.js": &fstest.MapFile{Data: []byte("console.log(1)")},
	}
	sub, err := Subtree(fsys, "dist")
	if err != nil {
		t.Fatalf("Subtree() error = %v, want nil", err)
	}
	if _, err := fs.Stat(sub, "assets/app.js"); err != nil {
		t.Errorf("fs.Stat(sub, \"assets/app.js\") error = %v, want nil", err)
	}
}
