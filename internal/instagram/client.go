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
	// DefaultMaxImageBytes is far above the 1080 px JPEGs Instagram serves
	// (a few hundred KiB) and keeps 20 of them inside a 128 MiB pod, the
	// limit charts/ostendorf.tech/values.yaml sets.
	DefaultMaxImageBytes = 4 << 20
)

// allowedImageTypes are the formats served back to browsers.
var allowedImageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

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
}

// NewClient returns a client for the real API with a 15 second timeout.
func NewClient(version string) *Client {
	return &Client{
		BaseURL:       DefaultBaseURL,
		Version:       version,
		HTTP:          &http.Client{Timeout: 15 * time.Second},
		MaxImageBytes: DefaultMaxImageBytes,
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
	if resp.StatusCode != http.StatusOK {
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

// Media returns the newest posts of the token's account, newest first.
func (c *Client) Media(ctx context.Context, token string, limit int) ([]Media, error) {
	params := url.Values{}
	params.Set("fields", "id,caption,media_type,media_url,thumbnail_url,permalink,timestamp")
	params.Set("limit", strconv.Itoa(limit))
	params.Set("access_token", token)
	var resp struct {
		Data []Media `json:"data"`
	}
	if err := c.getJSON(ctx, c.endpoint("me/media"), params, &resp); err != nil {
		return nil, fmt.Errorf("instagram media: %w", err)
	}
	if len(resp.Data) > limit {
		resp.Data = resp.Data[:limit]
	}
	return resp.Data, nil
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

// Image downloads one picture and returns it with its sniffed content type.
func (c *Client) Image(ctx context.Context, rawURL string) ([]byte, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, "", errors.New("instagram image: not an https URL")
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
