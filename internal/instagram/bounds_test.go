package instagram

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/secrettest"
)

// TestMediaZeroMeansTheMaximum proves a cap of 0 stops at MaxPosts and does
// not page on for as long as the account has posts.
func TestMediaZeroMeansTheMaximum(t *testing.T) {
	t.Parallel()

	meta := newPagedMeta(MaxPosts + 250)
	res, err := pagedClient(meta).Media(t.Context(), fakeToken, MaxPostsPerRequest, 0)
	if err != nil {
		t.Fatalf("Media() error = %s", redact(err))
	}
	if len(res.Media) != MaxPosts || res.Pages != MaxPosts/MaxPostsPerRequest {
		t.Errorf("Media() = %d posts in %d pages, want %d in %d", len(res.Media), res.Pages, MaxPosts, MaxPosts/MaxPostsPerRequest)
	}
}

// TestMediaZeroMeansTheMaximumAtOnePostPerRequest proves the smallest page
// size reaches the maximum of posts within the page bound: 10000 posts at
// one per request take 10000 pages and end at the cap, not at
// ErrTooManyPages.
func TestMediaZeroMeansTheMaximumAtOnePostPerRequest(t *testing.T) {
	t.Parallel()

	meta := newPagedMeta(MaxPosts + 5)
	res, err := pagedClient(meta).Media(t.Context(), fakeToken, 1, 0)
	if err != nil {
		t.Fatalf("Media() error = %s", redact(err))
	}
	if len(res.Media) != MaxPosts || res.Pages != MaxPosts {
		t.Errorf("Media() = %d posts in %d pages, want %d in %d", len(res.Media), res.Pages, MaxPosts, MaxPosts)
	}
}

// TestMediaCapAboveTheMaximumIsBounded proves a cap above MaxPosts is held
// to MaxPosts as well.
func TestMediaCapAboveTheMaximumIsBounded(t *testing.T) {
	t.Parallel()

	meta := newPagedMeta(MaxPosts + 250)
	res, err := pagedClient(meta).Media(t.Context(), fakeToken, MaxPostsPerRequest, MaxPosts+200)
	if err != nil {
		t.Fatalf("Media() error = %s", redact(err))
	}
	if len(res.Media) != MaxPosts {
		t.Errorf("Media() = %d posts, want %d", len(res.Media), MaxPosts)
	}
}

// TestMediaStopsOnATwoCursorLoop proves a loop over two cursors, A, B, A,
// is found as well as a cursor repeated at once.
func TestMediaStopsOnATwoCursorLoop(t *testing.T) {
	t.Parallel()

	meta := newPagedMeta(100)
	meta.cursor = func(i int, _ string) string {
		if i%2 == 0 {
			return "cursorA"
		}
		return "cursorB"
	}
	res, err := pagedClient(meta).Media(t.Context(), fakeToken, 3, 0)
	if !errors.Is(err, ErrCursorLoop) {
		t.Fatalf("Media() error = %s, want ErrCursorLoop", redact(err))
	}
	calls := meta.calls()
	if len(calls) != 3 || res.Pages != 3 {
		t.Fatalf("requests = %d, pages = %d; want 3 and 3: A, B, then A again", len(calls), res.Pages)
	}
	if calls[1].Get("after") != "cursorA" || calls[2].Get("after") != "cursorB" {
		t.Errorf("after = %q, %q; want cursorA, cursorB", calls[1].Get("after"), calls[2].Get("after"))
	}
	secrettest.AssertHidden(t, err, "cursorA")
	secrettest.AssertHidden(t, err, fakeToken)
}

// TestMediaStopsAtThePageBound proves a cycle that keeps getting pages ends
// with ErrTooManyPages once it reaches the bound.
func TestMediaStopsAtThePageBound(t *testing.T) {
	t.Parallel()

	meta := newPagedMeta(100)
	client := pagedClient(meta)
	client.maxPages = 3
	res, err := client.Media(t.Context(), fakeToken, 2, 0)
	if !errors.Is(err, ErrTooManyPages) {
		t.Fatalf("Media() error = %s, want ErrTooManyPages", redact(err))
	}
	if n := len(meta.calls()); n != 3 || res.Pages != 3 {
		t.Errorf("requests = %d, pages = %d; want 3 and 3", n, res.Pages)
	}
	secrettest.AssertHidden(t, err, fakeToken)
}

// TestDefaultPageBoundCoversTheMaximum proves the arithmetic: the page
// bound is at least the pages MaxPosts take at the smallest page size.
func TestDefaultPageBoundCoversTheMaximum(t *testing.T) {
	t.Parallel()

	if defaultMaxPages < MaxPosts {
		t.Errorf("defaultMaxPages = %d, want at least %d: MaxPosts at one post per request", defaultMaxPages, MaxPosts)
	}
}

// redirectingServer answers every request with a 302 to target, carrying the
// query of the request along.
func redirectingServer(t *testing.T, tls bool, target func() string) *httptest.Server {
	t.Helper()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target()+r.URL.Path+"?"+r.URL.RawQuery, http.StatusFound)
	})
	var srv *httptest.Server
	if tls {
		srv = httptest.NewTLSServer(h)
	} else {
		srv = httptest.NewServer(h)
	}
	t.Cleanup(srv.Close)
	return srv
}

// TestGraphRedirectIsNotFollowed proves a 3xx from the Graph API ends the
// call with ErrRedirect and the second server it names receives no request,
// even with an http.Client that would follow redirects on its own.
func TestGraphRedirectIsNotFollowed(t *testing.T) {
	t.Parallel()

	second := &countingHandler{}
	secondSrv := httptest.NewServer(second)
	t.Cleanup(secondSrv.Close)
	graph := redirectingServer(t, false, func() string { return secondSrv.URL })

	for name, hc := range map[string]*http.Client{
		"NewClient's own": NewClient("v23.0").HTTP,
		"a plain client":  {},
	} {
		client := NewClient("v23.0")
		client.BaseURL = graph.URL
		client.HTTP = hc
		client.PagePause = 0

		_, err := client.Media(t.Context(), fakeToken, 3, 0)
		if !errors.Is(err, ErrRedirect) {
			t.Errorf("%s: Media() error = %s, want ErrRedirect", name, redact(err))
		}
		secrettest.AssertHidden(t, err, fakeToken)
		secrettest.AssertHidden(t, err, secondSrv.URL)

		_, err = client.RefreshToken(t.Context(), fakeToken)
		if !errors.Is(err, ErrRedirect) {
			t.Errorf("%s: RefreshToken() error = %s, want ErrRedirect", name, redact(err))
		}
		secrettest.AssertHidden(t, err, fakeToken)
	}
	if n := second.count(); n != 0 {
		t.Errorf("the redirect target received %d requests, want 0", n)
	}
}

// TestImageRedirectIsNotFollowed proves a 3xx from an allowed picture host
// is not followed: the host allow list is checked on the one URL that is
// requested, and the second server receives no request.
func TestImageRedirectIsNotFollowed(t *testing.T) {
	t.Parallel()

	second := &countingHandler{}
	secondSrv := httptest.NewTLSServer(second)
	t.Cleanup(secondSrv.Close)
	cdn := redirectingServer(t, true, func() string { return secondSrv.URL })

	client := NewClient("v23.0")
	// Both TLS test servers share one certificate, so this client trusts
	// the second as well and would reach it if it followed the redirect.
	client.HTTP = cdn.Client()
	client.AllowedImageHosts = []string{"127.0.0.1"}

	_, _, err := client.Image(t.Context(), cdn.URL+"/a.jpg")
	if !errors.Is(err, ErrRedirect) {
		t.Fatalf("Image() error = %v, want ErrRedirect", err)
	}
	if n := second.count(); n != 0 {
		t.Errorf("the redirect target received %d requests, want 0", n)
	}
	secrettest.AssertHidden(t, err, strconv.Quote(secondSrv.URL))
}
