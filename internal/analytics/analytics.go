// Package analytics reports page views to Umami from the server. No script
// runs in the browser and nothing is read from the visitor's device beyond
// the request it sends anyway; the privacy policy
// (web/src/views/PrivacyView.vue, "Besucherstatistik mit Umami") depends on
// that: "Umami läuft nicht in Ihrem Browser: Es wird kein Skript geladen,
// kein Cookie gesetzt und nichts auf Ihrem Endgerät gespeichert oder
// ausgelesen. Stattdessen meldet unser Webserver jeden Aufruf einer Seite
// an Umami". Every field in payload below is one the policy names, and
// nothing else is ever sent.
package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// QueueSize bounds the views waiting for the worker; beyond it they are
	// dropped, so a slow Umami never holds a page.
	QueueSize = 256
	// SendTimeout bounds one post to Umami.
	SendTimeout = 5 * time.Second
	// LogInterval is the most often failures are logged.
	LogInterval = time.Minute

	maxUserAgent = 500
	maxReferrer  = 500
)

// View is one page view as Umami receives it.
type View struct {
	// URL is the path, without query or fragment.
	URL   string
	Title string
	// Referrer is another site's address without query, or empty.
	Referrer  string
	Language  string
	UserAgent string
	IP        netip.Addr
}

// sharedAddressSpace is carrier grade NAT and VPN overlays (RFC 6598), which
// netip does not count as private.
var sharedAddressSpace = netip.MustParsePrefix("100.64.0.0/10")

// public reports whether a is a visitor on the internet. Home, the VPN and the
// cluster's own probes come from private, loopback or link local addresses.
func public(a netip.Addr) bool {
	return a.IsValid() && a.IsGlobalUnicast() && !a.IsPrivate() && !sharedAddressSpace.Contains(a)
}

// prefetch reports a request the browser made ahead of a click that may never
// come.
func prefetch(h http.Header) bool {
	for _, name := range []string{"Sec-Purpose", "Purpose", "X-Purpose", "X-Moz"} {
		if strings.Contains(strings.ToLower(h.Get(name)), "prefetch") {
			return true
		}
	}
	return false
}

// FromRequest builds the view of a page request answered with 200, or reports
// false when it must not be counted: anything but GET, Do Not Track or Global
// Privacy Control, a prefetch, no user agent, or a client that is not public.
// ip is the client address the server already resolved behind the proxy.
func FromRequest(r *http.Request, ip netip.Addr, title, ownHost string) (View, bool) {
	ip = ip.Unmap()
	if r.Method != http.MethodGet ||
		r.Header.Get("DNT") == "1" || r.Header.Get("Sec-GPC") == "1" ||
		prefetch(r.Header) || !public(ip) {
		return View{}, false
	}
	ua := strings.TrimSpace(r.UserAgent())
	if ua == "" {
		return View{}, false
	}
	return View{
		URL:       r.URL.Path,
		Title:     title,
		Referrer:  referrer(r.Header.Get("Referer"), ownHost),
		Language:  language(r.Header.Get("Accept-Language")),
		UserAgent: truncate(ua, maxUserAgent),
		IP:        ip,
	}, true
}

// referrer keeps another site's address without query, fragment or user
// info. The site's own pages, with or without www, are no referrer.
func referrer(raw, ownHost string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	own := strings.ToLower(ownHost)
	if host == own || host == strings.TrimPrefix(own, "www.") || host == "www."+own {
		return ""
	}
	u.User, u.RawQuery, u.ForceQuery, u.Fragment, u.RawFragment = nil, "", false, "", ""
	return truncate(u.String(), maxReferrer)
}

var languageTag = regexp.MustCompile(`^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8}){0,3}$`)

// language is the first tag of Accept-Language, or empty when it is no tag.
func language(header string) string {
	first, _, _ := strings.Cut(header, ",")
	first, _, _ = strings.Cut(first, ";")
	first = strings.TrimSpace(first)
	if !languageTag.MatchString(first) {
		return ""
	}
	return first
}

// truncate cuts s to at most n bytes without splitting a character.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.ToValidUTF8(s[:n], "")
}

// Client queues views and posts them to Umami's /api/send.
type Client struct {
	endpoint  string
	websiteID string
	hostname  string
	http      *http.Client
	logger    *slog.Logger
	now       func() time.Time
	queue     chan View

	mu       sync.Mutex
	failed   int
	dropped  int
	lastErr  error
	lastLog  time.Time
	reported bool
}

// New returns a client for the Umami at baseURL, counting for websiteID under
// hostname. transport may be nil for the default one.
func New(baseURL, websiteID, hostname string, transport http.RoundTripper, logger *slog.Logger) *Client {
	return &Client{
		endpoint:  strings.TrimRight(baseURL, "/") + "/api/send",
		websiteID: websiteID,
		hostname:  hostname,
		http:      &http.Client{Transport: transport, Timeout: SendTimeout},
		logger:    logger,
		now:       time.Now,
		queue:     make(chan View, QueueSize),
	}
}

// Track queues v without waiting; a full queue drops it. Nothing here can
// block the handler that calls it, which is what keeps a slow or missing
// Umami from ever reaching a visitor.
func (c *Client) Track(v View) {
	select {
	case c.queue <- v:
	default:
		c.record(0, 1, nil)
	}
}

// Run posts queued views until ctx ends; views still queued then are lost.
// Every LogInterval it logs what failed or was dropped since the last line.
func (c *Client) Run(ctx context.Context) {
	tick := time.NewTicker(LogInterval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			c.flush()
		case v := <-c.queue:
			if err := c.Send(ctx, v); err != nil && ctx.Err() == nil {
				c.record(1, 0, err)
			}
		}
	}
}

type event struct {
	Type    string  `json:"type"`
	Payload payload `json:"payload"`
}

// payload carries ip and userAgent itself, which Umami prefers over the
// request's: the request comes from this server, not the visitor.
type payload struct {
	Website   string `json:"website"`
	Hostname  string `json:"hostname"`
	URL       string `json:"url"`
	Title     string `json:"title,omitempty"`
	Referrer  string `json:"referrer,omitempty"`
	Language  string `json:"language,omitempty"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
}

// Send posts one view. Errors name the endpoint at most, never the visitor's
// address or agent.
func (c *Client) Send(ctx context.Context, v View) error {
	body, err := json.Marshal(event{Type: "event", Payload: payload{
		Website:   c.websiteID,
		Hostname:  c.hostname,
		URL:       v.URL,
		Title:     v.Title,
		Referrer:  v.Referrer,
		Language:  v.Language,
		IP:        v.IP.String(),
		UserAgent: v.UserAgent,
	}})
	if err != nil {
		return fmt.Errorf("encode page view: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build page view request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Umami's bot check reads the agent; the payload's wins, this one agrees.
	req.Header.Set("User-Agent", v.UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("post page view: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("post page view: umami answered %d", resp.StatusCode)
	}
	return nil
}

// record counts failures and drops and logs them at most once per
// LogInterval, so an Umami that is down does not flood the log. The first one
// is logged at once; counts gathered after it come with the next line, from
// the next failure or Run's ticker.
func (c *Client) record(failed, dropped int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failed += failed
	c.dropped += dropped
	if err != nil {
		c.lastErr = err
	}
	c.logPending()
}

// flush logs pending counts once LogInterval has passed since the last line.
func (c *Client) flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logPending()
}

// logPending writes the pending counts if there are any and the interval
// allows. c.mu is held.
func (c *Client) logPending() {
	now := c.now()
	if c.failed+c.dropped == 0 || c.reported && now.Sub(c.lastLog) < LogInterval {
		return
	}
	attrs := []any{"failed", c.failed, "dropped", c.dropped}
	if c.lastErr != nil {
		attrs = append(attrs, "error", c.lastErr.Error())
	}
	c.logger.Warn("page views not counted", attrs...)
	c.failed, c.dropped, c.lastErr = 0, 0, nil
	c.lastLog, c.reported = now, true
}
