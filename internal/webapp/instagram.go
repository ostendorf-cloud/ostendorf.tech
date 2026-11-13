package webapp

import (
	"bytes"
	"net/http"
	"time"

	"github.com/ostendorf-cloud/govue/api"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/instagram"
)

// imageMaxAge is how long a browser may keep one post's picture. A post's
// picture never changes once published, and the id is part of the URL, so
// a week costs nothing and spares both this server and Meta the repeat.
const imageMaxAge = "public, max-age=604800"

// instagramPost is one post as the Impressionen section needs it: the id
// the picture is fetched by, its alt text and the post's own page on
// Instagram. The picture itself is never linked to Meta's CDN here, only
// to this service's own /api/v1/instagram/{id}, which is what keeps the
// privacy policy's claim true that a visitor's browser opens no connection
// to Instagram or Meta at all.
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
// an access token ever gives: the frontend then renders no Impressionen
// section at all. A feed that is empty because the first sync has not
// finished, or because Meta is down, answers exactly the same way, so
// neither can ever turn into an error a visitor sees.
func (a *App) handleInstagramFeed(w http.ResponseWriter, _ *http.Request) error {
	posts := a.instagramFeed.Posts()
	out := make([]instagramPost, 0, len(posts))
	for _, p := range posts {
		out = append(out, instagramPost{ID: p.ID, Alt: p.Alt, Permalink: p.Permalink})
	}
	return api.Write(w, http.StatusOK, instagramFeedResponse{Posts: out})
}

// handleInstagramImage serves one post's picture from memory, the copy
// this server downloaded from Meta itself. It writes raw image bytes
// rather than JSON, which is why it is a plain handler instead of one
// api.Handle adapts; a refusal still uses the one API error shape.
func (a *App) handleInstagramImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !instagram.ValidID(id) {
		api.WriteError(w, r, api.NotFound("unknown post"))
		return
	}
	post, ok := a.instagramFeed.Post(id)
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
