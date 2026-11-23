// Package instagram loads the latest posts of the business account through
// the Instagram API with Instagram Login and keeps them, images included, in
// memory, so visitors' browsers never connect to Meta. The privacy policy
// (web/src/views/PrivacyView.vue, "Instagram-Beiträge auf dieser Website")
// states exactly that, so it is a contract, not an implementation detail:
// the pictures are fetched here, held here and served from this site's own
// domain, and the only thing that ever reaches Meta is this server's own
// address and the access token.
package instagram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the Instagram Graph host.
	DefaultBaseURL = "https://graph.instagram.com"
	// DefaultVersion pins the Graph API version; Meta retires a version
	// about two years after its release, so this is reviewed rather than
	// left to float. internal/config's DefaultInstagramAPIVersion reads
	// this constant, so the two can never drift.
	DefaultVersion = "v23.0"

	maxAPIBytes = 1 << 20
	// DefaultMaxImageBytes bounds one picture. It is far above the 1080 px
	// JPEGs Instagram serves (a few hundred KiB); what a full cache costs is
	// the number of posts times the real size of their pictures, see the
	// chart's README.
	DefaultMaxImageBytes = 4 << 20

	// DefaultPostsPerRequest is how many posts one call to the media edge
	// asks for.
	DefaultPostsPerRequest = 50
	// MaxPostsPerRequest is the most Meta's limit parameter accepts.
	MaxPostsPerRequest = 100
	// DefaultPagePause is the pause between two pages of one cycle. The
	// pages are fetched one after the other, never in parallel, and the
	// pause spreads a cycle of many pages over time, so a full fetch stays
	// well inside Meta's rate limits.
	DefaultPagePause = 500 * time.Millisecond
	// maxPages bounds one cycle even with no cap on the posts: 1000 pages of
	// 100 posts is far beyond any real account, so reaching it means a
	// misbehaving answer, not a large account.
	maxPages = 1000
)

// Error kinds of a media cycle. The returned error wraps one of them, so a
// caller can tell them apart with errors.Is; none of them carries the access
// token or the paging cursor.
var (
	// ErrRateLimited is a 429 from Meta.
	ErrRateLimited = errors.New("rate limited by Meta")
	// ErrRefused is any other 4xx from Meta.
	ErrRefused = errors.New("request refused by Meta")
	// ErrInvalidCursor is a next page whose cursor is missing or fails
	// validCursor.
	ErrInvalidCursor = errors.New("invalid paging cursor")
	// ErrCursorLoop is a cursor that came back a second time in one cycle.
	ErrCursorLoop = errors.New("paging cursor repeated")
	// ErrTooManyPages is a cycle that reached maxPages.
	ErrTooManyPages = errors.New("too many pages")
)

// validCursor is what a paging cursor from Meta may look like before it is
// sent back: base64 like characters and a bounded length. The cursor only
// ever travels as the after query parameter on this client's own BaseURL,
// encoded by url.Values, so the check bounds what an answer can make this
// client send, not where it is sent.
var validCursor = regexp.MustCompile(`^[A-Za-z0-9_=+/-]{1,512}$`)

// allowedImageTypes are the formats served back to browsers.
var allowedImageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

// DefaultAllowedImageHosts are Meta's actual CDN domains for Instagram
// media. This is a domain *suffix* allowlist, not a list of exact
// hostnames: Meta rotates and adds CDN subdomains (for example
// "scontent-waw2-1.cdninstagram.com" or "video.xx.fna.fbcdn.net") without
// notice, so a fixed hostname list would silently start breaking the feed
// the moment Meta pointed a post at a subdomain not already on it.
var DefaultAllowedImageHosts = []string{"cdninstagram.com", "fbcdn.net"}

// Media is one post as the API returns it.
type Media struct {
	ID           string `json:"id"`
	Caption      string `json:"caption"`
	MediaType    string `json:"media_type"`
	MediaURL     string `json:"media_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Permalink    string `json:"permalink"`
	Timestamp    string `json:"timestamp"`
}

// ImageURL is the picture to show: the thumbnail for videos, the first
// image for carousels (media_url), the image itself otherwise.
func (m Media) ImageURL() string {
	if m.MediaType == "VIDEO" {
		return m.ThumbnailURL
	}
	return m.MediaURL
}

// Client calls the Graph API and downloads images.
type Client struct {
	// BaseURL is the Graph host without a trailing slash; tests point it at a fake.
	BaseURL string
	// Version is the pinned API version, empty for unversioned calls.
	Version string
	// HTTP sends the requests.
	HTTP *http.Client
	// MaxImageBytes bounds one image download.
	MaxImageBytes int64
	// AllowedImageHosts bounds which hosts Image will ever fetch from, as
	// domain suffixes checked by hostAllowed. Overridable primarily so
	// tests can point it at a fake host instead of real Meta domains.
	AllowedImageHosts []string
	// PagePause is the wait between two pages of one Media call; zero
	// means none. NewClient sets DefaultPagePause.
	PagePause time.Duration
}

// NewClient returns a client for the real API with a 15 second timeout.
func NewClient(version string) *Client {
	return &Client{
		BaseURL:           DefaultBaseURL,
		Version:           version,
		HTTP:              &http.Client{Timeout: 15 * time.Second},
		MaxImageBytes:     DefaultMaxImageBytes,
		AllowedImageHosts: DefaultAllowedImageHosts,
		PagePause:         DefaultPagePause,
	}
}

// scrub drops the request URL from transport errors: it carries the access
// token (API calls) or a signed CDN link (images), neither belongs in a log.
func scrub(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s: %w", urlErr.Op, urlErr.Err)
	}
	return err
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func describe(status int, body []byte) error {
	var e apiError
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
		msg := e.Error.Message
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return fmt.Errorf("status %d, %s %d: %s", status, e.Error.Type, e.Error.Code, msg)
	}
	return fmt.Errorf("unexpected status %d", status)
}

func (c *Client) getJSON(ctx context.Context, endpoint string, params url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return scrub(err)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return scrub(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAPIBytes))
	if err != nil {
		return scrub(err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return fmt.Errorf("%w: %w", ErrRateLimited, describe(resp.StatusCode, body))
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return fmt.Errorf("%w: %w", ErrRefused, describe(resp.StatusCode, body))
	case resp.StatusCode != http.StatusOK:
		return describe(resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) endpoint(path string) string {
	if c.Version == "" {
		return c.BaseURL + "/" + path
	}
	return c.BaseURL + "/" + c.Version + "/" + path
}

// mediaPage is one answer of the media edge. Paging.Next is kept as raw
// JSON on purpose: only its presence is read, as the signal that another
// page exists, and its value, a full URL Meta built, is never parsed,
// logged or requested.
type mediaPage struct {
	Data   []Media `json:"data"`
	Paging struct {
		Cursors struct {
			After string `json:"after"`
		} `json:"cursors"`
		Next json.RawMessage `json:"next"`
	} `json:"paging"`
}

// hasNext reports whether the answer announces a further page.
func (p *mediaPage) hasNext() bool {
	next := strings.TrimSpace(string(p.Paging.Next))
	return next != "" && next != "null" && next != `""`
}

// MediaResult is what one Media call collected: the posts, newest first,
// and how many pages it took.
type MediaResult struct {
	Media []Media
	Pages int
}

// Media returns the posts of the token's account, newest first. It asks for
// perRequest posts per call (1 to MaxPostsPerRequest) and follows the
// paging cursor until Meta announces no further page or maxPosts posts are
// collected; maxPosts 0 means no cap. The pages are fetched one after the
// other with PagePause between them.
//
// Every page is requested from this client's own endpoint, with the cursor
// from paging.cursors.after as the after parameter. The URL in paging.next
// is never followed, so an answer cannot send the client, or the access
// token, to another host. The cursor is checked against validCursor, and a
// cursor seen twice in one call ends it with ErrCursorLoop. Neither the
// token nor the cursor appears in a returned error.
//
// On an error the result holds what the pages before it collected, so the
// caller can tell how far the cycle got; internal/instagram.Syncer still
// keeps the last good feed until a cycle completes.
func (c *Client) Media(ctx context.Context, token string, perRequest, maxPosts int) (MediaResult, error) {
	var res MediaResult
	if perRequest < 1 || perRequest > MaxPostsPerRequest {
		return res, fmt.Errorf("instagram media: posts per request must be between 1 and %d", MaxPostsPerRequest)
	}
	if maxPosts < 0 {
		return res, errors.New("instagram media: the post cap must not be negative")
	}
	seen := map[string]bool{}
	after := ""
	for {
		if res.Pages >= maxPages {
			return res, fmt.Errorf("instagram media: %w (%d)", ErrTooManyPages, maxPages)
		}
		if res.Pages > 0 && !pause(ctx, c.PagePause) {
			return res, fmt.Errorf("instagram media: %w", ctx.Err())
		}
		limit := perRequest
		if maxPosts > 0 {
			limit = min(limit, maxPosts-len(res.Media))
		}
		params := url.Values{}
		params.Set("fields", "id,caption,media_type,media_url,thumbnail_url,permalink,timestamp")
		params.Set("limit", strconv.Itoa(limit))
		if after != "" {
			params.Set("after", after)
		}
		params.Set("access_token", token)
		var page mediaPage
		if err := c.getJSON(ctx, c.endpoint("me/media"), params, &page); err != nil {
			return res, fmt.Errorf("instagram media page %d: %w", res.Pages+1, err)
		}
		res.Pages++
		if len(page.Data) > limit {
			page.Data = page.Data[:limit]
		}
		res.Media = append(res.Media, page.Data...)
		if maxPosts > 0 && len(res.Media) >= maxPosts {
			return res, nil
		}
		if !page.hasNext() || len(page.Data) == 0 {
			return res, nil
		}
		next := page.Paging.Cursors.After
		if !validCursor.MatchString(next) {
			return res, fmt.Errorf("instagram media page %d: %w", res.Pages, ErrInvalidCursor)
		}
		if seen[next] {
			return res, fmt.Errorf("instagram media page %d: %w", res.Pages, ErrCursorLoop)
		}
		seen[next] = true
		after = next
	}
}

// pause waits d, or not at all for d <= 0. It reports false when ctx ended
// first.
func pause(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// RefreshToken exchanges a long lived token for a new one valid 60 days.
// The endpoint is unversioned in Meta's documentation.
func (c *Client) RefreshToken(ctx context.Context, token string) (string, error) {
	params := url.Values{}
	params.Set("grant_type", "ig_refresh_token")
	params.Set("access_token", token)
	var resp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := c.getJSON(ctx, c.BaseURL+"/refresh_access_token", params, &resp); err != nil {
		return "", fmt.Errorf("instagram token refresh: %w", err)
	}
	if resp.AccessToken == "" {
		return "", errors.New("instagram token refresh: no token in response")
	}
	return resp.AccessToken, nil
}

// hostAllowed reports whether host exactly matches one of allowed, or is
// a subdomain of one, bounded on a label boundary. A bare
// strings.HasSuffix on the domain string alone would be a classic
// bypass: "evil-cdninstagram.com" has "cdninstagram.com" as a string
// suffix without being a subdomain of it at all. Matching only on
// host == domain or host ending in "."+domain closes that.
func hostAllowed(host string, allowed []string) bool {
	host = strings.ToLower(host)
	for _, domain := range allowed {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

// Image downloads one picture and returns it with its sniffed content type.
func (c *Client) Image(ctx context.Context, rawURL string) ([]byte, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, "", errors.New("instagram image: not an https URL")
	}
	if !hostAllowed(u.Hostname(), c.AllowedImageHosts) {
		return nil, "", fmt.Errorf("instagram image: host %q is not an allowed Meta CDN host", u.Hostname())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", scrub(err)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("instagram image from %s: %w", u.Host, scrub(err))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("instagram image from %s: unexpected status %d", u.Host, resp.StatusCode)
	}
	declared, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if !allowedImageTypes[declared] {
		return nil, "", fmt.Errorf("instagram image from %s: content type %q", u.Host, declared)
	}
	if resp.ContentLength > c.MaxImageBytes {
		return nil, "", fmt.Errorf("instagram image from %s: %d bytes", u.Host, resp.ContentLength)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, c.MaxImageBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("instagram image from %s: %w", u.Host, scrub(err))
	}
	if int64(len(data)) > c.MaxImageBytes {
		return nil, "", fmt.Errorf("instagram image from %s: larger than %d bytes", u.Host, c.MaxImageBytes)
	}
	// Serve the type the bytes really have, never what the header claims.
	sniffed := http.DetectContentType(data)
	if !allowedImageTypes[sniffed] {
		return nil, "", fmt.Errorf("instagram image from %s: content is %q", u.Host, strings.SplitN(sniffed, ";", 2)[0])
	}
	return data, sniffed, nil
}
