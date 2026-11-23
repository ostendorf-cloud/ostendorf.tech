// Package feed builds and runs the feed service: the background loop that
// keeps an instagram.Feed loaded from Meta using a store-backed access
// token, and the internal HTTP handlers that serve the cached posts and
// image bytes to the website.
//
// This package never logs or returns the access token in any error. Every
// path that touches a token goes through instagram.TokenStore (via
// instagram.Syncer.Tokens) and internal/openbao, both of which already
// keep the token out of logs and errors; nothing in this package reads a
// token itself or carries one in a variable, a log line, or an error it
// builds, the same way internal/instagram/token.go's own doc comment
// states and internal/openbao/client.go's do.
package feed

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ostendorf-cloud/govue/health"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/feedconfig"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/instagram"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/openbao"
)

// readinessCheckTimeout bounds the registered health check.
const readinessCheckTimeout = time.Second

// App is the feed service: the background sync loop and the HTTP handlers
// that serve its result.
type App struct {
	logger *slog.Logger
	health *health.Registry

	feed     *instagram.Feed
	syncer   *instagram.Syncer
	interval time.Duration

	// ready is true once one sync cycle has completed without error: a
	// usable token was obtained and Meta answered. It is sticky — once
	// ready, a later transient failure keeps serving the last good feed
	// (Syncer.Sync's own behavior) rather than flapping readiness back to
	// false, the same "last good feed stays" principle Syncer.Run already
	// applies to the feed contents themselves.
	ready atomic.Bool

	handler http.Handler
}

// New builds the App from cfg and logger: the OpenBao client, the
// store-backed token source on top of it, the Graph API client, the feed
// cache, and the syncer that ties them together. Nothing here performs any
// IO; the OpenBao login and the first Meta call both happen lazily, on the
// first call Start's background loop makes.
func New(cfg feedconfig.Config, logger *slog.Logger) *App {
	bao := openbao.New(cfg.OpenBaoAddr.String(), login(cfg), logger)
	client := instagram.NewClient(cfg.APIVersion)
	f := &instagram.Feed{}
	store := &instagram.TokenStore{
		Store:          bao,
		Refresher:      client,
		Path:           cfg.OpenBaoSecretPath,
		AccessTokenKey: cfg.SecretKeyAccessToken,
		RefreshedAtKey: cfg.SecretKeyRefreshedAt,
		RefreshAge:     cfg.TokenRefreshAge,
		Logger:         logger,
		Now:            time.Now,
	}
	syncer := &instagram.Syncer{
		Client:     client,
		Tokens:     store,
		Count:      cfg.PostCount,
		PerRequest: cfg.PostsPerRequest,
		Feed:       f,
		Logger:     logger,
	}

	a := &App{logger: logger, feed: f, syncer: syncer, interval: cfg.SyncInterval, health: health.New()}
	a.health.Add("feed", readinessCheckTimeout, a.checkReady)
	a.handler = a.buildHandler()
	return a
}

// login is the OpenBao auth method cfg names. feedconfig.Load accepts no
// other value than the two below, so anything else is the Kubernetes
// default, the method for a cluster.
func login(cfg feedconfig.Config) openbao.Login {
	if cfg.OpenBaoAuthMethod == feedconfig.AuthMethodToken {
		return openbao.StaticToken{Token: cfg.OpenBaoToken}
	}
	return openbao.Kubernetes{AuthPath: cfg.OpenBaoAuthPath, Role: cfg.OpenBaoRole, JWTFile: cfg.OpenBaoJWTFile}
}

// checkReady is the registered health check: a pure in-memory read, never a
// network call of its own, so polling /readyz costs nothing against
// OpenBao or Meta no matter how often a kubelet asks.
func (a *App) checkReady(context.Context) error {
	if !a.ready.Load() {
		return errors.New("no usable token and feed yet")
	}
	return nil
}

// Health is the app's readiness and liveness registry.
func (a *App) Health() *health.Registry { return a.health }

// Handler is the app's whole HTTP surface.
func (a *App) Handler() http.Handler { return a.handler }

// Start launches the background sync loop; it returns immediately. The
// loop stops when ctx is done.
func (a *App) Start(ctx context.Context) {
	go a.run(ctx)
}
