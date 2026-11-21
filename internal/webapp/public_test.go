package webapp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// testDist is a minimal dist: one asset plus every public file the server
// refuses to start without.
func testDist() fstest.MapFS {
	return fstest.MapFS{
		"assets/app.js":        &fstest.MapFile{Data: []byte("console.log('app')")},
		"favicon.svg":          &fstest.MapFile{Data: []byte("<svg></svg>")},
		"favicon-32x32.png":    &fstest.MapFile{Data: []byte("png32")},
		"apple-touch-icon.png": &fstest.MapFile{Data: []byte("png180")},
		"og-image.png":         &fstest.MapFile{Data: []byte("png1200")},
	}
}

func TestPublicFilesAreServedAtTheRoot(t *testing.T) {
	mux := http.NewServeMux()
	p, err := newPageServer(testDist(), fstest.MapFS{})
	if err != nil {
		t.Fatalf("newPageServer() error = %v", err)
	}
	for path := range publicFiles {
		mux.Handle("GET "+path, p.publicFileHandler(path))
	}

	tests := map[string]string{
		"/favicon.svg":          "image/svg+xml",
		"/favicon-32x32.png":    "image/png",
		"/apple-touch-icon.png": "image/png",
		"/og-image.png":         "image/png",
	}
	for path, wantType := range tests {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Content-Type"); got != wantType {
				t.Errorf("Content-Type = %q, want %q", got, wantType)
			}
			if got := rec.Header().Get("Cache-Control"); got != "public, max-age=86400" {
				t.Errorf("Cache-Control = %q, want one day", got)
			}
			if rec.Body.Len() == 0 {
				t.Error("body is empty")
			}
		})
	}
}

// TestAppServesThePublicFiles goes through the real handler chain and mux
// New builds, so a registration dropped from buildHandler is caught, which
// the test above with its own mux cannot do.
func TestAppServesThePublicFiles(t *testing.T) {
	handler := testApp(t).Handler()
	dist := testDist()

	tests := map[string]string{
		"/favicon.svg":          "image/svg+xml",
		"/favicon-32x32.png":    "image/png",
		"/apple-touch-icon.png": "image/png",
		"/og-image.png":         "image/png",
	}
	for path, wantType := range tests {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("GET status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Content-Type"); got != wantType {
				t.Errorf("Content-Type = %q, want %q", got, wantType)
			}
			if got := rec.Header().Get("Cache-Control"); got != "public, max-age=86400" {
				t.Errorf("Cache-Control = %q, want one day", got)
			}
			if want := dist[path[1:]].Data; !bytes.Equal(rec.Body.Bytes(), want) {
				t.Errorf("body = %q, want %q", rec.Body.Bytes(), want)
			}

			head := httptest.NewRequestWithContext(t.Context(), http.MethodHead, path, nil)
			hrec := httptest.NewRecorder()
			handler.ServeHTTP(hrec, head)
			if hrec.Code != http.StatusOK {
				t.Errorf("HEAD status = %d, want %d", hrec.Code, http.StatusOK)
			}

			post := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, nil)
			prec := httptest.NewRecorder()
			handler.ServeHTTP(prec, post)
			if prec.Code != http.StatusMethodNotAllowed {
				t.Errorf("POST status = %d, want %d", prec.Code, http.StatusMethodNotAllowed)
			}
		})
	}

	t.Run("trailing slash", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/favicon.svg/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}

func TestNewPageServerFailsWithoutAPublicFile(t *testing.T) {
	for path := range publicFiles {
		t.Run(path, func(t *testing.T) {
			dist := testDist()
			delete(dist, path[1:])
			if _, err := newPageServer(dist, fstest.MapFS{}); err == nil {
				t.Fatalf("newPageServer() error = nil, want an error naming %s", path)
			}
		})
	}
}
