package feed

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ostendorf-cloud/govue/health"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/instagram"
)

// discardLogger is a logger that writes nowhere, for tests that care about
// behavior, not log output.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// newTestApp builds an App directly as a struct literal, reaching its
// unexported fields, rather than through New: New would need a real
// feedconfig.Config and build real OpenBao and Meta clients, neither of
// which a handler test wants.
func newTestApp(f *instagram.Feed) *App {
	a := &App{logger: discardLogger(), feed: f, health: health.New()}
	a.health.Add("feed", readinessCheckTimeout, a.checkReady)
	a.handler = a.buildHandler()
	return a
}

func decodePosts(t *testing.T, body []byte) postsResponse {
	t.Helper()
	var resp postsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("response is not the posts shape: %v: %s", err, body)
	}
	return resp
}

func TestHandlePostsEmptyFeed(t *testing.T) {
	t.Parallel()

	a := newTestApp(&instagram.Feed{})
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/posts", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	resp := decodePosts(t, rec.Body.Bytes())
	if resp.Posts == nil || len(resp.Posts) != 0 {
		t.Errorf("posts = %#v, want an empty, non-nil slice", resp.Posts)
	}
}

func TestHandlePostsWithPosts(t *testing.T) {
	t.Parallel()

	f := &instagram.Feed{}
	f.Set([]*instagram.Post{
		{ID: "1", Alt: "first post", Permalink: "https://www.instagram.com/p/A/", Image: []byte("a"), ContentType: "image/jpeg", ETag: `"a"`},
		{ID: "2", Alt: "second post", Permalink: "https://www.instagram.com/p/B/", Image: []byte("b"), ContentType: "image/png", ETag: `"b"`},
	})
	a := newTestApp(f)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/posts", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	resp := decodePosts(t, rec.Body.Bytes())
	if len(resp.Posts) != 2 {
		t.Fatalf("posts = %#v, want 2", resp.Posts)
	}
	if resp.Posts[0] != (post{ID: "1", Alt: "first post", Permalink: "https://www.instagram.com/p/A/"}) {
		t.Errorf("posts[0] = %#v", resp.Posts[0])
	}
	if resp.Posts[1] != (post{ID: "2", Alt: "second post", Permalink: "https://www.instagram.com/p/B/"}) {
		t.Errorf("posts[1] = %#v", resp.Posts[1])
	}
}

func TestHandleImageUnknownID(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"not a valid id":         "bad.id",
		"well formed but absent": "17912345678901234",
	}
	for name, id := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			a := newTestApp(&instagram.Feed{})
			rec := httptest.NewRecorder()
			a.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/posts/"+id+"/image", nil))

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", rec.Code)
			}
			var env struct {
				Error struct{ Code string } `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("response is not the API error shape: %v: %s", err, rec.Body.Bytes())
			}
			if env.Error.Code != "not_found" {
				t.Errorf("error code = %q, want not_found", env.Error.Code)
			}
		})
	}
}

func TestHandleImageKnownID(t *testing.T) {
	t.Parallel()

	f := &instagram.Feed{}
	f.Set([]*instagram.Post{
		{ID: "1", Alt: "first post", Permalink: "https://www.instagram.com/p/A/", Image: []byte("image-bytes"), ContentType: "image/jpeg", ETag: `"etag-1"`},
	})
	a := newTestApp(f)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/posts/1/image", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != imageMaxAge {
		t.Errorf("Cache-Control = %q, want %q", cc, imageMaxAge)
	}
	if etag := rec.Header().Get("ETag"); etag != `"etag-1"` {
		t.Errorf("ETag = %q, want %q", etag, `"etag-1"`)
	}
	if rec.Body.String() != "image-bytes" {
		t.Errorf("body = %q, want image-bytes", rec.Body.String())
	}
}

// TestHandleImageConditionalRequest proves http.ServeContent's conditional
// handling works end to end through this handler: a matching If-None-Match
// gets a 304 with no body.
func TestHandleImageConditionalRequest(t *testing.T) {
	t.Parallel()

	f := &instagram.Feed{}
	f.Set([]*instagram.Post{
		{ID: "1", Alt: "first post", Permalink: "https://www.instagram.com/p/A/", Image: []byte("image-bytes"), ContentType: "image/jpeg", ETag: `"etag-1"`},
	})
	a := newTestApp(f)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/posts/1/image", nil)
	req.Header.Set("If-None-Match", `"etag-1"`)
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty on a 304", rec.Body.String())
	}
}

func TestHealthEndpointsReachableThroughTheBuiltHandler(t *testing.T) {
	t.Parallel()

	a := newTestApp(&instagram.Feed{})

	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/readyz before any sync: status = %d, want 503", rec.Code)
	}

	rec = httptest.NewRecorder()
	a.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("/healthz = %d, want 200 regardless of readiness", rec.Code)
	}

	a.ready.Store(true)
	rec = httptest.NewRecorder()
	a.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("/readyz once ready: status = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	a.handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("/healthz = %d, want 200 regardless of readiness", rec.Code)
	}
}
