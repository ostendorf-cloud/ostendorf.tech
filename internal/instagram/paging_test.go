package instagram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/secrettest"
)

// fakeToken is the access token every paging test uses. It is not a real
// token, and no failure message below quotes an error unredacted, so even a
// leak would not print it.
const fakeToken = "fake-instagram-token-0b7e4c"

// redact makes an error safe to print in a failure message.
func redact(err error) string {
	if err == nil {
		return "<nil>"
	}
	return strings.ReplaceAll(err.Error(), fakeToken, "[token]")
}

// pagedMeta is a fake media edge that pages its posts the way Meta does: an
// answer carries up to limit posts, paging.cursors.after and, while posts are
// left, paging.next. The cursor is "off<offset>", so the fake can serve the
// next page from it. The hooks rewrite single answers by request index.
type pagedMeta struct {
	mu       sync.Mutex
	posts    []Media
	requests []url.Values
	paths    []string
	// statusAt answers the request with that index with an error status.
	statusAt map[int]int
	// cursor rewrites the after cursor of the answer to request i.
	cursor func(i int, natural string) string
	// next rewrites paging.next of the answer to request i.
	next func(i int, natural string) string
}

func newPagedMeta(n int) *pagedMeta {
	m := &pagedMeta{statusAt: map[int]int{}}
	for i := 1; i <= n; i++ {
		m.posts = append(m.posts, Media{
			ID: strconv.Itoa(i), MediaType: "IMAGE",
			MediaURL:  "https://cdn.test/a.jpg",
			Permalink: fmt.Sprintf("https://www.instagram.com/p/%d/", i),
		})
	}
	return m
}

func (m *pagedMeta) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := len(m.requests)
	q := r.URL.Query()
	m.requests = append(m.requests, q)
	m.paths = append(m.paths, r.URL.Path)
	if q.Get("access_token") != fakeToken {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"message":"Invalid OAuth access token","type":"OAuthException","code":190}}`)
		return
	}
	if status, ok := m.statusAt[i]; ok {
		w.WriteHeader(status)
		fmt.Fprint(w, `{"error":{"message":"Application request limit reached","type":"OAuthException","code":4}}`)
		return
	}
	offset := 0
	if after := q.Get("after"); after != "" {
		offset, _ = strconv.Atoi(strings.TrimPrefix(after, "off"))
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset = min(max(offset, 0), len(m.posts))
	end := min(offset+limit, len(m.posts))

	natural := "off" + strconv.Itoa(end)
	cursor := natural
	if m.cursor != nil {
		cursor = m.cursor(i, natural)
	}
	paging := map[string]any{"cursors": map[string]string{"before": "off" + strconv.Itoa(offset), "after": cursor}}
	if end < len(m.posts) {
		next := "https://graph.instagram.com/v23.0/me/media?limit=" + strconv.Itoa(limit) +
			"&after=" + url.QueryEscape(cursor) + "&access_token=" + fakeToken
		if m.next != nil {
			next = m.next(i, next)
		}
		paging["next"] = next
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": m.posts[offset:end], "paging": paging})
}

func (m *pagedMeta) calls() []url.Values {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]url.Values(nil), m.requests...)
}

// countingHandler counts every request it gets; it stands for a host the
// client must never reach.
type countingHandler struct {
	mu sync.Mutex
	n  int
}

func (c *countingHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	w.WriteHeader(http.StatusTeapot)
}

func (c *countingHandler) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// hostsTransport serves each host from its own handler, in process. An
// unknown host answers 502.
type hostsTransport map[string]http.Handler

func (t hostsTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	if h, ok := t[r.URL.Host]; ok {
		h.ServeHTTP(rec, r)
	} else {
		rec.WriteHeader(http.StatusBadGateway)
	}
	return rec.Result(), nil
}

// pagedClient is a client against meta at graph.test, with no pause.
func pagedClient(meta http.Handler) *Client {
	c := NewClient("v23.0")
	c.BaseURL = "https://graph.test"
	c.HTTP = &http.Client{Transport: hostsTransport{"graph.test": meta}}
	c.AllowedImageHosts = []string{"cdn.test"}
	c.MaxImageBytes = 1000
	c.PagePause = 0
	return c
}

func ids(media []Media) string {
	out := make([]string, 0, len(media))
	for _, m := range media {
		out = append(out, m.ID)
	}
	return strings.Join(out, ",")
}

func TestMediaFollowsEveryPage(t *testing.T) {
	t.Parallel()

	meta := newPagedMeta(7)
	res, err := pagedClient(meta).Media(t.Context(), fakeToken, 3, 0)
	if err != nil {
		t.Fatalf("Media() error = %s", redact(err))
	}
	if got := ids(res.Media); got != "1,2,3,4,5,6,7" || res.Pages != 3 {
		t.Fatalf("Media() = %s in %d pages, want 1..7 in 3 pages", got, res.Pages)
	}
	calls := meta.calls()
	wantAfter := []string{"", "off3", "off6"}
	for i, q := range calls {
		if q.Get("after") != wantAfter[i] || q.Get("limit") != "3" || meta.paths[i] != "/v23.0/me/media" {
			t.Errorf("request %d: path %s, limit %s, after %q; want /v23.0/me/media, 3, %q",
				i, meta.paths[i], q.Get("limit"), q.Get("after"), wantAfter[i])
		}
	}
}

func TestMediaStopsAtTheCap(t *testing.T) {
	t.Parallel()

	meta := newPagedMeta(20)
	res, err := pagedClient(meta).Media(t.Context(), fakeToken, 3, 5)
	if err != nil {
		t.Fatalf("Media() error = %s", redact(err))
	}
	if got := ids(res.Media); got != "1,2,3,4,5" || res.Pages != 2 {
		t.Fatalf("Media() = %s in %d pages, want 1..5 in 2 pages", got, res.Pages)
	}
	// The last request asks only for what the cap leaves.
	if calls := meta.calls(); len(calls) != 2 || calls[1].Get("limit") != "2" {
		t.Errorf("requests = %v, want 2 with the second asking for 2", len(calls))
	}
}

func TestMediaStopsOnACursorLoop(t *testing.T) {
	t.Parallel()

	meta := newPagedMeta(30)
	// Every answer names the same cursor: a loop that would never end.
	meta.cursor = func(int, string) string { return "off3" }
	done := make(chan struct{})
	var res MediaResult
	var err error
	go func() {
		defer close(done)
		res, err = pagedClient(meta).Media(t.Context(), fakeToken, 3, 0)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Media() hangs on a cursor loop")
	}
	if !errors.Is(err, ErrCursorLoop) {
		t.Fatalf("Media() error = %s, want ErrCursorLoop", redact(err))
	}
	if n := len(meta.calls()); n != 2 {
		t.Errorf("requests = %d, want 2: the repeated cursor must stop the cycle", n)
	}
	if res.Pages != 2 {
		t.Errorf("Pages = %d, want 2", res.Pages)
	}
	secrettest.AssertHidden(t, err, fakeToken)
	secrettest.AssertHidden(t, err, "off3")
}

func TestMediaRefusesAnInvalidCursor(t *testing.T) {
	t.Parallel()

	for name, cursor := range map[string]string{
		"missing":        "",
		"bad characters": "off3&host=evil.test",
		"too long":       strings.Repeat("A", 513),
		"a url":          "https://evil.test/x",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			meta := newPagedMeta(10)
			meta.cursor = func(int, string) string { return cursor }
			_, err := pagedClient(meta).Media(t.Context(), fakeToken, 3, 0)
			if !errors.Is(err, ErrInvalidCursor) {
				t.Fatalf("Media() error = %s, want ErrInvalidCursor", redact(err))
			}
			if n := len(meta.calls()); n != 1 {
				t.Errorf("requests = %d, want 1", n)
			}
			secrettest.AssertHidden(t, err, fakeToken)
			if cursor != "" {
				secrettest.AssertHidden(t, err, cursor)
			}
		})
	}
}

func TestMediaPostsPerRequestRange(t *testing.T) {
	t.Parallel()

	for _, perRequest := range []int{-1, 0, MaxPostsPerRequest + 1, 1000} {
		meta := newPagedMeta(3)
		_, err := pagedClient(meta).Media(t.Context(), fakeToken, perRequest, 0)
		if err == nil {
			t.Errorf("Media(perRequest=%d) error = nil, want a refusal", perRequest)
		}
		if n := len(meta.calls()); n != 0 {
			t.Errorf("Media(perRequest=%d) sent %d requests, want 0", perRequest, n)
		}
	}
	for _, perRequest := range []int{1, MaxPostsPerRequest} {
		meta := newPagedMeta(3)
		if _, err := pagedClient(meta).Media(t.Context(), fakeToken, perRequest, 0); err != nil {
			t.Errorf("Media(perRequest=%d) error = %s", perRequest, redact(err))
		}
		if got := meta.calls()[0].Get("limit"); got != strconv.Itoa(perRequest) {
			t.Errorf("Media(perRequest=%d) asked for limit %s", perRequest, got)
		}
	}
	if _, err := pagedClient(newPagedMeta(3)).Media(t.Context(), fakeToken, 3, -1); err == nil {
		t.Error("Media(maxPosts=-1) error = nil, want a refusal")
	}
}

// TestMediaNeverFollowsPagingNext proves the next page is requested from
// the client's own base URL with the validated cursor, never from the URL
// in paging.next: a second server that paging.next points at must receive
// zero requests, whether the answer also carries a cursor or not.
func TestMediaNeverFollowsPagingNext(t *testing.T) {
	t.Parallel()

	for name, withCursor := range map[string]bool{"with a cursor": true, "without a cursor": false} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			evil := &countingHandler{}
			evilSrv := httptest.NewServer(evil)
			defer evilSrv.Close()

			meta := newPagedMeta(7)
			meta.next = func(_ int, natural string) string {
				u, _ := url.Parse(natural)
				return evilSrv.URL + u.Path + "?" + u.RawQuery
			}
			if !withCursor {
				meta.cursor = func(int, string) string { return "" }
			}
			graph := httptest.NewServer(meta)
			defer graph.Close()

			client := NewClient("v23.0")
			client.BaseURL = graph.URL
			client.PagePause = 0
			res, err := client.Media(t.Context(), fakeToken, 3, 0)

			if evil.count() != 0 {
				t.Fatalf("the host in paging.next received %d requests, want 0", evil.count())
			}
			if withCursor {
				if err != nil || ids(res.Media) != "1,2,3,4,5,6,7" {
					t.Fatalf("Media() = %s, %s; want 1..7 from the own base URL", ids(res.Media), redact(err))
				}
				return
			}
			if !errors.Is(err, ErrInvalidCursor) {
				t.Fatalf("Media() error = %s, want ErrInvalidCursor", redact(err))
			}
			secrettest.AssertHidden(t, err, fakeToken)
			secrettest.AssertHidden(t, err, evilSrv.URL)
		})
	}
}

func TestMediaEndsTheCycleOnA4xxOr429(t *testing.T) {
	t.Parallel()

	for status, kind := range map[int]error{
		http.StatusTooManyRequests: ErrRateLimited,
		http.StatusForbidden:       ErrRefused,
		http.StatusBadRequest:      ErrRefused,
	} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			t.Parallel()

			meta := newPagedMeta(10)
			meta.statusAt[1] = status
			res, err := pagedClient(meta).Media(t.Context(), fakeToken, 3, 0)
			if !errors.Is(err, kind) {
				t.Fatalf("Media() error = %s, want %v", redact(err), kind)
			}
			if n := len(meta.calls()); n != 2 {
				t.Errorf("requests = %d, want 2: the error ends the cycle", n)
			}
			// What the pages before the error collected is handed back.
			if res.Pages != 1 || ids(res.Media) != "1,2,3" {
				t.Errorf("partial result = %s in %d pages, want 1,2,3 in 1", ids(res.Media), res.Pages)
			}
			if !strings.Contains(err.Error(), "page 2") {
				t.Errorf("error = %s, want it to name page 2", redact(err))
			}
			secrettest.AssertHidden(t, err, fakeToken)
			secrettest.AssertHidden(t, err, "off3")
		})
	}
}

func TestMediaPausesBetweenPages(t *testing.T) {
	t.Parallel()

	const pause = 30 * time.Millisecond
	client := pagedClient(newPagedMeta(7))
	client.PagePause = pause
	start := time.Now()
	res, err := client.Media(t.Context(), fakeToken, 3, 0)
	if err != nil {
		t.Fatalf("Media() error = %s", redact(err))
	}
	if elapsed := time.Since(start); elapsed < 2*pause {
		t.Errorf("3 pages took %v, want at least two pauses of %v", elapsed, pause)
	}
	if res.Pages != 3 {
		t.Errorf("Pages = %d, want 3", res.Pages)
	}

	// The pause ends with the context.
	client.PagePause = time.Hour
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	res, err = client.Media(ctx, fakeToken, 3, 0)
	if !errors.Is(err, context.DeadlineExceeded) || res.Pages != 1 {
		t.Errorf("Media() with a short context = %d pages, %s; want 1 and the deadline", res.Pages, redact(err))
	}
}

func TestNewClientPausesBetweenPages(t *testing.T) {
	t.Parallel()

	if got := NewClient("v23.0").PagePause; got != DefaultPagePause || got <= 0 {
		t.Errorf("NewClient().PagePause = %v, want DefaultPagePause > 0", got)
	}
}

// pagedSyncer is a Syncer against meta at graph.test and an image CDN at
// cdn.test, logging JSON into logs.
func pagedSyncer(meta *pagedMeta, logs *bytes.Buffer) *Syncer {
	cdn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a.jpg" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpegBytes)
	})
	client := pagedClient(meta)
	client.HTTP = &http.Client{Transport: hostsTransport{"graph.test": meta, "cdn.test": cdn}}
	return &Syncer{
		Client: client, Tokens: &fakeTokenSource{token: fakeToken},
		Count: 0, PerRequest: 3, Feed: &Feed{},
		Logger: slog.New(slog.NewJSONHandler(logs, nil)),
	}
}

// assertLogsClean fails when the token or a cursor appears in the logs. The
// failure message names only what leaked, never the log text.
func assertLogsClean(t *testing.T, logs string, cursors ...string) {
	t.Helper()
	if strings.Contains(logs, fakeToken) {
		t.Error("a log line contains the access token")
	}
	for _, c := range cursors {
		if strings.Contains(logs, `"`+c+`"`) || strings.Contains(logs, "after="+c) {
			t.Errorf("a log line contains the cursor %s", c)
		}
	}
}

func TestSyncCollectsEveryPageAndLogsTheCycle(t *testing.T) {
	t.Parallel()

	meta := newPagedMeta(7)
	meta.posts[4].MediaURL = "https://cdn.test/missing.jpg"
	var logs bytes.Buffer
	s := pagedSyncer(meta, &logs)
	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("Sync() error = %s", redact(err))
	}
	if n := len(s.Feed.Posts()); n != 6 {
		t.Fatalf("feed has %d posts, want 6 (7 minus the one failed picture)", n)
	}

	var line map[string]any
	for _, raw := range bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte("\n")) {
		var entry map[string]any
		if json.Unmarshal(raw, &entry) == nil && entry["msg"] == "instagram feed updated" {
			line = entry
		}
	}
	if line == nil {
		t.Fatal("no log line for the completed cycle")
	}
	if line["posts"] != float64(6) || line["pages"] != float64(3) || line["failed_images"] != float64(1) {
		t.Errorf("cycle log posts=%v pages=%v failed_images=%v, want 6, 3, 1", line["posts"], line["pages"], line["failed_images"])
	}
	if d, ok := line["duration"].(string); !ok || d == "" {
		t.Errorf("cycle log duration = %v, want a duration", line["duration"])
	}
	assertLogsClean(t, logs.String(), "off3", "off6")
}

// TestSyncKeepsTheLastFeedWhenAPageFails proves a cycle that fails on page
// 2 keeps the whole previous feed, not the first page of the new one.
func TestSyncKeepsTheLastFeedWhenAPageFails(t *testing.T) {
	t.Parallel()

	meta := newPagedMeta(4)
	var logs bytes.Buffer
	s := pagedSyncer(meta, &logs)
	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("first Sync() error = %s", redact(err))
	}

	for status, kind := range map[int]error{
		http.StatusInternalServerError: nil,
		http.StatusTooManyRequests:     ErrRateLimited,
	} {
		meta.mu.Lock()
		meta.posts = newPagedMeta(9).posts
		meta.statusAt = map[int]int{len(meta.requests) + 1: status}
		meta.mu.Unlock()

		err := s.Sync(t.Context())
		if err == nil {
			t.Fatalf("status %d on page 2: Sync() error = nil", status)
		}
		if kind != nil && !errors.Is(err, kind) {
			t.Errorf("status %d: error = %s, want %v", status, redact(err), kind)
		}
		if n := len(s.Feed.Posts()); n != 4 {
			t.Errorf("status %d on page 2: feed has %d posts, want the last good 4", status, n)
		}
		secrettest.AssertHidden(t, err, fakeToken)
	}
	assertLogsClean(t, logs.String(), "off3", "off6")
}

// TestSyncPassesPerRequestAndCap proves Syncer hands its own settings to the
// client, and that zero PerRequest means the default.
func TestSyncPassesPerRequestAndCap(t *testing.T) {
	t.Parallel()

	meta := newPagedMeta(300)
	var logs bytes.Buffer
	s := pagedSyncer(meta, &logs)
	s.PerRequest = 0
	s.Count = 120
	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("Sync() error = %s", redact(err))
	}
	calls := meta.calls()
	if len(calls) != 3 || calls[0].Get("limit") != strconv.Itoa(DefaultPostsPerRequest) || calls[2].Get("limit") != "20" {
		t.Errorf("requests = %d, want 3 asking for 50, 50, 20", len(calls))
	}
	if n := len(s.Feed.Posts()); n != 120 {
		t.Errorf("feed has %d posts, want the cap of 120", n)
	}
}
