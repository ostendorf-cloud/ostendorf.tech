package feedclient

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"
)

// CachedPost is one post as the website serves it: its id, alt text, the
// post's own Instagram permalink, and the picture bytes this server
// downloaded from the feed service. Mirrors internal/instagram.Post's
// shape, field order included, since internal/webapp's two handlers were
// written against that shape and read this one the same way.
type CachedPost struct {
	ID          string
	Alt         string
	Permalink   string
	Image       []byte
	ContentType string
	ETag        string
}

// Cache holds the posts currently shown; the zero value is an empty
// cache, which is also what a feed service that is unreachable, not yet
// configured, or has nothing to show leaves in place forever: the
// Impressionen section then simply renders nothing, exactly like an
// absent Instagram token did before this package existed.
type Cache struct {
	mu    sync.RWMutex
	posts []*CachedPost
	byID  map[string]*CachedPost
}

// Posts returns the posts, newest first as the feed service answered
// them. The slice must not be changed.
func (c *Cache) Posts() []*CachedPost {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.posts
}

// Post returns the post with id.
func (c *Cache) Post(id string) (*CachedPost, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.byID[id]
	return p, ok
}

// Set replaces the posts.
func (c *Cache) Set(posts []*CachedPost) {
	byID := make(map[string]*CachedPost, len(posts))
	for _, p := range posts {
		byID[p.ID] = p
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.posts, c.byID = posts, byID
}

// Syncer periodically asks the feed service for its current post list and
// the picture behind each one, keeping Cache up to date.
//
// Refresh strategy: Run polls on a jittered interval (the jitter keeps a
// restarted pod from hitting the feed service on the same schedule as the
// one it replaced, the same reasoning internal/instagram.Syncer.Run
// applies to Meta). Polling faster than the feed service's own sync
// against Meta (DefaultSyncInterval in internal/feedconfig, an hour)
// cannot show a visitor anything fresher, but costs only a cheap
// in-cluster call, so the default interval is shorter than that: see
// internal/config.DefaultFeedRefreshInterval. A failing cycle keeps the
// last good cache in place and logs rather than clearing it, the same
// "last good feed stays" rule internal/instagram.Syncer.Run applies, so a
// feed service outage costs freshness, never the section's availability.
// While the cache is still empty, the retry is capped at five minutes
// rather than waiting a full interval, the same bound Run uses there too,
// so a cold start or a still-unreachable feed service is not amplified
// into a tighter loop than that cap, whatever the configured interval is.
//
// A cold start therefore never blocks the website: New builds an empty
// Cache synchronously, Start (internal/webapp) launches Run in the
// background and returns at once, and the first page is served with no
// Impressionen section until the first cycle actually completes.
type Syncer struct {
	Client *Client
	Cache  *Cache
	Logger *slog.Logger
}

// Run syncs at start and then every interval until ctx ends.
func (s *Syncer) Run(ctx context.Context, interval time.Duration) {
	for {
		if err := s.Sync(ctx); err != nil && ctx.Err() == nil {
			s.Logger.Warn("feed sync failed, keeping the last cache", "error", err.Error(), "posts", len(s.Cache.Posts()))
			if len(s.Cache.Posts()) == 0 {
				if !sleep(ctx, min(interval, 5*time.Minute)) {
					return
				}
				continue
			}
		}
		if !sleep(ctx, interval+time.Duration(rand.Int64N(int64(interval/10)+1))) {
			return
		}
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// Sync runs one cycle: the post list, then any picture not already cached.
func (s *Syncer) Sync(ctx context.Context) error {
	posts, err := s.Client.Posts(ctx)
	if err != nil {
		return fmt.Errorf("feedclient: sync: %w", err)
	}
	if len(posts) > maxPosts {
		// A buggy or compromised feed service ignoring its own POST_COUNT
		// ceiling must not drive this process into downloading and holding
		// images for far more posts than the feed service was ever asked
		// to serve; see maxPosts's own doc comment (feedclient.go).
		posts = posts[:maxPosts]
	}

	out := make([]*CachedPost, 0, len(posts))
	var failures int
	for _, p := range posts {
		if old, ok := s.Cache.Post(p.ID); ok {
			// A post's picture does not change; reuse it instead of
			// asking the feed service again.
			out = append(out, &CachedPost{ID: p.ID, Alt: p.Alt, Permalink: p.Permalink, Image: old.Image, ContentType: old.ContentType, ETag: old.ETag})
			continue
		}
		data, contentType, etag, err := s.Client.Image(ctx, p.ID)
		if err != nil {
			failures++
			s.Logger.Warn("feed image skipped", "id", p.ID, "error", err.Error())
			continue
		}
		out = append(out, &CachedPost{ID: p.ID, Alt: p.Alt, Permalink: p.Permalink, Image: data, ContentType: contentType, ETag: etag})
	}
	if len(out) == 0 && failures > 0 {
		return fmt.Errorf("feedclient: all %d image downloads failed", failures)
	}
	// An empty answer is more likely a feed service hiccup than every post
	// having been deleted at once; it never replaces posts being shown.
	if len(out) == 0 && len(s.Cache.Posts()) > 0 {
		return fmt.Errorf("feedclient: no usable posts among %d returned", len(posts))
	}
	s.Cache.Set(out)
	s.Logger.Info("feed cache updated", "posts", len(out), "failed_images", failures)
	return nil
}
