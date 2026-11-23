package instagram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// hostTransport serves every host in process from one handler, so no test
// here ever opens a socket: the Graph API and the CDN are both this
// handler.
type hostTransport struct{ h http.Handler }

func (t hostTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	t.h.ServeHTTP(rec, r)
	return rec.Result(), nil
}

// Minimal valid image headers, enough for http.DetectContentType.
var (
	jpegBytes = append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{1}, 100)...)
	pngBytes  = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{2}, 100)...)
)

// testNow is the fixed clock every test reads, so nothing depends on the
// wall clock. It is never written, which is what keeps the tests below
// safe to run in parallel.
var testNow = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

// fakeMeta is the Graph API plus the CDN.
type fakeMeta struct {
	mu            sync.Mutex
	validTokens   map[string]bool
	media         []Media
	mediaCalls    int
	refreshCalls  int
	imageCalls    map[string]int
	lastMediaPath string
	lastLimit     string
	lastToken     string
	failMedia     bool
}

func newFakeMeta() *fakeMeta {
	return &fakeMeta{validTokens: map[string]bool{"tok-1": true}, imageCalls: map[string]int{}}
}

func (f *fakeMeta) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.Host == "graph.test" && strings.HasSuffix(r.URL.Path, "/me/media"):
		f.mediaCalls++
		f.lastMediaPath, f.lastLimit = r.URL.Path, r.URL.Query().Get("limit")
		f.lastToken = r.URL.Query().Get("access_token")
		if f.failMedia || !f.validTokens[r.URL.Query().Get("access_token")] {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"message":"Invalid OAuth access token","type":"OAuthException","code":190}}`)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": f.media})
	case r.Host == "graph.test" && r.URL.Path == "/refresh_access_token":
		f.refreshCalls++
		if r.URL.Query().Get("grant_type") != "ig_refresh_token" || !f.validTokens[r.URL.Query().Get("access_token")] {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next := fmt.Sprintf("tok-%d", len(f.validTokens)+1)
		f.validTokens[next] = true
		fmt.Fprintf(w, `{"access_token":%q,"token_type":"bearer","expires_in":5184000}`, next)
	case r.Host == "cdn.test":
		f.imageCalls[r.URL.Path]++
		switch r.URL.Path {
		case "/a.jpg", "/v.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(jpegBytes)
		case "/c.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pngBytes)
		case "/html":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("<html><script>alert(1)</script></html>"))
		case "/text":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write(jpegBytes)
		case "/huge":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(append(jpegBytes, make([]byte, 2000)...))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeMeta) token() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastToken
}

func (f *fakeMeta) refreshes() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refreshCalls
}

// newSyncer builds a Syncer whose every dependency is in process: the fake
// Meta behind a transport, and a TokenSource answering with the token the
// environment would have supplied. Token refresh itself is TokenStore's
// job now (token.go, token_test.go), not Syncer's, so this fixed source is
// all Syncer ever needs from one.
func newSyncer(meta *fakeMeta, logs io.Writer) *Syncer {
	client := NewClient("v23.0")
	client.BaseURL = "https://graph.test"
	client.HTTP = &http.Client{Transport: hostTransport{meta}}
	client.MaxImageBytes = 1000
	client.AllowedImageHosts = []string{"cdn.test"}
	client.PagePause = 0
	return &Syncer{
		Client: client, Tokens: &fakeTokenSource{token: "tok-1"},
		Count: 20, Feed: &Feed{},
		Logger: slog.New(slog.NewJSONHandler(logs, nil)),
	}
}

func TestSyncFeed(t *testing.T) {
	t.Parallel()

	meta := newFakeMeta()
	meta.media = []Media{
		{ID: "1", Caption: "#foh für die @fhaachenbigband\n in der Aula", MediaType: "IMAGE", MediaURL: "https://cdn.test/a.jpg", Permalink: "https://www.instagram.com/p/A/"},
		{ID: "2", MediaType: "VIDEO", MediaURL: "https://cdn.test/video.mp4", ThumbnailURL: "https://cdn.test/v.jpg", Permalink: "https://www.instagram.com/reel/B/", Timestamp: "2026-06-01T18:30:00+0000"},
		{ID: "3", Caption: strings.Repeat("Weltfest ", 40), MediaType: "CAROUSEL_ALBUM", MediaURL: "https://cdn.test/c.png", Permalink: "https://www.instagram.com/p/C/"},
		{ID: "4", MediaType: "IMAGE", MediaURL: "https://cdn.test/html", Permalink: "https://www.instagram.com/p/D/"},
		{ID: "5", MediaType: "IMAGE", MediaURL: "https://cdn.test/text", Permalink: "https://www.instagram.com/p/E/"},
		{ID: "6", MediaType: "IMAGE", MediaURL: "https://cdn.test/huge", Permalink: "https://www.instagram.com/p/F/"},
		{ID: "7", MediaType: "IMAGE", MediaURL: "http://cdn.test/a.jpg", Permalink: "https://www.instagram.com/p/G/"},
		{ID: "../8", MediaType: "IMAGE", MediaURL: "https://cdn.test/a.jpg", Permalink: "https://www.instagram.com/p/H/"},
		{ID: "9", MediaType: "IMAGE", MediaURL: "https://cdn.test/a.jpg", Permalink: "javascript:alert(1)"},
	}
	var logs bytes.Buffer
	s := newSyncer(meta, &logs)
	if err := s.Sync(t.Context()); err != nil {
		t.Fatal(err)
	}
	if meta.lastMediaPath != "/v23.0/me/media" || meta.lastLimit != "20" {
		t.Errorf("media request %s limit %s", meta.lastMediaPath, meta.lastLimit)
	}
	posts := s.Feed.Posts()
	var ids []string
	for _, p := range posts {
		ids = append(ids, p.ID)
	}
	if strings.Join(ids, ",") != "1,2,3" {
		t.Fatalf("posts = %v, want 1,2,3", ids)
	}
	if posts[0].Alt != "#foh für die @fhaachenbigband in der Aula" || posts[0].ContentType != "image/jpeg" {
		t.Errorf("post 1 = %q %q", posts[0].Alt, posts[0].ContentType)
	}
	if posts[1].Alt != "Instagram-Beitrag vom 01.06.2026" || meta.imageCalls["/v.jpg"] != 1 || meta.imageCalls["/video.mp4"] != 0 {
		t.Errorf("video: alt %q, calls %v", posts[1].Alt, meta.imageCalls)
	}
	if n := len([]rune(posts[2].Alt)); n > maxAltRunes || !strings.HasSuffix(posts[2].Alt, "…") || posts[2].ContentType != "image/png" {
		t.Errorf("carousel: alt of %d runes %q, type %q", n, posts[2].Alt, posts[2].ContentType)
	}
	for _, secret := range []string{"tok-1", "tok-2"} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("log contains the access token %s", secret)
		}
	}

	// A second sync reuses the images in memory.
	if err := s.Sync(t.Context()); err != nil {
		t.Fatal(err)
	}
	if meta.imageCalls["/a.jpg"] != 1 {
		t.Errorf("image downloaded %d times, want 1", meta.imageCalls["/a.jpg"])
	}
	if p, ok := s.Feed.Post("1"); !ok || !bytes.Equal(p.Image, jpegBytes) || p.ETag == "" {
		t.Error("post 1 not served from memory")
	}
}

func TestKeepLastFeedOnError(t *testing.T) {
	t.Parallel()

	meta := newFakeMeta()
	meta.media = []Media{{ID: "1", MediaType: "IMAGE", MediaURL: "https://cdn.test/a.jpg", Permalink: "https://www.instagram.com/p/A/"}}
	var logs bytes.Buffer
	s := newSyncer(meta, &logs)
	if err := s.Sync(t.Context()); err != nil {
		t.Fatal(err)
	}

	meta.failMedia = true
	err := s.Sync(t.Context())
	if err == nil || !strings.Contains(err.Error(), "Invalid OAuth access token") {
		t.Fatalf("err = %v, want the API message", err)
	}
	if strings.Contains(err.Error(), "tok-1") || strings.Contains(err.Error(), "tok-2") {
		t.Error("error contains the access token")
	}
	if len(s.Feed.Posts()) != 1 {
		t.Error("feed dropped after an API error")
	}

	// Every image failing keeps the old feed too.
	meta.failMedia = false
	meta.media = []Media{{ID: "2", MediaType: "IMAGE", MediaURL: "https://cdn.test/missing.jpg", Permalink: "https://www.instagram.com/p/B/"}}
	if err := s.Sync(t.Context()); err == nil {
		t.Fatal("sync with only failed images succeeded")
	}
	if p := s.Feed.Posts(); len(p) != 1 || p[0].ID != "1" {
		t.Error("feed replaced by nothing")
	}

	// An empty answer, or one with nothing usable, keeps the old feed as well.
	for name, media := range map[string][]Media{
		"empty data":       nil,
		"all filtered out": {{ID: "../x", MediaType: "IMAGE", MediaURL: "https://cdn.test/a.jpg", Permalink: "https://www.instagram.com/p/X/"}},
	} {
		meta.media = media
		if err := s.Sync(t.Context()); err == nil {
			t.Errorf("%s: sync succeeded", name)
		}
		if p := s.Feed.Posts(); len(p) != 1 || p[0].ID != "1" {
			t.Errorf("%s: feed replaced by nothing", name)
		}
	}
}

func TestEmptyFeedStaysEmpty(t *testing.T) {
	t.Parallel()

	// Without posts to protect, an empty answer is simply the current state.
	s := newSyncer(newFakeMeta(), io.Discard)
	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("empty account: %v", err)
	}
	if len(s.Feed.Posts()) != 0 {
		t.Error("posts from nowhere")
	}
}

// TestSyncNoTokenSourceConfiguredFails is the belt to cmd/feed's own
// braces: Syncer refuses to call Meta at all when it was built without a
// TokenSource, rather than panicking on a nil Tokens. Token refresh
// itself (age bounds, failure tolerance, CAS handling) is TokenStore's
// own contract now; see token_test.go.
func TestSyncNoTokenSourceConfiguredFails(t *testing.T) {
	t.Parallel()

	s := newSyncer(newFakeMeta(), io.Discard)
	s.Tokens = nil
	err := s.Sync(t.Context())
	if err == nil || !strings.Contains(err.Error(), "no token source") {
		t.Errorf("err = %v, want one naming the missing token source", err)
	}
}

func TestUnversionedClient(t *testing.T) {
	t.Parallel()

	meta := newFakeMeta()
	s := newSyncer(meta, io.Discard)
	s.Client.Version = ""
	if _, err := s.Client.Media(t.Context(), "tok-1", 5, 0); err != nil {
		t.Fatal(err)
	}
	if meta.lastMediaPath != "/me/media" || meta.lastLimit != "5" {
		t.Errorf("path %s limit %s", meta.lastMediaPath, meta.lastLimit)
	}
}

func TestRunStopsWithItsContext(t *testing.T) {
	t.Parallel()

	s := newSyncer(newFakeMeta(), io.Discard)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		s.Run(ctx, time.Hour)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after its context was cancelled")
	}
}

func TestValidID(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"17912345678901234":     true,
		"abc_DEF":               true,
		"":                      false,
		"../x":                  false,
		"1 2":                   false,
		strings.Repeat("1", 65): false,
	}
	for id, want := range tests {
		t.Run(id, func(t *testing.T) {
			t.Parallel()

			if got := ValidID(id); got != want {
				t.Errorf("ValidID(%q) = %v, want %v", id, got, want)
			}
		})
	}
}
