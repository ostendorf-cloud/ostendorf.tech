// Command fakefeed is a development-only stand-in for cmd/feed. It answers the
// same routes with the same shapes, GET /api/v1/posts, GET /api/v1/posts/{id}/image,
// /healthz and /readyz, so the website's internal/feedclient accepts it unchanged and
// the Impressionen section can be seen locally without OpenBao or a Meta token.
//
// Nothing here is real. The pictures are drawn at startup with Go's image packages
// (coloured gradients and one shape each), the captions say plainly that they are
// samples, and no Instagram content or person is involved. The permalinks only have the
// shape the real feed guarantees, an https://www.instagram.com/ prefix; they point at
// no real post.
//
// This binary is never part of the website or feed image: the Dockerfile builds it as
// its own "fakefeed" target, and no workflow publishes that target.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	// defaultListenAddr is 9081, not the feed service's own 8081: 8081 is the
	// website's admin listener, and 9081 is the port .env.example already uses for a
	// feed service on the same machine.
	defaultListenAddr = ":9081"
	// defaultPostCount is the size of the sample account: enough posts for several
	// "load more" clicks, few enough to draw in a moment at startup. The real feed
	// service reads POST_COUNT as a cap on a real account, with 0 for its maximum; here
	// it is the number of samples, and 0 or empty means the whole sample account.
	// maxPostCount keeps the startup drawing short.
	defaultPostCount = 40
	minPostCount     = 0
	maxPostCount     = 200

	// imageSize is the edge of the square pictures, the shape of an Instagram feed
	// post.
	imageSize = 640

	// imageMaxAge matches internal/feed's own value for the same header.
	imageMaxAge = "public, max-age=604800"
)

// fixture is one sample post with its generated picture.
type fixture struct {
	ID          string
	Alt         string
	Permalink   string
	Image       []byte
	ContentType string
	ETag        string
}

// postJSON mirrors internal/feed's post type and feedclient.Post.
type postJSON struct {
	ID        string `json:"id"`
	Alt       string `json:"alt"`
	Permalink string `json:"permalink"`
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = defaultListenAddr
	}
	count, err := parsePostCount(os.Getenv("POST_COUNT"))
	if err != nil {
		return err
	}
	fixtures, err := buildFixtures(count)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:              addr,
		Handler:           newHandler(fixtures),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Info("fake feed listening, sample posts only, development use", "addr", addr, "posts", len(fixtures))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("fakefeed: %w", err)
	}
	return nil
}

// parsePostCount reads POST_COUNT as the number of sample posts: empty and 0 are
// the whole sample account, defaultPostCount, anything else must be an integer from
// 1 to maxPostCount.
func parsePostCount(raw string) (int, error) {
	if raw == "" {
		return defaultPostCount, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < minPostCount || n > maxPostCount {
		return 0, fmt.Errorf("fakefeed: POST_COUNT %q: want an integer from %d to %d", raw, minPostCount, maxPostCount)
	}
	if n == 0 {
		return defaultPostCount, nil
	}
	return n, nil
}

// buildFixtures draws n sample posts, newest first. Odd numbered posts are JPEG and
// even numbered ones PNG, so both formats the real feed serves are exercised.
func buildFixtures(n int) ([]fixture, error) {
	out := make([]fixture, 0, n)
	for i := 1; i <= n; i++ {
		img := drawPicture(i, n)
		var buf bytes.Buffer
		var contentType string
		var err error
		if i%2 == 1 {
			contentType = "image/jpeg"
			err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
		} else {
			contentType = "image/png"
			err = png.Encode(&buf, img)
		}
		if err != nil {
			return nil, fmt.Errorf("fakefeed: encode sample post %d: %w", i, err)
		}
		sum := sha256.Sum256(buf.Bytes())
		id := fmt.Sprintf("sample%02d", i)
		out = append(out, fixture{
			ID:          id,
			Alt:         fmt.Sprintf("Sample post %d: fixture for local testing", i),
			Permalink:   "https://www.instagram.com/p/" + id + "/",
			Image:       buf.Bytes(),
			ContentType: contentType,
			ETag:        `"` + hex.EncodeToString(sum[:8]) + `"`,
		})
	}
	return out, nil
}

// drawPicture draws post i of n: a diagonal gradient whose hue depends on i, and one
// disc or one stripe band on top, so neighbouring posts differ at a glance.
func drawPicture(i, n int) *image.RGBA {
	hue := float64(i-1) / float64(n) * 360
	from := hsv(hue, 0.55, 0.95)
	to := hsv(hue+40, 0.85, 0.45)
	accent := hsv(hue+180, 0.35, 1)

	img := image.NewRGBA(image.Rect(0, 0, imageSize, imageSize))
	centre := float64(imageSize) / 2
	radius := float64(imageSize) / 4
	for y := range imageSize {
		for x := range imageSize {
			t := float64(x+y) / float64(2*imageSize-2)
			c := mix(from, to, t)
			if i%2 == 1 {
				dx, dy := float64(x)-centre, float64(y)-centre
				if dx*dx+dy*dy <= radius*radius {
					c = accent
				}
			} else if y > imageSize/3 && y < imageSize/3+imageSize/6 {
				c = accent
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// mix blends a towards b by t, from 0 to 1.
func mix(a, b color.RGBA, t float64) color.RGBA {
	lerp := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{R: lerp(a.R, b.R), G: lerp(a.G, b.G), B: lerp(a.B, b.B), A: 255}
}

// hsv converts a hue in degrees, a saturation and a value to an opaque colour.
func hsv(h, s, v float64) color.RGBA {
	h -= 360 * float64(int(h/360))
	c := v * s
	hp := h / 60
	x := c * (1 - abs(float64(int(hp)%2)+(hp-float64(int(hp)))-1))
	var r, g, b float64
	switch int(hp) {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	m := v - c
	to8 := func(f float64) uint8 { return uint8((f + m) * 255) }
	return color.RGBA{R: to8(r), G: to8(g), B: to8(b), A: 255}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// newHandler serves the fixtures on the real feed service's routes.
func newHandler(fixtures []fixture) http.Handler {
	byID := make(map[string]*fixture, len(fixtures))
	posts := make([]postJSON, 0, len(fixtures))
	for i := range fixtures {
		f := &fixtures[i]
		byID[f.ID] = f
		posts = append(posts, postJSON{ID: f.ID, Alt: f.Alt, Permalink: f.Permalink})
	}

	mux := http.NewServeMux()
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	mux.HandleFunc("GET /healthz", ok)
	mux.HandleFunc("GET /readyz", ok)
	mux.HandleFunc("GET /api/v1/posts", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, struct {
			Posts []postJSON `json:"posts"`
		}{Posts: posts})
	})
	mux.HandleFunc("GET /api/v1/posts/{id}/image", func(w http.ResponseWriter, r *http.Request) {
		f, found := byID[r.PathValue("id")]
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown post"})
			return
		}
		h := w.Header()
		h.Set("Content-Type", f.ContentType)
		h.Set("Cache-Control", imageMaxAge)
		h.Set("ETag", f.ETag)
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(f.Image))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
