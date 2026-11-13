package feedclient

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// roundTripFunc adapts a function to http.RoundTripper, so every test
// below fakes the feed service in process: none of them opens a real
// socket.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// serve builds a client whose transport calls h in process, the same
// pattern internal/instagram's own tests use for their fake Meta.
func serve(h http.Handler) *Client {
	c := New("https://feed.internal")
	c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec.Result(), nil
	})}
	return c
}

// Minimal valid image headers, enough for http.DetectContentType.
var jpegBytes = append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{1}, 100)...)

func TestClientPosts(t *testing.T) {
	t.Parallel()

	var gotPath string
	c := serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"posts":[{"id":"1","alt":"Lichtpult","permalink":"https://www.instagram.com/p/A/"}]}`)
	}))

	posts, err := c.Posts(t.Context())
	if err != nil {
		t.Fatalf("Posts() error = %v", err)
	}
	if gotPath != "/api/v1/posts" {
		t.Errorf("path = %q, want /api/v1/posts", gotPath)
	}
	want := []Post{{ID: "1", Alt: "Lichtpult", Permalink: "https://www.instagram.com/p/A/"}}
	if len(posts) != 1 || posts[0] != want[0] {
		t.Errorf("Posts() = %+v, want %+v", posts, want)
	}
}

func TestClientPostsUnexpectedStatus(t *testing.T) {
	t.Parallel()

	c := serve(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	if _, err := c.Posts(t.Context()); err == nil {
		t.Fatal("Posts() error = nil, want one for a 503")
	}
}

func TestClientPostsRejectsMalformedBody(t *testing.T) {
	t.Parallel()

	c := serve(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `not json`)
	}))

	if _, err := c.Posts(t.Context()); err == nil {
		t.Fatal("Posts() error = nil, want a decode error")
	}
}

func TestClientImageFetchesAndValidates(t *testing.T) {
	t.Parallel()

	var gotPath string
	c := serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("ETag", `"abc"`)
		_, _ = w.Write(jpegBytes)
	}))

	data, contentType, etag, err := c.Image(t.Context(), "1")
	if err != nil {
		t.Fatalf("Image() error = %v", err)
	}
	if gotPath != "/api/v1/posts/1/image" {
		t.Errorf("path = %q, want /api/v1/posts/1/image", gotPath)
	}
	if !bytes.Equal(data, jpegBytes) {
		t.Error("data does not match the served bytes")
	}
	if contentType != "image/jpeg" {
		t.Errorf("contentType = %q, want image/jpeg", contentType)
	}
	if etag != `"abc"` {
		t.Errorf("etag = %q, want the feed service's own", etag)
	}
}

func TestClientImagePathEscapesTheID(t *testing.T) {
	t.Parallel()

	var gotURL *url.URL
	c := serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpegBytes)
	}))

	// A feed service response is already validated upstream (see
	// internal/instagram.ValidID in internal/feed), but this client takes
	// no dependency on that package at all; it only has to build a request
	// that cannot smuggle an extra path segment into the URL, whatever id
	// it is asked for.
	if _, _, _, err := c.Image(t.Context(), "../etc/passwd"); err != nil {
		t.Fatalf("Image() error = %v", err)
	}
	if gotURL == nil {
		t.Fatal("handler never ran")
	}
	// EscapedPath is what reached the wire; Path is what it decodes back
	// to. The escaped slashes must stay literal percent-encodings, not
	// turn into extra path segments once a server (or this test's own
	// in-process handler) parses the request line.
	if segs := strings.Split(strings.Trim(gotURL.EscapedPath(), "/"), "/"); len(segs) != 5 {
		t.Errorf("request path %q has %d segments, want exactly 5 (api/v1/posts/<id>/image)", gotURL.EscapedPath(), len(segs))
	}
	if gotURL.Path != "/api/v1/posts/../etc/passwd/image" {
		// Decoding the escaped id back out must reproduce exactly the id
		// asked for, proving no segment was lost or merged by escaping.
		t.Errorf("decoded path = %q, want the id round tripping through PathEscape", gotURL.Path)
	}
}

func TestClientImageMissingETagIsDerived(t *testing.T) {
	t.Parallel()

	c := serve(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpegBytes)
	}))

	_, _, etag, err := c.Image(t.Context(), "1")
	if err != nil {
		t.Fatalf("Image() error = %v", err)
	}
	if etag == "" || !strings.HasPrefix(etag, `"`) || !strings.HasSuffix(etag, `"`) {
		t.Errorf("etag = %q, want a derived quoted value when the feed service sent none", etag)
	}
}

func TestClientImageRejectsDisallowedContentType(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		declared string
		body     []byte
	}{
		"declared content type not allowed": {"text/html", jpegBytes},
		"declared jpeg but body is html":    {"image/jpeg", []byte("<html><script>alert(1)</script></html>")},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := serve(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", tc.declared)
				_, _ = w.Write(tc.body)
			}))
			if _, _, _, err := c.Image(t.Context(), "1"); err == nil {
				t.Fatal("Image() error = nil, want one for disallowed content")
			}
		})
	}
}

func TestClientImageUnexpectedStatus(t *testing.T) {
	t.Parallel()

	c := serve(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	if _, _, _, err := c.Image(t.Context(), "1"); err == nil {
		t.Fatal("Image() error = nil, want one for a 404")
	}
}

func TestClientImageTooLarge(t *testing.T) {
	t.Parallel()

	huge := append(append([]byte{}, jpegBytes...), make([]byte, MaxImageBytes)...)
	c := serve(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(huge)
	}))

	if _, _, _, err := c.Image(t.Context(), "1"); err == nil {
		t.Fatal("Image() error = nil, want one for a body larger than MaxImageBytes")
	}
}
