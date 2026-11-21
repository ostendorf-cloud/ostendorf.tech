package webapp

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

// TestPublicFilesAnswerConditionalRequests checks the strong ETag and the
// 304 a client gets when it revalidates, through the real handler.
func TestPublicFilesAnswerConditionalRequests(t *testing.T) {
	handler := testApp(t).Handler()
	get := func(t *testing.T, path, ifNoneMatch string, method string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	for path := range publicFiles {
		t.Run(path, func(t *testing.T) {
			first := get(t, path, "", http.MethodGet)
			etag := first.Header().Get("ETag")
			sum := sha256.Sum256(testDist()[path[1:]].Data)
			if want := `"` + hex.EncodeToString(sum[:]) + `"`; etag != want {
				t.Fatalf("ETag = %q, want the quoted strong SHA-256 %q", etag, want)
			}
			if second := get(t, path, "", http.MethodGet).Header().Get("ETag"); second != etag {
				t.Errorf("ETag changed between requests: %q then %q", etag, second)
			}
			if got := first.Header().Get("Last-Modified"); got != "" {
				t.Errorf("Last-Modified = %q, want none", got)
			}

			hit := get(t, path, etag, http.MethodGet)
			if hit.Code != http.StatusNotModified {
				t.Errorf("matching If-None-Match status = %d, want %d", hit.Code, http.StatusNotModified)
			}
			if hit.Body.Len() != 0 {
				t.Errorf("304 body = %q, want empty", hit.Body.Bytes())
			}
			if got := hit.Header().Get("Cache-Control"); got != "public, max-age=86400" {
				t.Errorf("304 Cache-Control = %q, want one day", got)
			}

			miss := get(t, path, `"other"`, http.MethodGet)
			if miss.Code != http.StatusOK {
				t.Errorf("wrong If-None-Match status = %d, want %d", miss.Code, http.StatusOK)
			}
			if miss.Body.Len() == 0 {
				t.Error("wrong If-None-Match answered no body")
			}

			head := get(t, path, "", http.MethodHead)
			if head.Code != http.StatusOK || head.Body.Len() != 0 {
				t.Errorf("HEAD = %d with %d body bytes, want 200 and none", head.Code, head.Body.Len())
			}
			if got := head.Header().Get("Content-Type"); got != publicFiles[path] {
				t.Errorf("HEAD Content-Type = %q, want %q", got, publicFiles[path])
			}
		})
	}
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
