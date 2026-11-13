package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

const (
	websiteID = "0f1e2d3c-4b5a-4968-8776-655443322110"
	ownHost   = "ostendorf.tech"
)

var visitor = netip.MustParseAddr("203.0.113.7")

func pageRequest(t *testing.T, headers ...string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://ostendorf.tech/de/privacy/?utm_source=x#top", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 TestBrowser/1.0")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	return req
}

func TestFromRequest(t *testing.T) {
	t.Parallel()

	req := pageRequest(t, "Referer", "https://duckduckgo.com/?q=veranstaltungstechnik+aachen", "Accept-Language", "de-DE,de;q=0.9,en;q=0.8")
	v, ok := FromRequest(req, visitor, "Datenschutzerklärung – Simon Ostendorf", ownHost)
	if !ok {
		t.Fatal("a plain page view was not counted")
	}
	want := View{
		URL:       "/de/privacy/",
		Title:     "Datenschutzerklärung – Simon Ostendorf",
		Referrer:  "https://duckduckgo.com/",
		Language:  "de-DE",
		UserAgent: "Mozilla/5.0 TestBrowser/1.0",
		IP:        visitor,
	}
	if v != want {
		t.Errorf("view = %+v, want %+v", v, want)
	}

	v, ok = FromRequest(pageRequest(t), netip.MustParseAddr("2001:db8::1"), "x", ownHost)
	if !ok || v.IP.String() != "2001:db8::1" || v.Referrer != "" || v.Language != "" {
		t.Errorf("IPv6 view = %+v, %v", v, ok)
	}
	v, ok = FromRequest(pageRequest(t), netip.MustParseAddr("::ffff:203.0.113.7"), "x", ownHost)
	if !ok || v.IP != visitor {
		t.Errorf("mapped address not unmapped: %+v, %v", v, ok)
	}
}

func TestFromRequestLeavesOut(t *testing.T) {
	t.Parallel()

	head := pageRequest(t)
	head.Method = http.MethodHead
	post := pageRequest(t)
	post.Method = http.MethodPost
	noAgent := pageRequest(t)
	noAgent.Header.Del("User-Agent")
	blankAgent := pageRequest(t, "User-Agent", "   ")

	cases := map[string]struct {
		req *http.Request
		ip  string
	}{
		"HEAD":                {head, "203.0.113.7"},
		"POST":                {post, "203.0.113.7"},
		"Do Not Track":        {pageRequest(t, "DNT", "1"), "203.0.113.7"},
		"Global Privacy":      {pageRequest(t, "Sec-GPC", "1"), "203.0.113.7"},
		"Sec-Purpose":         {pageRequest(t, "Sec-Purpose", "prefetch;prerender"), "203.0.113.7"},
		"Purpose":             {pageRequest(t, "Purpose", "prefetch"), "203.0.113.7"},
		"X-Moz":               {pageRequest(t, "X-Moz", "Prefetch"), "203.0.113.7"},
		"X-Purpose":           {pageRequest(t, "X-Purpose", "preview, prefetch"), "203.0.113.7"},
		"no user agent":       {noAgent, "203.0.113.7"},
		"blank user agent":    {blankAgent, "203.0.113.7"},
		"home network":        {pageRequest(t), "10.10.30.5"},
		"home network /16":    {pageRequest(t), "192.168.1.20"},
		"pod network":         {pageRequest(t), "172.16.0.9"},
		"shared address":      {pageRequest(t), "100.64.3.4"},
		"loopback":            {pageRequest(t), "127.0.0.1"},
		"IPv6 loopback":       {pageRequest(t), "::1"},
		"unique local IPv6":   {pageRequest(t), "fd00:10:120::10"},
		"link local IPv6":     {pageRequest(t), "fe80::1"},
		"unspecified":         {pageRequest(t), "0.0.0.0"},
		"invalid address":     {pageRequest(t), ""},
		"mapped private IPv4": {pageRequest(t), "::ffff:10.0.0.1"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var ip netip.Addr
			if tc.ip != "" {
				ip = netip.MustParseAddr(tc.ip)
			}
			if v, ok := FromRequest(tc.req, ip, "x", ownHost); ok {
				t.Errorf("counted: %+v", v)
			}
		})
	}

	// DNT: 0 is no objection.
	if _, ok := FromRequest(pageRequest(t, "DNT", "0"), visitor, "x", ownHost); !ok {
		t.Error("DNT: 0 was not counted")
	}
}

func TestReferrer(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                                           "",
		"https://ostendorf.tech/de/imprint/":         "",
		"https://OstenDorf.Tech/":                    "",
		"https://www.ostendorf.tech/de/agb/":         "",
		"https://www.google.com/":                    "https://www.google.com/",
		"https://www.google.com/search?q=secret#x":   "https://www.google.com/search",
		"https://user:pass@example.org/path?token=1": "https://example.org/path",
		"http://example.org":                         "http://example.org",
		"android-app://com.google.android.gm/":       "",
		"javascript:alert(1)":                        "",
		"/relative/path":                             "",
		"https://":                                   "",
		"::not a url":                                "",
		"https://ostendorf.tech.evil.example/":       "https://ostendorf.tech.evil.example/",
	}
	for raw, want := range cases {
		if got := referrer(raw, ownHost); got != want {
			t.Errorf("referrer(%q) = %q, want %q", raw, got, want)
		}
	}
	// A site served under www counts its bare name as its own as well.
	if got := referrer("https://example.org/", "www.example.org"); got != "" {
		t.Errorf("bare name of the www own host = %q", got)
	}
	long := "https://example.org/" + strings.Repeat("ä", 400)
	if got := referrer(long, ownHost); len(got) > maxReferrer || !utf8.ValidString(got) {
		t.Errorf("long referrer: %d bytes, valid %v", len(got), utf8.ValidString(got))
	}
}

func TestLanguage(t *testing.T) {
	t.Parallel()

	// A slice, not a map: one case is a header with the padding a browser
	// may send around the tag, which a map key cannot carry readably.
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{name: "weighted list", header: "de-DE,de;q=0.9,en;q=0.8", want: "de-DE"},
		{name: "bare tag", header: "en", want: "en"},
		{name: "padded tag with a weight", header: " fr-CH ; q=0.7", want: "fr-CH"},
		{name: "script and region", header: "zh-Hant-TW", want: "zh-Hant-TW"},
		{name: "wildcard", header: "*", want: ""},
		{name: "empty", header: "", want: ""},
		{name: "underscore instead of dash", header: "de_DE", want: ""},
		{name: "markup", header: "<script>", want: ""},
		{name: "longer than a tag may be", header: strings.Repeat("a", 9), want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := language(tc.header); got != tc.want {
				t.Errorf("language(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	if got := truncate("abc", 5); got != "abc" {
		t.Errorf("short = %q", got)
	}
	got := truncate(strings.Repeat("é", 300), maxUserAgent+1)
	if len(got) > maxUserAgent+1 || !utf8.ValidString(got) {
		t.Errorf("cut through a character: %d bytes", len(got))
	}
}

// fakeUmami records what reaches /api/send and answers with status. It
// never opens a socket: it is the transport itself.
type fakeUmami struct {
	mu       sync.Mutex
	status   int
	err      error
	requests []*http.Request
	bodies   [][]byte
	seen     chan struct{}
}

func (f *fakeUmami) RoundTrip(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.bodies = append(f.bodies, body)
	f.mu.Unlock()
	if f.seen != nil {
		f.seen <- struct{}{}
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{
		StatusCode: f.status,
		Body:       io.NopCloser(strings.NewReader(`{"cache":"x","sessionId":"y","visitId":"z"}`)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    req,
	}, nil
}

func (f *fakeUmami) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func newClient(f *fakeUmami, logs *bytes.Buffer) *Client {
	return New("http://umami.umami.svc.cluster.local/", websiteID, ownHost, f, slog.New(slog.NewJSONHandler(logs, nil)))
}

// awaitRequests waits for n requests to reach the fake, or fails rather
// than hanging. It waits on the condition, never on a fixed sleep.
func awaitRequests(t *testing.T, f *fakeUmami, n int) {
	t.Helper()
	for range n {
		select {
		case <-f.seen:
		case <-time.After(5 * time.Second):
			t.Fatalf("only %d of %d views reached Umami", f.count(), n)
		}
	}
}

func TestSend(t *testing.T) {
	t.Parallel()

	f := &fakeUmami{status: http.StatusOK}
	c := newClient(f, &bytes.Buffer{})
	v := View{URL: "/de/", Title: "Simon Ostendorf – Veranstaltungstechnik", Referrer: "https://www.google.com/", Language: "de-DE", UserAgent: "Mozilla/5.0 TestBrowser/1.0", IP: visitor}
	if err := c.Send(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	req := f.requests[0]
	if req.Method != http.MethodPost || req.URL.String() != "http://umami.umami.svc.cluster.local/api/send" {
		t.Errorf("request = %s %s", req.Method, req.URL)
	}
	if req.Header.Get("Content-Type") != "application/json" || req.Header.Get("User-Agent") != v.UserAgent {
		t.Errorf("headers = %v", req.Header)
	}
	for _, h := range []string{"Cookie", "Authorization", "X-Forwarded-For"} {
		if req.Header.Get(h) != "" {
			t.Errorf("%s sent", h)
		}
	}
	var got map[string]any
	if err := json.Unmarshal(f.bodies[0], &got); err != nil {
		t.Fatal(err)
	}
	body, _ := got["payload"].(map[string]any)
	want := map[string]any{
		"website": websiteID, "hostname": ownHost, "url": "/de/",
		"title": v.Title, "referrer": "https://www.google.com/", "language": "de-DE",
		"ip": "203.0.113.7", "userAgent": v.UserAgent,
	}
	if got["type"] != "event" || len(body) != len(want) {
		t.Fatalf("body = %s", f.bodies[0])
	}
	for k, w := range want {
		if body[k] != w {
			t.Errorf("payload %s = %v, want %v", k, body[k], w)
		}
	}

	// Empty optional fields are left out, not sent empty.
	if err := c.Send(t.Context(), View{URL: "/de/agb/", UserAgent: "x", IP: visitor}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{`"title"`, `"referrer"`, `"language"`} {
		if bytes.Contains(f.bodies[1], []byte(k)) {
			t.Errorf("empty %s sent: %s", k, f.bodies[1])
		}
	}
}

func TestSendErrors(t *testing.T) {
	t.Parallel()

	f := &fakeUmami{status: http.StatusBadRequest}
	c := newClient(f, &bytes.Buffer{})
	v := View{URL: "/de/", UserAgent: "SecretAgent/1.0", IP: visitor}
	err := c.Send(t.Context(), v)
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("400 = %v", err)
	}
	f.status, f.err = 0, errors.New("connection refused")
	err = c.Send(t.Context(), v)
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("transport error = %v", err)
	}
	if strings.Contains(err.Error(), "203.0.113.7") || strings.Contains(err.Error(), "SecretAgent") {
		t.Errorf("error leaks the visitor: %v", err)
	}
}

func TestRunSendsQueuedViews(t *testing.T) {
	t.Parallel()

	f := &fakeUmami{status: http.StatusOK, seen: make(chan struct{}, 8)}
	c := newClient(f, &bytes.Buffer{})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		c.Run(ctx)
		close(done)
	}()
	for range 3 {
		c.Track(View{URL: "/de/", UserAgent: "x", IP: visitor})
	}
	awaitRequests(t, f, 3)

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop with its context")
	}
}

func TestTrackDropsWhenFull(t *testing.T) {
	t.Parallel()

	logs := &bytes.Buffer{}
	c := newClient(&fakeUmami{status: http.StatusOK}, logs)
	c.queue = make(chan View, 2)
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	now := start
	c.now = func() time.Time { return now }

	// Nobody runs the worker: two fit, the rest are dropped without blocking.
	for range 5 {
		c.Track(View{URL: "/de/", UserAgent: "x", IP: visitor})
	}
	if len(c.queue) != 2 {
		t.Errorf("queue holds %d, want 2", len(c.queue))
	}
	if n := strings.Count(logs.String(), "page views not counted"); n != 1 {
		t.Errorf("logged %d times within the interval, want 1: %s", n, logs)
	}
	if !strings.Contains(logs.String(), `"dropped":1`) {
		t.Errorf("first drop not logged at once: %s", logs)
	}

	// After the interval the next drop reports what gathered meanwhile.
	now = start.Add(LogInterval)
	c.Track(View{URL: "/de/", UserAgent: "x", IP: visitor})
	if n := strings.Count(logs.String(), "page views not counted"); n != 2 || !strings.Contains(logs.String(), `"dropped":3`) {
		t.Errorf("second log line missing or wrong count: %s", logs)
	}
}

func TestFlushLogsWhatGatheredMeanwhile(t *testing.T) {
	t.Parallel()

	logs := &bytes.Buffer{}
	c := newClient(&fakeUmami{status: http.StatusOK}, logs)
	c.queue = make(chan View)
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	now := start
	c.now = func() time.Time { return now }

	// Nothing pending: no line, even long after the last one.
	c.flush()
	if logs.Len() != 0 {
		t.Fatalf("flush without anything pending logged: %s", logs)
	}

	// A burst inside one interval: the first drop at once, the rest wait.
	for range 4 {
		c.Track(View{URL: "/de/", UserAgent: "x", IP: visitor})
	}
	now = start.Add(LogInterval / 2)
	c.flush()
	if n := strings.Count(logs.String(), "page views not counted"); n != 1 {
		t.Fatalf("flush inside the interval logged: %s", logs)
	}

	// The burst is over; the ticker's flush after the interval reports it.
	now = start.Add(LogInterval)
	c.flush()
	if n := strings.Count(logs.String(), "page views not counted"); n != 2 || !strings.Contains(logs.String(), `"dropped":3`) {
		t.Errorf("trailing drops not reported: %s", logs)
	}

	// And then nothing more until something new happens.
	now = start.Add(3 * LogInterval)
	c.flush()
	if n := strings.Count(logs.String(), "page views not counted"); n != 2 {
		t.Errorf("flush repeated an empty report: %s", logs)
	}
}

func TestRunLogsFailuresRateLimited(t *testing.T) {
	t.Parallel()

	logs := &bytes.Buffer{}
	f := &fakeUmami{err: errors.New("connection refused"), seen: make(chan struct{}, 8)}
	c := newClient(f, logs)
	c.now = func() time.Time { return time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC) }
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go c.Run(ctx)
	for range 4 {
		c.Track(View{URL: "/de/", UserAgent: "SecretAgent/1.0", IP: visitor})
	}
	awaitRequests(t, f, 4)
	cancel()

	c.mu.Lock()
	out := logs.String()
	c.mu.Unlock()
	if n := strings.Count(out, "page views not counted"); n != 1 {
		t.Errorf("logged %d times, want 1: %s", n, out)
	}
	if !strings.Contains(out, "connection refused") || strings.Contains(out, "203.0.113.7") || strings.Contains(out, "SecretAgent") {
		t.Errorf("log line = %s", out)
	}
}
