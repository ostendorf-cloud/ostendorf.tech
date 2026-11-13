package feed

import (
	"bytes"
	"net/http"
	"time"

	"github.com/ostendorf-cloud/govue/api"
	"github.com/ostendorf-cloud/govue/middleware"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/instagram"
)

// imageMaxAge is how long the website may cache one post's picture. A
// post's picture never changes once published, and the id is part of the
// URL, so a week costs nothing and spares this server, OpenBao and Meta
// the repeat; see internal/webapp/instagram.go, which applies the same
// value to the browser-facing equivalent of this handler.
const imageMaxAge = "public, max-age=604800"

// Routes are named generically, GET /api/v1/posts and
// GET /api/v1/posts/{id}/image, not "instagram": the plan deliberately
// named the binary and pod "feed", not "instagram", so a second source
// later needs no rename. This HTTP surface is the pod-to-pod equivalent of
// internal/webapp/instagram.go's browser-facing one, and it stays equally
// source-agnostic.
//
// Neither route authenticates its caller. This listener is ClusterIP-only
// with a NetworkPolicy planned (a later step) admitting only the website
// pod: a network-layer trust boundary, not an application-layer one. What
// is served here, captions, alt text and picture bytes, is public
// marketing content with no confidentiality requirement even if briefly
// reachable inside the cluster. Adding a shared token or mTLS on top would
// be real operational cost, distribution and rotation, for no
// corresponding risk reduction, on a binary whose one actual secret, the
// Instagram access token, never leaves OpenBao and this process's own
// memory. Ordinary hygiene still applies regardless of that trust
// boundary: the id path parameter is validated the same way the
// public-facing handler validates it, and middleware.Recover/
// middleware.RequestID stay in the chain.

// post is one post as the website needs it: the id its picture is fetched
// by, its alt text and the post's own page on Instagram. Mirrors
// internal/webapp/instagram.go's instagramPost exactly, field order
// included, since the website decodes this response the same way.
type post struct {
	ID        string `json:"id"`
	Alt       string `json:"alt"`
	Permalink string `json:"permalink"`
}

// postsResponse is what GET /api/v1/posts answers.
type postsResponse struct {
	Posts []post `json:"posts"`
}

// handlePosts lists the posts currently held in memory. An empty list is a
// perfectly good answer: a feed that has not synced yet, or an account
// with nothing to show, answers exactly the same way, so neither ever
// turns into an error the website sees.
func (a *App) handlePosts(w http.ResponseWriter, _ *http.Request) error {
	posts := a.feed.Posts()
	out := make([]post, 0, len(posts))
	for _, p := range posts {
		out = append(out, post{ID: p.ID, Alt: p.Alt, Permalink: p.Permalink})
	}
	return api.Write(w, http.StatusOK, postsResponse{Posts: out})
}

// handleImage serves one post's picture from memory, the copy this
// service downloaded from Meta itself. It writes raw image bytes rather
// than JSON, which is why it is a plain handler instead of one api.Handle
// adapts; a refusal still uses the one API error shape.
func (a *App) handleImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !instagram.ValidID(id) {
		api.WriteError(w, r, api.NotFound("unknown post"))
		return
	}
	p, ok := a.feed.Post(id)
	if !ok {
		api.WriteError(w, r, api.NotFound("unknown post"))
		return
	}
	h := w.Header()
	h.Set("Content-Type", p.ContentType)
	h.Set("Cache-Control", imageMaxAge)
	h.Set("ETag", p.ETag)
	// A zero modtime leaves out Last-Modified; the ETag is what a
	// conditional request is answered from.
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(p.Image))
}

// buildHandler assembles the whole HTTP surface: the health endpoints, the
// two post routes, a JSON 404 for everything else, panic recovery and a
// request id on every response.
func (a *App) buildHandler() http.Handler {
	mux := http.NewServeMux()
	a.health.Handle(mux)
	mux.Handle("GET /api/v1/posts", api.Handle(a.handlePosts))
	mux.Handle("GET /api/v1/posts/{id}/image", http.HandlerFunc(a.handleImage))
	mux.Handle("/", api.NotFoundHandler())
	return middleware.Chain(mux, middleware.Recover(a.logger), middleware.RequestID(a.logger))
}
