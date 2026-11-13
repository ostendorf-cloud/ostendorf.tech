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

// Syncer refreshes the token when it is due and reloads the feed.
//
// It has two mutually exclusive ways of getting a token, described by
// Tokens below. Left nil, the token lives in this struct and nowhere else.
// old_simon-ostendorf.de kept it in OpenBao and wrote every refresh back,
// so a refreshed token survived a restart and two replicas could not
// clobber each other's refresh. This repository's own internal/webapp has
// no secret manager client, by design, so a refresh here updates the
// in-memory token only: a pod that runs for months keeps working, but a
// restart falls back to the token the environment supplies, with whatever
// age it has. Meta's long lived tokens last 60 days, so that configured
// value still has to be rotated by whoever owns the deployment; see the
// root README.md. Setting Tokens instead, as cmd/feed does, hands all of
// that over to a store-backed TokenSource such as *TokenStore, which reads,
// refreshes and writes the token back to OpenBao itself; Syncer then never
// touches Token/RefreshAge or runs its own refresh against Meta at all.
type Syncer struct {
	Client *Client
	// Token is the access token the environment supplied, used only when
	// Tokens is nil. Set it before Run or Sync is ever called; from then on
	// it is read and written only under mu, because a refresh replaces it
	// in place.
	Token string
	// RefreshAge is how old the token may get before a refresh is
	// attempted, used only when Tokens is nil. Meta refuses to refresh a
	// token younger than 24 hours; such a refusal is tolerated, logged and
	// retried next cycle, since the current token keeps working either way.
	RefreshAge time.Duration
	// Tokens, when set, is asked for the token every cycle instead of the
	// in-memory Token/RefreshAge pair above: a store-backed source already
	// handles its own refresh and write-back, so Syncer defers to it
	// entirely rather than running a second, uncoordinated refresh against
	// Meta. Left nil, Syncer behaves exactly as it always has: Token is the
	// value the environment supplied and a refresh replaces it in place,
	// which is the path internal/webapp still uses, having no store to
	// defer to.
	Tokens TokenSource
	Count  int
	Feed   *Feed
	Logger *slog.Logger
	Now    func() time.Time

	mu          sync.Mutex
	refreshedAt time.Time
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
	token, err := s.token(ctx)
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

// token returns the token to call the API with. When Tokens is set, it is
// asked and owns every bit of refresh and write-back logic itself.
// Otherwise this refreshes the in-memory Token first when it has not been
// refreshed within RefreshAge. A refresh that fails is not an error: the
// token already in hand is valid until Meta expires it, so the failure is
// logged and the next cycle tries again.
func (s *Syncer) token(ctx context.Context) (string, error) {
	if s.Tokens != nil {
		return s.Tokens.Token(ctx)
	}

	s.mu.Lock()
	token, refreshedAt := s.Token, s.refreshedAt
	s.mu.Unlock()

	if token == "" {
		return "", errors.New("instagram: no access token configured")
	}
	now := s.Now()
	// A zero refreshedAt is a process that has not refreshed yet, which
	// after a restart is every process: the configured token's real age is
	// unknown here, so it is refreshed once rather than assumed fresh.
	if !refreshedAt.IsZero() && now.Sub(refreshedAt) < s.RefreshAge {
		return token, nil
	}

	fresh, err := s.Client.RefreshToken(ctx, token)
	if err != nil {
		s.Logger.Warn("instagram token refresh failed, using the current token", "error", err.Error())
		return token, nil
	}
	s.mu.Lock()
	s.Token, s.refreshedAt = fresh, now
	s.mu.Unlock()
	s.Logger.Info("instagram token refreshed, in memory only until this process ends")
	return fresh, nil
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
