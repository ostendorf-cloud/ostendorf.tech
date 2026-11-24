package webapp

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/headertest"
)

// TestAPIHeadersEqualTheFormerImage proves the API sends the security
// headers the single website image sent on its API routes and its 404s,
// with the cache and content type of each answer unchanged.
func TestAPIHeadersEqualTheFormerImage(t *testing.T) {
	app := testApp(t)
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/api/v1/instagram", http.StatusOK},
		{"/api/v1/nope", http.StatusNotFound},
	} {
		rec := get(t, app.Handler(), tc.path, "", nil)
		if rec.Code != tc.status {
			t.Fatalf("GET %s status = %d, want %d", tc.path, rec.Code, tc.status)
		}
		headertest.Equal(t, tc.path, rec.Header())
		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("GET %s Cache-Control = %q, want no-store", tc.path, got)
		}
		if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Errorf("GET %s Content-Type = %q, want JSON", tc.path, got)
		}
	}
}

// TestAPIPlainHTTPDropsOnlyHSTS proves ALLOW_PLAIN_HTTP still removes only
// Strict-Transport-Security, as it did on the single image.
func TestAPIPlainHTTPDropsOnlyHSTS(t *testing.T) {
	app := testAppWith(t, map[string]string{"ALLOW_PLAIN_HTTP": "true", "SITE_URL": "http://localhost:8080"})
	h := get(t, app.Handler(), "/api/v1/instagram", "", nil).Header()
	if h.Get("Strict-Transport-Security") != "" {
		t.Errorf("Strict-Transport-Security = %q, want none", h.Get("Strict-Transport-Security"))
	}
	if h.Get("Content-Security-Policy") == "" || h.Get("X-Frame-Options") != "DENY" {
		t.Error("other security headers are missing")
	}
}
