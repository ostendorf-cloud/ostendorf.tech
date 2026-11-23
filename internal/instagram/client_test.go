package instagram

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/secrettest"
)

// TestHostAllowed covers hostAllowed's actual decision logic against the
// bypass shapes a careless suffix check would get wrong.
func TestHostAllowed(t *testing.T) {
	t.Parallel()

	allowed := []string{"cdninstagram.com", "fbcdn.net"}
	tests := map[string]struct {
		host string
		want bool
	}{
		"exact domain match":             {"cdninstagram.com", true},
		"real subdomain":                 {"scontent-waw2-1.cdninstagram.com", true},
		"real subdomain of other domain": {"video.xx.fna.fbcdn.net", true},
		"naive-suffix bypass":            {"evil-cdninstagram.com", false},
		"reversed/prefix bypass":         {"cdninstagram.com.evil.com", false},
		"unrelated host":                 {"attacker.example.com", false},
		"case folding":                   {"CDNInstagram.com", true},
		"empty host":                     {"", false},
		"unrelated cdn":                  {"scontent.akamaihd.net", false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := hostAllowed(tc.host, allowed); got != tc.want {
				t.Errorf("hostAllowed(%q, %v) = %v, want %v", tc.host, allowed, got, tc.want)
			}
		})
	}
}

// countingTransport records whether it was ever invoked and fails the test
// if it is: used to prove a disallowed host is rejected before any HTTP
// request is sent.
type countingTransport struct {
	t     *testing.T
	calls int
}

func (c *countingTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	c.t.Helper()
	c.calls++
	c.t.Fatal("transport was called for a host that should have been rejected before any request")
	return nil, nil
}

func TestImageRejectsDisallowedHost(t *testing.T) {
	t.Parallel()

	transport := &countingTransport{t: t}
	client := NewClient("v23.0")
	client.HTTP = &http.Client{Transport: transport}
	client.AllowedImageHosts = []string{"cdninstagram.com", "fbcdn.net"}

	_, _, err := client.Image(t.Context(), "https://attacker.example.com/x.jpg")
	if err == nil {
		t.Fatal("Image() error = nil, want an error for a disallowed host")
	}
	if transport.calls != 0 {
		t.Errorf("transport called %d times, want 0", transport.calls)
	}
}

// failingTransport fails every request the way an unreachable host or a
// reset connection would. net/http.Client.Do wraps a RoundTripper error
// into a *url.Error carrying the full request URL, access_token query
// string included: exactly what scrub (client.go) exists to strip before
// it reaches a returned error or, from internal/feed/sync.go, a log line.
type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
}

// TestMediaAndRefreshTokenScrubTheAccessTokenOnTransportFailure drives both
// of the only two calls that carry the access token in a URL (Media and
// RefreshToken) through a failing transport and asserts the token appears
// in neither returned error. TokenStore's own leak test (token_test.go)
// covers every path through TokenStore.Token, but RefreshToken there is a
// fake Refresher, never the real Client; this is the one test that
// exercises the real HTTP path and would turn red if scrub were removed or
// broken, which nothing previously did.
func TestMediaAndRefreshTokenScrubTheAccessTokenOnTransportFailure(t *testing.T) {
	t.Parallel()

	const secretToken = "leak-test-access-token-must-not-appear"
	client := NewClient("v23.0")
	client.HTTP = &http.Client{Transport: failingTransport{}}

	if _, err := client.Media(t.Context(), secretToken, 10, 0); err == nil {
		t.Fatal("Media() error = nil, want an error from the failing transport")
	} else {
		secrettest.AssertHidden(t, err, secretToken)
	}

	if _, err := client.RefreshToken(t.Context(), secretToken); err == nil {
		t.Fatal("RefreshToken() error = nil, want an error from the failing transport")
	} else if strings.Contains(err.Error(), secretToken) {
		t.Errorf("RefreshToken() error contains the access token: %q", err.Error())
	}
}

// TestClientRefreshToken exercises RefreshToken's own HTTP behaviour
// against Meta's refresh endpoint: this is its only direct test now that
// Syncer no longer runs a refresh of its own (that moved to TokenStore,
// token.go, which exercises this method only through the Refresher
// interface's fake). fakeMeta and hostTransport are instagram_test.go's.
func TestClientRefreshToken(t *testing.T) {
	t.Parallel()

	meta := newFakeMeta()
	client := NewClient("v23.0")
	client.BaseURL = "https://graph.test"
	client.HTTP = &http.Client{Transport: hostTransport{meta}}

	fresh, err := client.RefreshToken(t.Context(), "tok-1")
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if fresh != "tok-2" {
		t.Errorf("RefreshToken() = %q, want tok-2", fresh)
	}
	if meta.refreshes() != 1 {
		t.Errorf("refresh calls = %d, want 1", meta.refreshes())
	}

	if _, err := client.RefreshToken(t.Context(), "not-a-real-token"); err == nil {
		t.Fatal("RefreshToken() error = nil, want one for a token Meta does not recognise")
	}
}

func TestImageFetchesAllowedHost(t *testing.T) {
	t.Parallel()

	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 100)...)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpeg)
	})

	client := NewClient("v23.0")
	client.HTTP = &http.Client{Transport: hostTransport{handler}}
	client.AllowedImageHosts = []string{"cdninstagram.com"}

	data, contentType, err := client.Image(t.Context(), "https://scontent.cdninstagram.com/a.jpg")
	if err != nil {
		t.Fatalf("Image() error = %v, want the allowed host fetched successfully", err)
	}
	if contentType != "image/jpeg" {
		t.Errorf("contentType = %q, want image/jpeg", contentType)
	}
	if len(data) == 0 {
		t.Error("data is empty")
	}
}
