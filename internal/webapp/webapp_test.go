package webapp

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
)

// discardLogger is a logger that writes nowhere, for tests that care about
// behaviour, not log output.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func testApp(t *testing.T) *App {
	t.Helper()
	cfg := config.Config{
		ListenAddr:     ":0",
		RateLimitBurst: 100,
		RateLimitPer:   time.Second,
	}
	dist := fstest.MapFS{
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('app')")},
	}
	prerendered := fstest.MapFS{
		"de/index.html": &fstest.MapFile{Data: []byte("<html><body>Startseite</body></html>")},
		"de/404.html":   &fstest.MapFile{Data: []byte("<html><body>Nicht gefunden</body></html>")},
		"en/index.html": &fstest.MapFile{Data: []byte("<html><body>Home</body></html>")},
		"en/404.html":   &fstest.MapFile{Data: []byte("<html><body>Not found</body></html>")},
	}
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return app
}

func TestAppServesLivenessAndReadiness(t *testing.T) {
	app := testApp(t)
	handler := app.Handler()

	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, rec.Code, http.StatusOK)
		}
	}
}

func TestAppDrainingFailsReadiness(t *testing.T) {
	app := testApp(t)
	app.Health().Drain()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestAppServesMetrics(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAppServesPrerenderedHome(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAppRateLimitsExcessRequests(t *testing.T) {
	cfg := config.Config{RateLimitBurst: 1, RateLimitPer: time.Minute}
	dist := fstest.MapFS{"assets/app.js": &fstest.MapFile{Data: []byte("x")}}
	prerendered := fstest.MapFS{
		"de/index.html": &fstest.MapFile{Data: []byte("<html></html>")},
		"de/404.html":   &fstest.MapFile{Data: []byte("<html></html>")},
	}
	app, err := New(cfg, discardLogger(), dist, prerendered)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	for i, wantStatus := range []int{http.StatusOK, http.StatusTooManyRequests} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/", nil)
		req.RemoteAddr = "203.0.113.1:1234"
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != wantStatus {
			t.Errorf("request %d status = %d, want %d", i, rec.Code, wantStatus)
		}
	}
}

func TestAppUnknownRouteServesNotFound(t *testing.T) {
	app := testApp(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/de/does-not-exist/", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
