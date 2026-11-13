package feed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ostendorf-cloud/govue/health"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/instagram"
)

// hostTransport serves every host in process from one handler, so no test
// here ever opens a socket, matching internal/instagram/instagram_test.go's
// own approach.
type hostTransport struct{ h http.Handler }

func (t hostTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	t.h.ServeHTTP(rec, r)
	return rec.Result(), nil
}

// jpegBytes are minimal valid JPEG headers, enough for http.DetectContentType.
var jpegBytes = append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{1}, 100)...)

// fakeMeta is a minimal in-process fake of the Graph API's me/media
// endpoint plus its CDN, just enough for Syncer.Sync to succeed.
type fakeMeta struct {
	mediaCalls int
}

func (f *fakeMeta) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Host == "graph.test" && strings.HasSuffix(r.URL.Path, "/me/media"):
		f.mediaCalls++
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []instagram.Media{
			{ID: "1", MediaType: "IMAGE", MediaURL: "https://cdn.test/a.jpg", Permalink: "https://www.instagram.com/p/A/"},
		}})
	case r.Host == "cdn.test" && r.URL.Path == "/a.jpg":
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpegBytes)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// fakeTokenSource is a hand-written instagram.TokenSource: it returns
// token or err, and counts how often it was asked.
type fakeTokenSource struct {
	token string
	err   error
	calls int
}

func (f *fakeTokenSource) Token(context.Context) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.token, nil
}

// newTestAppWithSyncer builds an App whose syncer talks to an in-process
// fake Meta through a fake TokenSource, with no OpenBao or real network
// involved anywhere.
func newTestAppWithSyncer(meta *fakeMeta, tokens *fakeTokenSource, interval time.Duration) *App {
	client := instagram.NewClient("v23.0")
	client.BaseURL = "https://graph.test"
	client.HTTP = &http.Client{Transport: hostTransport{meta}}
	client.MaxImageBytes = 1000
	client.AllowedImageHosts = []string{"cdn.test"}

	f := &instagram.Feed{}
	syncer := &instagram.Syncer{
		Client: client,
		Tokens: tokens,
		Count:  20,
		Feed:   f,
		Logger: discardLogger(),
		Now:    time.Now,
	}

	a := &App{logger: discardLogger(), feed: f, syncer: syncer, interval: interval, health: health.New()}
	a.health.Add("feed", readinessCheckTimeout, a.checkReady)
	a.handler = a.buildHandler()
	return a
}

// TestRunFlipsReadyOnSuccessfulCycle exercises run itself, not Sync
// directly: a context that outlives one cycle but not the long interval
// that follows it proves run flips ready to true before it goes back to
// sleep.
func TestRunFlipsReadyOnSuccessfulCycle(t *testing.T) {
	t.Parallel()

	a := newTestAppWithSyncer(&fakeMeta{}, &fakeTokenSource{token: "tok-1"}, time.Hour)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	a.run(ctx)

	if !a.ready.Load() {
		t.Error("ready = false after a successful cycle, want true")
	}
	if len(a.feed.Posts()) != 1 {
		t.Errorf("posts = %d, want 1", len(a.feed.Posts()))
	}
}

func TestRunKeepsReadyFalseOnTokenSourceError(t *testing.T) {
	t.Parallel()

	a := newTestAppWithSyncer(&fakeMeta{}, &fakeTokenSource{err: errors.New("openbao unreachable")}, time.Millisecond)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	a.run(ctx)

	if a.ready.Load() {
		t.Error("ready = true after every cycle failed, want false")
	}
	if len(a.feed.Posts()) != 0 {
		t.Errorf("posts = %d, want 0: nothing ever synced", len(a.feed.Posts()))
	}
}

func TestRunStopsWithItsContext(t *testing.T) {
	t.Parallel()

	a := newTestAppWithSyncer(&fakeMeta{}, &fakeTokenSource{err: errors.New("unreachable")}, time.Hour)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		a.run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after its context was cancelled")
	}
}

// TestRunStaysReadyAfterALaterFailure proves readiness is sticky: once a
// cycle has succeeded, a later failed cycle must not flip it back to
// false, the same "last good feed stays" principle Syncer.Sync already
// applies to the feed contents themselves.
func TestRunStaysReadyAfterALaterFailure(t *testing.T) {
	t.Parallel()

	meta := &fakeMeta{}
	tokens := &fakeTokenSource{token: "tok-1"}
	a := newTestAppWithSyncer(meta, tokens, time.Hour)

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	a.run(ctx)
	cancel()
	if !a.ready.Load() {
		t.Fatal("ready = false after the first cycle, want true")
	}

	// From here, every further call to the TokenSource fails, mirroring
	// what the loop would see on a later cycle.
	tokens.err = fmt.Errorf("openbao unreachable")
	ctx, cancel = context.WithTimeout(t.Context(), 100*time.Millisecond)
	a.run(ctx)
	cancel()

	if !a.ready.Load() {
		t.Error("ready = false after a later failure, want it to stay true")
	}
	if len(a.feed.Posts()) != 1 {
		t.Errorf("posts = %d, want the feed from the first cycle to still be there", len(a.feed.Posts()))
	}
}
