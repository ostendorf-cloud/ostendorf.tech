package instagram

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"regexp"
	"strings"
	"sync"
	"time"
)

// maxAltRunes keeps alt texts short; captions can be 2200 characters.
const maxAltRunes = 150

// validID matches Instagram media IDs; anything else never reaches the map.
var validID = regexp.MustCompile(`^[0-9A-Za-z_]{1,64}$`)

// ValidID reports whether id can name a post.
func ValidID(id string) bool { return validID.MatchString(id) }

// Post is one downloaded post.
type Post struct {
	ID          string
	Permalink   string
	Alt         string
	Image       []byte
	ContentType string
	ETag        string
}

// Feed holds the current posts; the zero value is an empty feed, which is
// also what a deployment without an access token serves forever: the
// Impressionen section then simply has nothing to render.
type Feed struct {
	mu    sync.RWMutex
	posts []*Post
	byID  map[string]*Post
}

// Posts returns the posts, newest first. The slice must not be changed.
func (f *Feed) Posts() []*Post {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.posts
}

// Post returns the post with id.
func (f *Feed) Post(id string) (*Post, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	p, ok := f.byID[id]
	return p, ok
}

// Set replaces the posts.
func (f *Feed) Set(posts []*Post) {
	byID := make(map[string]*Post, len(posts))
	for _, p := range posts {
		byID[p.ID] = p
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.posts, f.byID = posts, byID
}

// TokenSource returns the token to call the Graph API with, handling its
// own refresh. *TokenStore (token.go) is one, backed by OpenBao.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// Syncer asks Tokens for a usable token and reloads the feed.
//
// cmd/feed is this package's only caller now: internal/webapp lost its
// own direct Meta client and in-memory token when the website moved to
// internal/feedclient against this binary's own HTTP API instead (see
// that package and internal/feed). Syncer therefore always defers to a
// store-backed TokenSource such as *TokenStore (token.go), which reads,
// refreshes and writes the token back to OpenBao itself; Syncer never
// runs a refresh against Meta on its own, and carries no token of its
// own to protect with a mutex.
type Syncer struct {
	Client *Client
	// Tokens is asked for the token every cycle; required.
	Tokens TokenSource
	Count  int
	Feed   *Feed
	Logger *slog.Logger
}

// Run syncs at start and then every interval until ctx ends. Errors are
// logged and the last good feed stays: a failing sync must never cost a
// visitor the page, only the freshness of what the section shows.
func (s *Syncer) Run(ctx context.Context, interval time.Duration) {
	for {
		if err := s.Sync(ctx); err != nil && ctx.Err() == nil {
			s.Logger.Warn("instagram sync failed, keeping the last feed", "error", err.Error(), "posts", len(s.Feed.Posts()))
			// Retry sooner while there is nothing to show at all.
			if len(s.Feed.Posts()) == 0 {
				if !sleep(ctx, min(interval, 5*time.Minute)) {
					return
				}
				continue
			}
		}
		// Jitter keeps a restarted pod from hitting Meta on the same
		// schedule as the one it replaced.
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

// Sync runs one cycle: token, then media, then images.
func (s *Syncer) Sync(ctx context.Context) error {
	if s.Tokens == nil {
		return errors.New("instagram: no token source configured")
	}
	token, err := s.Tokens.Token(ctx)
	if err != nil {
		return err
	}
	media, err := s.Client.Media(ctx, token, s.Count)
	if err != nil {
		return err
	}

	var posts []*Post
	var failures int
	for _, m := range media {
		if !ValidID(m.ID) || !strings.HasPrefix(m.Permalink, "https://www.instagram.com/") || m.ImageURL() == "" {
			continue
		}
		if old, ok := s.Feed.Post(m.ID); ok {
			// A post's picture does not change; reuse it instead of downloading again.
			posts = append(posts, &Post{ID: m.ID, Permalink: m.Permalink, Alt: altText(m), Image: old.Image, ContentType: old.ContentType, ETag: old.ETag})
			continue
		}
		data, contentType, err := s.Client.Image(ctx, m.ImageURL())
		if err != nil {
			failures++
			s.Logger.Warn("instagram image skipped", "id", m.ID, "error", err.Error())
			continue
		}
		sum := sha256.Sum256(data)
		posts = append(posts, &Post{
			ID: m.ID, Permalink: m.Permalink, Alt: altText(m),
			Image: data, ContentType: contentType, ETag: `"` + hex.EncodeToString(sum[:8]) + `"`,
		})
	}
	if len(posts) == 0 && failures > 0 {
		return fmt.Errorf("instagram: all %d image downloads failed", failures)
	}
	// An empty answer is more likely an API hiccup than a deleted account;
	// it never replaces posts that are being shown.
	if len(posts) == 0 && len(s.Feed.Posts()) > 0 {
		return fmt.Errorf("instagram: no usable posts among %d returned", len(media))
	}
	s.Feed.Set(posts)
	s.Logger.Info("instagram feed updated", "posts", len(posts), "failed_images", failures)
	return nil
}

// altText describes a post by its caption, one line and shortened.
func altText(m Media) string {
	caption := strings.Join(strings.Fields(m.Caption), " ")
	if caption == "" {
		if t, err := time.Parse("2006-01-02T15:04:05-0700", m.Timestamp); err == nil {
			return "Instagram-Beitrag vom " + t.Format("02.01.2006")
		}
		return "Instagram-Beitrag"
	}
	r := []rune(caption)
	if len(r) > maxAltRunes {
		return strings.TrimSpace(string(r[:maxAltRunes-1])) + "…"
	}
	return caption
}
