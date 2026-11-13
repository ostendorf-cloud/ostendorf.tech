package instagram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/openbao"
)

// accessTokenKey and refreshedAtKey name the two fields
// old_simon-ostendorf.de wrote into the secret. TokenStore reads and writes
// the same two keys, at the same path, so a secret an operator wrote in by
// hand keeps working unchanged.
const (
	accessTokenKey = "access_token"
	refreshedAtKey = "refreshed_at"
)

// Store reads and writes the KV v2 secret that holds the access token;
// *openbao.Client is one.
type Store interface {
	Read(ctx context.Context, path string) (openbao.Secret, error)
	Write(ctx context.Context, path string, values map[string]any, cas int) error
}

// Refresher exchanges a long lived Instagram access token for a new one.
// *Client already implements this against the real Graph API, so TokenStore
// takes it as a dependency instead of a second copy of the refresh call.
type Refresher interface {
	RefreshToken(ctx context.Context, token string) (string, error)
}

// TokenStore owns the Instagram access token's lifecycle in OpenBao: it
// reads the stored token, refreshes it against Meta once it is older than
// RefreshAge, and writes the refresh back as a full-document overwrite
// guarded by the version it just read, so two replicas refreshing at the
// same time cannot silently clobber each other.
//
// OpenBao is the only source: the initial token is written in by hand once,
// and there is no environment variable fallback. A read failure, including
// openbao.ErrNotFound when nothing was ever written, is returned to the
// caller unchanged, because there is no other token to fall back to.
//
// A refresh failure is not fatal: the current token stays valid for weeks
// after it is due for a refresh, so it is returned as is and the next cycle
// tries again. A write failure after a successful refresh is handled the
// same way: the freshly fetched token is used for this cycle, unstored, and
// the next cycle re-reads and, if still due, refreshes again. A CAS
// mismatch means another replica's refresh already won; this one re-reads
// and uses whichever token is now stored, rather than treating that as an
// error.
type TokenStore struct {
	// Store is the secret backend; *openbao.Client in production.
	Store Store
	// Refresher calls Meta's refresh endpoint; *Client in production.
	Refresher Refresher
	// Path is the KV v2 data path the secret lives at.
	Path string
	// RefreshAge is how old refreshed_at may get before a refresh is tried.
	RefreshAge time.Duration
	// Logger receives the lifecycle events below. Never the token itself.
	Logger *slog.Logger
	// Now is the clock Token reads, injected so the age check needs no sleep in a test.
	Now func() time.Time
}

// Token returns the token to call the Graph API with, refreshing it first
// when refreshed_at is missing, unparsable, or older than RefreshAge. See
// the type doc for what a read, refresh, write or CAS failure each do.
func (s *TokenStore) Token(ctx context.Context) (string, error) {
	secret, err := s.Store.Read(ctx, s.Path)
	if err != nil {
		return "", fmt.Errorf("instagram token: %w", err)
	}
	token := secret.String(accessTokenKey)
	if token == "" {
		return "", errors.New("instagram token: access_token missing in OpenBao")
	}

	// A missing or unparsable refreshed_at cannot be assumed fresh, so it
	// falls through to a refresh attempt the same as a stale one would.
	now := s.Now()
	refreshedAt, err := time.Parse(time.RFC3339, secret.String(refreshedAtKey))
	if err == nil && now.Sub(refreshedAt) < s.RefreshAge {
		return token, nil
	}

	fresh, err := s.Refresher.RefreshToken(ctx, token)
	if err != nil {
		// Meta's 60 day window leaves weeks of slack; try again next cycle.
		s.Logger.Warn("instagram token refresh failed, using the current token", "error", err.Error())
		return token, nil
	}

	values := make(map[string]any, len(secret.Data))
	for k, v := range secret.Data {
		values[k] = v
	}
	values[accessTokenKey] = fresh
	values[refreshedAtKey] = now.UTC().Format(time.RFC3339)

	switch writeErr := s.Store.Write(ctx, s.Path, values, secret.Version); {
	case writeErr == nil:
		s.Logger.Info("instagram token refreshed")
		return fresh, nil
	case errors.Is(writeErr, openbao.ErrCASMismatch):
		return s.tokenAfterCASMismatch(ctx, fresh)
	default:
		s.Logger.Warn("instagram token refreshed but not stored, using it for this cycle", "error", writeErr.Error())
		return fresh, nil
	}
}

// tokenAfterCASMismatch handles another replica having refreshed and
// written first: that write won, so this one re-reads and uses whichever
// token is now stored instead of treating the mismatch as an error.
func (s *TokenStore) tokenAfterCASMismatch(ctx context.Context, fresh string) (string, error) {
	s.Logger.Info("instagram token refreshed by another replica")
	again, err := s.Store.Read(ctx, s.Path)
	if err == nil {
		if token := again.String(accessTokenKey); token != "" {
			return token, nil
		}
		err = errors.New("access_token missing")
	}
	// The fresh token fetched here is valid too, just not stored; the next
	// cycle re-reads and, if still due, refreshes again.
	s.Logger.Warn("instagram token of the other replica unreadable, using the unstored fresh token", "error", err.Error())
	return fresh, nil
}
