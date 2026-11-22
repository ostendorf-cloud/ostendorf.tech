package main

import (
	"bytes"
	"image"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	// Decoders for image.Decode.
	_ "image/jpeg"
	_ "image/png"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/feedclient"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/instagram"
)

func newServer(t *testing.T, n int) *httptest.Server {
	t.Helper()
	fixtures, err := buildFixtures(n)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(newHandler(fixtures))
	t.Cleanup(srv.Close)
	return srv
}

// TestRealFeedClientAcceptsPosts runs the website's own client against the fake: the
// decoding and every check the client makes must pass unchanged.
func TestRealFeedClientAcceptsPosts(t *testing.T) {
	t.Parallel()

	srv := newServer(t, defaultPostCount)
	posts, err := feedclient.New(srv.URL).Posts(t.Context())
	if err != nil {
		t.Fatalf("Posts() error = %v", err)
	}
	if len(posts) != defaultPostCount {
		t.Fatalf("got %d posts, want %d", len(posts), defaultPostCount)
	}
	seen := map[string]bool{}
	for _, p := range posts {
		if !instagram.ValidID(p.ID) {
			t.Errorf("post id %q is not a valid post id", p.ID)
		}
		if seen[p.ID] {
			t.Errorf("post id %q repeats", p.ID)
		}
		seen[p.ID] = true
		if !strings.HasPrefix(p.Alt, "Sample post ") || !strings.Contains(p.Alt, "fixture for local testing") {
			t.Errorf("alt %q does not say plainly that it is a sample", p.Alt)
		}
		if !strings.HasPrefix(p.Permalink, "https://www.instagram.com/") {
			t.Errorf("permalink %q lacks the prefix the real feed guarantees", p.Permalink)
		}
	}
}

// TestRealFeedClientAcceptsImages proves the pictures pass the client's content type,
// size and sniffing checks, and are real, decodable images in both formats.
func TestRealFeedClientAcceptsImages(t *testing.T) {
	t.Parallel()

	srv := newServer(t, 4)
	c := feedclient.New(srv.URL)
	posts, err := c.Posts(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	types := map[string]bool{}
	for _, p := range posts {
		data, contentType, etag, err := c.Image(t.Context(), p.ID)
		if err != nil {
			t.Fatalf("Image(%s) error = %v", p.ID, err)
		}
		if len(data) == 0 || len(data) > feedclient.MaxImageBytes {
			t.Errorf("Image(%s) is %d bytes, want 1 to %d", p.ID, len(data), feedclient.MaxImageBytes)
		}
		if etag == "" {
			t.Errorf("Image(%s) has no ETag", p.ID)
		}
		types[contentType] = true
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Image(%s) does not decode: %v", p.ID, err)
		}
		if b := img.Bounds(); b.Dx() != imageSize || b.Dy() != imageSize {
			t.Errorf("Image(%s) is %dx%d, want %d square", p.ID, b.Dx(), b.Dy(), imageSize)
		}
	}
	if !types["image/jpeg"] || !types["image/png"] {
		t.Errorf("served types %v, want both image/jpeg and image/png", types)
	}
}

// TestRealSyncerFillsCache runs the website's whole sync cycle against the fake.
func TestRealSyncerFillsCache(t *testing.T) {
	t.Parallel()

	srv := newServer(t, 5)
	cache := &feedclient.Cache{}
	s := &feedclient.Syncer{Client: feedclient.New(srv.URL), Cache: cache, Logger: slog.New(slog.DiscardHandler)}
	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := len(cache.Posts()); got != 5 {
		t.Fatalf("cache holds %d posts, want 5", got)
	}
}

func TestRoutes(t *testing.T) {
	t.Parallel()

	srv := newServer(t, 3)
	cases := []struct {
		name, method, path string
		want               int
	}{
		{"healthz", http.MethodGet, "/healthz", http.StatusOK},
		{"readyz", http.MethodGet, "/readyz", http.StatusOK},
		{"posts", http.MethodGet, "/api/v1/posts", http.StatusOK},
		{"image", http.MethodGet, "/api/v1/posts/sample01/image", http.StatusOK},
		{"unknown image", http.MethodGet, "/api/v1/posts/nope/image", http.StatusNotFound},
		{"unknown route", http.MethodGet, "/other", http.StatusNotFound},
		// Like the real service, whose catch all route answers any other method with 404.
		{"posts only GET", http.MethodPost, "/api/v1/posts", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), tc.method, srv.URL+tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = resp.Body.Close() }()
			_, _ = io.Copy(io.Discard, resp.Body)
			if resp.StatusCode != tc.want {
				t.Fatalf("%s %s = %d, want %d", tc.method, tc.path, resp.StatusCode, tc.want)
			}
		})
	}
}

func TestImageIsCacheable(t *testing.T) {
	t.Parallel()

	srv := newServer(t, 1)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/api/v1/posts/sample01/image", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.Header.Get("ETag") == "" || resp.Header.Get("Cache-Control") != imageMaxAge {
		t.Errorf("headers ETag=%q Cache-Control=%q, want an ETag and %q", resp.Header.Get("ETag"), resp.Header.Get("Cache-Control"), imageMaxAge)
	}
}

func TestParsePostCount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		raw     string
		want    int
		wantErr bool
	}{
		{"", defaultPostCount, false},
		{"1", 1, false},
		{"100", 100, false},
		{"0", 0, true},
		{"101", 0, true},
		{"many", 0, true},
	}
	for _, tc := range cases {
		got, err := parsePostCount(tc.raw)
		if (err != nil) != tc.wantErr || got != tc.want {
			t.Errorf("parsePostCount(%q) = %d, %v; want %d, error %v", tc.raw, got, err, tc.want, tc.wantErr)
		}
	}
}
