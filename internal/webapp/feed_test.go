package webapp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/feedclient"
)

// jpeg is a picture only in the sense that matters here: the bytes the
// cache holds and the handler hands back unchanged.
var jpeg = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x01, 0x02, 0x03}

func decodeFeed(t *testing.T, body []byte) instagramFeedResponse {
	t.Helper()
	var feed instagramFeedResponse
	if err := json.Unmarshal(body, &feed); err != nil {
		t.Fatalf("decode feed response: %v", err)
	}
	return feed
}

func TestInstagramFeedIsEmptyWithoutAFeedServiceConfigured(t *testing.T) {
	// No FEED_SERVICE_URL: no syncer runs, so the cache never fills and
	// the endpoint answers an empty list rather than an error. The
	// frontend renders no Impressionen section at all for this answer.
	app := testApp(t)
	if app.feedSyncer != nil {
		t.Error("a syncer was built without a configured feed service")
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/instagram", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	feed := decodeFeed(t, rec.Body.Bytes())
	if len(feed.Posts) != 0 {
		t.Errorf("posts = %+v, want none", feed.Posts)
	}
	// An absent list and an empty one are different answers to a client;
	// this one is always a list, so the frontend never has to special case
	// a missing field.
	if body := rec.Body.String(); body != `{"posts":[]}` {
		t.Errorf("body = %s, want an empty list", body)
	}
}

func TestInstagramFeedListsPosts(t *testing.T) {
	app := testApp(t)
	app.feedCache.Set([]*feedclient.CachedPost{
		{ID: "1", Permalink: "https://www.instagram.com/p/A/", Alt: "Lichtpult im Saal", Image: jpeg, ContentType: "image/jpeg", ETag: `"abc"`},
		{ID: "2", Permalink: "https://www.instagram.com/p/B/", Alt: "Bühne von hinten", Image: jpeg, ContentType: "image/jpeg", ETag: `"def"`},
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/instagram", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	feed := decodeFeed(t, rec.Body.Bytes())
	if len(feed.Posts) != 2 {
		t.Fatalf("posts = %+v, want 2", feed.Posts)
	}
	want := instagramPost{ID: "1", Alt: "Lichtpult im Saal", Permalink: "https://www.instagram.com/p/A/"}
	if feed.Posts[0] != want {
		t.Errorf("first post = %+v, want %+v", feed.Posts[0], want)
	}

	// One post carries exactly id, alt and permalink. The picture itself
	// is fetched per post from this service's own image endpoint, so the
	// bytes never travel in this response, and nothing links to the feed
	// service or Meta.
	var raw struct {
		Posts []map[string]any `json:"posts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw feed: %v", err)
	}
	for i, post := range raw.Posts {
		if len(post) != 3 {
			t.Errorf("post %d carries %v, want exactly id, alt and permalink", i, post)
		}
	}
}

func TestInstagramImageServesTheCachedPicture(t *testing.T) {
	app := testApp(t)
	app.feedCache.Set([]*feedclient.CachedPost{
		{ID: "1", Permalink: "https://www.instagram.com/p/A/", Alt: "Lichtpult im Saal", Image: jpeg, ContentType: "image/jpeg", ETag: `"abc"`},
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/instagram/1", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.Bytes(); string(got) != string(jpeg) {
		t.Errorf("body = %v, want the stored picture", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Errorf("Content-Type = %q, want image/jpeg", got)
	}
	if got := rec.Header().Get("ETag"); got != `"abc"` {
		t.Errorf("ETag = %q, want the post's own", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != imageMaxAge {
		t.Errorf("Cache-Control = %q, want %q", got, imageMaxAge)
	}
}

func TestInstagramImageNotFound(t *testing.T) {
	app := testApp(t)
	app.feedCache.Set([]*feedclient.CachedPost{
		{ID: "1", Permalink: "https://www.instagram.com/p/A/", Image: jpeg, ContentType: "image/jpeg", ETag: `"abc"`},
	})

	tests := map[string]string{
		"unknown id":          "/api/v1/instagram/404",
		"id that is no id":    "/api/v1/instagram/not%20an%20id",
		"traversal attempt":   "/api/v1/instagram/..%2f..%2fetc%2fpasswd",
		"empty id":            "/api/v1/instagram/",
		"id with a semicolon": "/api/v1/instagram/1;2",
	}

	for name, path := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			app.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}

func TestFeedSyncerBuiltWhenFeedServiceConfigured(t *testing.T) {
	cfg := testConfig(t, map[string]string{
		"FEED_SERVICE_URL": "http://feed.ostendorf-tech.svc.cluster.local",
	})
	dist, prerendered := testFS()
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if app.feedSyncer == nil {
		t.Fatal("no syncer built for a configured feed service")
	}
	if app.feedSyncer.Cache != app.feedCache {
		t.Error("the syncer fills a different cache than the handlers read")
	}
	// Start is deliberately not called here: this syncer points at the
	// configured feed service, and no test in this repository talks to
	// the network. What Start does with a syncer is covered by
	// internal/feedclient's own Run test.
}

func TestStartWithoutFeedRunsNothing(t *testing.T) {
	// The default deployment configures no feed service, and Start has to
	// be callable anyway: main calls it unconditionally.
	app := testApp(t)
	if app.feedSyncer != nil {
		t.Fatal("a background worker was built without configuration for it")
	}
	app.Start(t.Context())
}
