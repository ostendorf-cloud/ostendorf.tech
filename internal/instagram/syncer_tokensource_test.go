package instagram

import (
	"context"
	"errors"
	"io"
	"testing"
)

// fakeTokenSource is a hand-written TokenSource: it returns token or err,
// and counts how often it was asked, so a test can prove Sync called it
// instead of falling back to the in-memory Token/RefreshAge path.
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

func TestSyncUsesTokenSourceWhenSet(t *testing.T) {
	t.Parallel()

	meta := newFakeMeta()
	meta.media = []Media{{ID: "1", MediaType: "IMAGE", MediaURL: "https://cdn.test/a.jpg", Permalink: "https://www.instagram.com/p/A/"}}
	s := newSyncer(meta, io.Discard, fixedClock())
	// Token is deliberately left empty and RefreshAge/refresh logic
	// untouched: Tokens being set must mean Sync never falls back to any
	// of that.
	s.Token = ""
	source := &fakeTokenSource{token: "tok-1"}
	s.Tokens = source

	if err := s.Sync(t.Context()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if source.calls != 1 {
		t.Errorf("TokenSource.Token called %d times, want 1", source.calls)
	}
	if meta.refreshes() != 0 {
		t.Errorf("refresh calls = %d, want 0: Tokens must bypass the in-memory refresh entirely", meta.refreshes())
	}
	if got := meta.token(); got != "tok-1" {
		t.Errorf("media called with %q, want the TokenSource's tok-1", got)
	}
	if len(s.Feed.Posts()) != 1 {
		t.Error("feed not populated")
	}
}

func TestSyncReturnsTokenSourceError(t *testing.T) {
	t.Parallel()

	meta := newFakeMeta()
	meta.media = []Media{{ID: "1", MediaType: "IMAGE", MediaURL: "https://cdn.test/a.jpg", Permalink: "https://www.instagram.com/p/A/"}}
	s := newSyncer(meta, io.Discard, fixedClock())
	wantErr := errors.New("openbao unreachable")
	source := &fakeTokenSource{err: wantErr}
	s.Tokens = source

	err := s.Sync(t.Context())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Sync() error = %v, want it to wrap %v", err, wantErr)
	}
	if meta.mediaCalls != 0 {
		t.Errorf("Client.Media called %d times, want 0: Sync must not call Meta without a token", meta.mediaCalls)
	}
}
