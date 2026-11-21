package webapp

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/ostendorf-cloud/govue/favicon"
)

// socialImageFile is the 1200 by 630 preview og:image and twitter:image
// point at, written next to the favicon files by "task favicon:generate".
const socialImageFile = "og-image.png"

// publicFiles are the files vite copies from web/public to the root of
// dist and the server answers at the root, under the names govue's favicon
// package fixes plus the social preview image. Fixed names, not a lookup
// of the request path, so nothing else in dist is ever served from the root.
var publicFiles = map[string]string{
	"/" + favicon.SVGFile:   "image/svg+xml",
	"/" + favicon.PNGFile:   "image/png",
	"/" + favicon.TouchFile: "image/png",
	"/" + socialImageFile:   "image/png",
}

// publicFile is one public file read once at startup, with its strong
// ETag, the quoted SHA-256 of the bytes.
type publicFile struct {
	data []byte
	etag string
}

// loadPublicFiles reads every one of publicFiles from dist. It fails when
// one is missing, a directory or empty, so a build without the icons is
// reported once at startup rather than as a 404.
func loadPublicFiles(dist fs.FS) (map[string]publicFile, error) {
	files := make(map[string]publicFile, len(publicFiles))
	for p := range publicFiles {
		info, err := fs.Stat(dist, p[1:])
		if err != nil || info.IsDir() || info.Size() == 0 {
			return nil, fmt.Errorf("webapp: dist has no usable %s, run task favicon:generate and task web:build", p[1:])
		}
		data, err := fs.ReadFile(dist, p[1:])
		if err != nil {
			return nil, fmt.Errorf("webapp: read %s from dist: %w", p[1:], err)
		}
		sum := sha256.Sum256(data)
		files[p] = publicFile{data: data, etag: `"` + hex.EncodeToString(sum[:]) + `"`}
	}
	return files, nil
}

// publicFileHandler serves one of publicFiles. The names carry no hash,
// so the cache lifetime is a day rather than immutable, and a client
// revalidates with the ETag afterwards. http.ServeContent answers a
// matching If-None-Match with 304 and a HEAD without a body. The zero
// modification time keeps it from sending a Last-Modified header, since
// an embedded file has no meaningful one.
func (p *pageServer) publicFileHandler(urlPath string) http.Handler {
	contentType := publicFiles[urlPath]
	file := p.public[urlPath]
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("ETag", file.etag)
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(file.data))
	})
}
