package webapp

import (
	"fmt"
	"io/fs"
	"net/http"

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

// checkPublicFiles fails when dist lacks one of publicFiles, so a build
// without the icons is reported once at startup rather than as a 404.
func checkPublicFiles(dist fs.FS) error {
	for p := range publicFiles {
		info, err := fs.Stat(dist, p[1:])
		if err != nil || info.IsDir() || info.Size() == 0 {
			return fmt.Errorf("webapp: dist has no usable %s, run task favicon:generate and task web:build", p[1:])
		}
	}
	return nil
}

// publicFileHandler serves one of publicFiles. The names carry no hash,
// so the cache lifetime is a day rather than immutable.
func (p *pageServer) publicFileHandler(urlPath string) http.Handler {
	contentType := publicFiles[urlPath]
	name := urlPath[1:]
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(p.public, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(data)
	})
}
