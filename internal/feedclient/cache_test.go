package feedclient

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCacheEmptyByDefault(t *testing.T) {
	t.Parallel()

	var c Cache
	if posts := c.Posts(); len(posts) != 0 {
		t.Errorf("Posts() = %v, want none from the zero value", posts)
	}
	if _, ok := c.Post("1"); ok {
		t.Error("Post() found something in an empty cache")
	}
}

func TestCacheSetReplacesAndIndexes(t *testing.T) {
	t.Parallel()

	var c Cache
	c.Set([]*CachedPost{{ID: "1", Alt: "a"}, {ID: "2", Alt: "b"}})
	if got := c.Posts(); len(got) != 2 {
		t.Fatalf("Posts() = %v, want 2", got)
	}
	p, ok := c.Post("2")
	if !ok || p.Alt != "b" {
		t.Errorf("Post(2) = %+v, %v, want the second post", p, ok)
	}

	c.Set([]*CachedPost{{ID: "3", Alt: "c"}})
	if _, ok := c.Post("1"); ok {
		t.Error("Post(1) still found after Set replaced the posts")
	}
	if got := c.Posts(); len(got) != 1 || got[0].ID != "3" {
		t.Errorf("Posts() = %v, want only post 3", got)
	}
}

// fakeFeedService answers /api/v1/posts and /api/v1/posts/{id}/image in
// process, counting how often each is asked; no test in this file opens a
// real socket.
type fakeFeedService struct {
	mu         sync.Mutex
	posts      []Post
	failPosts  bool
	failImage  map[string]bool
	imageCalls map[string]int
	postsCalls int
}

func newFakeFeedService() *fakeFeedService {
	return &fakeFeedService{failImage: map[string]bool{}, imageCalls: map[string]int{}}
}

func (f *fakeFeedService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.URL.Path == "/api/v1/posts":
		f.postsCalls++
		if f.failPosts {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var b strings.Builder
		b.WriteString(`{"posts":[`)
		for i, p := range f.posts {
			if i > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, `{"id":%q,"alt":%q,"permalink":%q}`, p.ID, p.Alt, p.Permalink)
		}
		b.WriteString(`]}`)
		_, _ = w.Write([]byte(b.String()))
	case strings.HasPrefix(r.URL.Path, "/api/v1/posts/") && strings.HasSuffix(r.URL.Path, "/image"):
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/posts/"), "/image")
		f.imageCalls[id]++
		if f.failImage[id] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("ETag", `"`+id+`"`)
		_, _ = w.Write(jpegBytes)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeFeedService) imageCallCount(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.imageCalls[id]
}

func newSyncer(f *fakeFeedService, logs io.Writer) *Syncer {
	return &Syncer{
		Client: serve(f),
		Cache:  &Cache{},
		Logger: slog.New(slog.NewJSONHandler(logs, nil)),
	}
}

func TestSyncFetchesPostsAndImages(t *testing.T) {
	t.Parallel()

	f := newFakeFeedService()
	f.posts = []Post{
		{ID: "1", Alt: "Lichtpult", Permalink: "https://www.instagram.com/p/A/"},
		{ID: "2", Alt: "Bühne", Permalink: "https://www.instagram.com/p/B/"},
	}
	s := newSyncer(f, io.Discard)

	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	posts := s.Cache.Posts()
	if len(posts) != 2 {
		t.Fatalf("Posts() = %v, want 2", posts)
	}
	if posts[0].ID != "1" || posts[0].ContentType != "image/jpeg" || posts[0].ETag != `"1"` {
		t.Errorf("post 1 = %+v", posts[0])
	}

	// A second sync must reuse the images already cached, never asking
	// the feed service for an id it already has.
	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}
	if f.imageCallCount("1") != 1 || f.imageCallCount("2") != 1 {
		t.Errorf("image calls = %d/%d, want each fetched exactly once", f.imageCallCount("1"), f.imageCallCount("2"))
	}
}

func TestSyncEmptyStaysEmpty(t *testing.T) {
	t.Parallel()

	s := newSyncer(newFakeFeedService(), io.Discard)
	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("Sync() on an empty feed service: %v", err)
	}
	if len(s.Cache.Posts()) != 0 {
		t.Error("posts from nowhere")
	}
}

// TestSyncCapsPostCount proves the website bounds how many posts one cycle
// ever processes on its own, independent of what the feed service sends:
// internal/feedconfig already caps POST_COUNT to 10000, but a buggy or
// compromised feed service ignoring that ceiling must not be able to drive
// this process into downloading and holding images for far more posts than
// that, inside a single-replica, Recreate-strategy pod. The alt texts have
// the full 150 runes, so the list is also far larger than one MiB, the
// bound the website read the list with before it took every post.
func TestSyncCapsPostCount(t *testing.T) {
	t.Parallel()

	f := newFakeFeedService()
	const sent = maxPosts + 50
	alt := strings.Repeat("ä", 150)
	for i := range sent {
		f.posts = append(f.posts, Post{ID: fmt.Sprintf("%d", i), Alt: alt, Permalink: "https://www.instagram.com/p/A/"})
	}
	s := newSyncer(f, io.Discard)

	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	posts := s.Cache.Posts()
	if len(posts) != maxPosts {
		t.Fatalf("Posts() = %d, want capped at %d even though the feed service sent %d", len(posts), maxPosts, sent)
	}
	if posts[0].ID != "0" || posts[maxPosts-1].ID != fmt.Sprintf("%d", maxPosts-1) {
		t.Errorf("kept posts = %q..%q, want the first %d in the order the feed service sent them", posts[0].ID, posts[len(posts)-1].ID, maxPosts)
	}
}

func TestSyncKeepsLastCacheWhenPostsCallFails(t *testing.T) {
	t.Parallel()

	f := newFakeFeedService()
	f.posts = []Post{{ID: "1", Alt: "a", Permalink: "https://www.instagram.com/p/A/"}}
	s := newSyncer(f, io.Discard)
	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	f.failPosts = true
	if err := s.Sync(t.Context()); err == nil {
		t.Fatal("Sync() error = nil, want one for the failing posts call")
	}
	if got := s.Cache.Posts(); len(got) != 1 || got[0].ID != "1" {
		t.Errorf("Posts() = %v, want the previous cache kept", got)
	}
}

func TestSyncKeepsLastCacheWhenAllImagesFail(t *testing.T) {
	t.Parallel()

	f := newFakeFeedService()
	f.posts = []Post{{ID: "1", Alt: "a", Permalink: "https://www.instagram.com/p/A/"}}
	s := newSyncer(f, io.Discard)
	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	// A different post, whose picture can never be fetched: a feed reload
	// that cannot show anything new must not erase what is already shown.
	f.posts = []Post{{ID: "2", Alt: "b", Permalink: "https://www.instagram.com/p/B/"}}
	f.failImage["2"] = true
	if err := s.Sync(t.Context()); err == nil {
		t.Fatal("Sync() error = nil, want one when every image download failed")
	}
	if got := s.Cache.Posts(); len(got) != 1 || got[0].ID != "1" {
		t.Errorf("Posts() = %v, want the previous cache kept", got)
	}
}

// runScript runs Run with a recording wait and calls step after each wait,
// with the number of waits so far. Run stops when step returns false.
func runScript(t *testing.T, s *Syncer, interval time.Duration, step func(n int) bool) []time.Duration {
	t.Helper()
	var delays []time.Duration
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s.wait = func(_ context.Context, d time.Duration) bool {
		delays = append(delays, d)
		if !step(len(delays)) {
			cancel()
			return false
		}
		return true
	}
	s.Run(ctx, interval)
	return delays
}

func TestRunRetriesFailedSyncWithGrowingDelay(t *testing.T) {
	t.Parallel()

	f := newFakeFeedService()
	f.failPosts = true
	s := newSyncer(f, io.Discard)
	const interval = 30 * time.Minute

	got := runScript(t, s, interval, func(n int) bool { return n < 8 })
	want := []time.Duration{
		5 * time.Second, 10 * time.Second, 20 * time.Second, 40 * time.Second,
		80 * time.Second, 160 * time.Second, 5 * time.Minute, 5 * time.Minute, // 320s exceeds MaxRetryDelay
	}
	if len(got) != len(want) {
		t.Fatalf("delays = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("delay %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestRunCapsRetryAtInterval(t *testing.T) {
	t.Parallel()

	f := newFakeFeedService()
	f.failPosts = true
	s := newSyncer(f, io.Discard)

	got := runScript(t, s, 12*time.Second, func(n int) bool { return n < 4 })
	want := []time.Duration{5 * time.Second, 10 * time.Second, 12 * time.Second, 12 * time.Second}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("delay %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestRunSuccessResetsRetryDelayAndReturnsToInterval(t *testing.T) {
	t.Parallel()

	f := newFakeFeedService()
	f.posts = []Post{{ID: "1", Alt: "a", Permalink: "https://www.instagram.com/p/A/"}}
	f.failPosts = true
	s := newSyncer(f, io.Discard)
	const interval = time.Hour

	setFail := func(v bool) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.failPosts = v
	}
	got := runScript(t, s, interval, func(n int) bool {
		switch n {
		case 2:
			setFail(false) // the next attempt succeeds
		case 3:
			// success: the interval wait. Fail again afterwards.
			setFail(true)
		case 5:
			return false
		}
		return true
	})
	if len(got) != 5 {
		t.Fatalf("delays = %v, want 5", got)
	}
	if got[0] != 5*time.Second || got[1] != 10*time.Second {
		t.Errorf("failure delays = %v, want 5s then 10s", got[:2])
	}
	if got[2] < interval || got[2] > interval+interval/10 {
		t.Errorf("delay after success = %v, want the interval plus jitter, not a short retry", got[2])
	}
	if got[3] != 5*time.Second {
		t.Errorf("first delay after the success = %v, want it reset to 5s", got[3])
	}
	if got[4] != 10*time.Second {
		t.Errorf("second delay after the success = %v, want 10s", got[4])
	}
	if len(s.Cache.Posts()) != 1 {
		t.Errorf("cache not filled by the retry")
	}
}

func TestRunRetryDelayIsConfigurable(t *testing.T) {
	t.Parallel()

	f := newFakeFeedService()
	f.failPosts = true
	s := newSyncer(f, io.Discard)
	s.RetryDelay = 3 * time.Second

	got := runScript(t, s, time.Hour, func(n int) bool { return n < 2 })
	if got[0] != 3*time.Second || got[1] != 6*time.Second {
		t.Errorf("delays = %v, want 3s, 6s", got)
	}
}

func TestRunLogsOneLinePerFailedAttempt(t *testing.T) {
	t.Parallel()

	f := newFakeFeedService()
	f.failPosts = true
	var logs strings.Builder
	s := newSyncer(f, &logs)

	runScript(t, s, time.Hour, func(n int) bool { return n < 3 })
	if n := strings.Count(logs.String(), "feed sync failed"); n != 3 {
		t.Errorf("failure log lines = %d, want 3 (one per attempt)", n)
	}
}

func TestRunStopsWithItsContext(t *testing.T) {
	t.Parallel()

	s := newSyncer(newFakeFeedService(), io.Discard)
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
