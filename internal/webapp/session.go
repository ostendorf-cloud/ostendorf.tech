package webapp

import (
	"context"
	"net/http"

	"github.com/ostendorf-cloud/govue/session"
)

// sessionContextKey is unexported, so only this package can set or read the
// value ensureSession puts in the request context.
type sessionContextKey struct{}

// ensureSession ties a request to a session: it reads the one the request
// already carries, or starts a fresh anonymous one and sets its cookie.
//
// This site has no login, so a session here carries no account, nothing
// but an opaque id. Its only purpose is giving the CSRF protector's
// Expected something to derive a token from, the same per-visitor binding
// a logged in application already gets from its own login session. It runs
// on every page request, which is what lets a visitor's browser carry a
// valid token by the time it reaches the contact form's POST route.
//
// A session that cannot be started, the store unreachable, leaves the
// request without one rather than failing it: a visitor reading a page
// must not see an error because a cookie could not be minted, and the CSRF
// protector already treats "no session" as nothing to forge, the same
// answer it gave before this milestone added a mutating route at all.
func (a *App) ensureSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, ok, err := a.sessions.Read(r.Context(), r)
		if err != nil {
			a.logger.ErrorContext(r.Context(), "session: read", "error", err.Error())
		}
		if !ok {
			started, startErr := a.sessions.Start(r.Context(), w, session.Session{})
			if startErr != nil {
				a.logger.ErrorContext(r.Context(), "session: start", "error", startErr.Error())
				next.ServeHTTP(w, r)
				return
			}
			s, ok = started, true
		}

		ctx := r.Context()
		if ok {
			ctx = context.WithValue(ctx, sessionContextKey{}, s)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// sessionFromContext returns the session ensureSession put in the request
// context, and whether there is one.
func sessionFromContext(ctx context.Context) (session.Session, bool) {
	s, ok := ctx.Value(sessionContextKey{}).(session.Session)
	return s, ok
}
