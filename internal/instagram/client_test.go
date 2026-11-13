package instagram

import (
	"net/http"
	"testing"
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
