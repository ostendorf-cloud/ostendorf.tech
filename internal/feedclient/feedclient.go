// Package feedclient is the website's only connection to the Impressionen
// section's content: it periodically asks the feed service's internal API
// for the current post list and the picture bytes behind each one, and
// keeps both in memory. A visitor's browser never reaches the feed
// service, Instagram or Meta directly; it only ever talks to this site's
// own domain. The privacy policy (web/src/views/PrivacyView.vue,
// "Instagram-Beiträge auf dieser Website") states exactly that.
//
// This is deliberately the website's one network dependency for that
// section, and the only one that replaces it: the website carries no
// Instagram or Meta credential, no OpenBao client, and makes no call to
// Meta, ever. internal/feed, running as its own binary (cmd/feed), owns
// all of that instead; see internal/instagram, which only internal/feed
// imports now.
package feedclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// RequestTimeout bounds every call this client makes. The feed service
	// is an in-cluster dependency, not Meta's own CDN, so this is tighter
	// than internal/instagram.Client's 15 second timeout to the real Graph
	// API: an unreachable feed service must fail fast rather than hold up a
	// sync cycle, so the last good cache stays in place and the next cycle
	// tries again. Applied as the one http.Client's Timeout, covering
	// connect, any redirect and reading the whole response.
	RequestTimeout = 5 * time.Second

	// maxPostsBytes bounds the post list. One entry is an id of at most 64
	// characters, an alt text of at most 150 runes and a permalink, well
	// under 1 KiB even with every character escaped, so maxPosts entries
	// fit with room to spare.
	maxPostsBytes = 16 << 20
	// MaxImageBytes bounds one picture download. The feed service already
	// enforces its own ceiling on what it ever downloaded from Meta's CDN
	// (internal/instagram.Client.MaxImageBytes); this is a second,
	// independent limit of the same size against a feed service response
	// that does not honour that ceiling, not the only one.
	MaxImageBytes = 4 << 20
	// maxPosts bounds how many posts one sync cycle will ever process,
	// mirroring internal/feedconfig's own ceiling on POST_COUNT (10000):
	// maxPostsBytes alone does not stand in for a post count limit, since
	// every entry can pull a MaxImageBytes image. A second, independent
	// limit against a feed service response that does not honour its own
	// ceiling, the same reasoning MaxImageBytes above already states for
	// image size. The feed service reads POST_COUNT=0 as the same maximum,
	// internal/instagram.MaxPosts.
	maxPosts = 10000
)

// allowedImageTypes are the formats served back to the browser.
var allowedImageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

// Post is one post as the feed service's GET /api/v1/posts answers it.
type Post struct {
	ID        string `json:"id"`
	Alt       string `json:"alt"`
	Permalink string `json:"permalink"`
}

// Client calls the feed service's internal HTTP API. Neither of its two
// calls carries or needs any credential: the feed service's own
// NetworkPolicy, not an application level token, is what is meant to keep
// this surface reachable only from the website (see internal/feed's own
// package doc).
type Client struct {
	// BaseURL is the feed service's origin, without a trailing slash.
	BaseURL string
	// HTTP sends the requests; its Timeout is RequestTimeout.
	HTTP *http.Client
}

// New returns a client against the feed service at baseURL.
func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		HTTP:    &http.Client{Timeout: RequestTimeout},
	}
}

// Posts asks the feed service for the current post list.
func (c *Client) Posts(ctx context.Context) ([]Post, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/posts", nil)
	if err != nil {
		return nil, fmt.Errorf("feedclient: build request: %w", err)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("feedclient: posts: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPostsBytes))
	if err != nil {
		return nil, fmt.Errorf("feedclient: posts: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feedclient: posts: unexpected status %d", resp.StatusCode)
	}
	var out struct {
		Posts []Post `json:"posts"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("feedclient: posts: decode response: %w", err)
	}
	return out.Posts, nil
}

// Image downloads one post's picture from the feed service, with its
// content type and ETag.
func (c *Client) Image(ctx context.Context, id string) (data []byte, contentType, etag string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/posts/"+url.PathEscape(id)+"/image", nil)
	if err != nil {
		return nil, "", "", fmt.Errorf("feedclient: build request: %w", err)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", "", fmt.Errorf("feedclient: image %s: %w", id, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("feedclient: image %s: unexpected status %d", id, resp.StatusCode)
	}
	declared, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if !allowedImageTypes[declared] {
		return nil, "", "", fmt.Errorf("feedclient: image %s: content type %q", id, declared)
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, MaxImageBytes+1))
	if err != nil {
		return nil, "", "", fmt.Errorf("feedclient: image %s: %w", id, err)
	}
	if int64(len(data)) > MaxImageBytes {
		return nil, "", "", fmt.Errorf("feedclient: image %s: larger than %d bytes", id, MaxImageBytes)
	}
	// Serve the type the bytes really have, never what the header claims.
	sniffed := http.DetectContentType(data)
	if !allowedImageTypes[sniffed] {
		return nil, "", "", fmt.Errorf("feedclient: image %s: content is %q", id, strings.SplitN(sniffed, ";", 2)[0])
	}
	etag = resp.Header.Get("ETag")
	if etag == "" {
		// The feed service always sets one (internal/feed/handlers.go); a
		// missing one is unexpected, not fatal, so one is derived the same
		// way internal/instagram.Syncer.Sync derives its own.
		sum := sha256.Sum256(data)
		etag = `"` + hex.EncodeToString(sum[:8]) + `"`
	}
	return data, sniffed, etag, nil
}
