package webapp

import (
	"bytes"
	"net/http"
	"time"

	"github.com/ostendorf-cloud/govue/api"
)

// imageMaxAge is how long a browser may keep one post's picture. A post's
// picture never changes once published, and the id is part of the URL, so
// a week costs nothing and spares both this server and the feed service
// the repeat.
const imageMaxAge = "public, max-age=604800"

// instagramPost is one post as the Impressionen section needs it: the id
// the picture is fetched by, its alt text and the post's own page on
// Instagram. The picture itself is never linked to Meta's CDN here, or to
// the feed service, only to this service's own /api/v1/instagram/{id},
// which is what keeps the privacy policy's claim true that a visitor's
// browser opens no connection to the feed service, Instagram or Meta at
// all.
type instagramPost struct {
	ID        string `json:"id"`
	Alt       string `json:"alt"`
	Permalink string `json:"permalink"`
}

// instagramFeedResponse is what GET /api/v1/instagram answers.
type instagramFeedResponse struct {
	Posts []instagramPost `json:"posts"`
}

// handleInstagramFeed lists the posts currently held in memory. An empty
// list is a perfectly good answer, and the only one a deployment without
// a configured feed service ever gives: the frontend then renders no
// Impressionen section at all. A cache that is empty because the first
// sync has not finished, or because the feed service is unreachable,
// answers exactly the same way, so neither can ever turn into an error a
// visitor sees.
func (a *App) handleInstagramFeed(w http.ResponseWriter, _ *http.Request) error {
	posts := a.feedCache.Posts()
	out := make([]instagramPost, 0, len(posts))
	for _, p := range posts {
		out = append(out, instagramPost{ID: p.ID, Alt: p.Alt, Permalink: p.Permalink})
	}
	return api.Write(w, http.StatusOK, instagramFeedResponse{Posts: out})
}

// handleInstagramImage serves one post's picture from memory, the copy
// this server downloaded from the feed service. It writes raw image
// bytes rather than JSON, which is why it is a plain handler instead of
// one api.Handle adapts; a refusal still uses the one API error shape.
//
// It takes no dependency on internal/instagram, including its ValidID
// helper, for id validation: a feedCache lookup already answers "not
// found" for any id that is not an actual post's, which is exactly the
// same outcome a pre-check would have produced, and this package has no
// other reason to import that package at all.
func (a *App) handleInstagramImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	post, ok := a.feedCache.Post(id)
	if !ok {
		api.WriteError(w, r, api.NotFound("unknown post"))
		return
	}
	h := w.Header()
	h.Set("Content-Type", post.ContentType)
	h.Set("Cache-Control", imageMaxAge)
	h.Set("ETag", post.ETag)
	// A zero modtime leaves out Last-Modified; the ETag is what a
	// conditional request is answered from.
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(post.Image))
}
