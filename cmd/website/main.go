// Command website serves ostendorf.tech: the built Vue SPA and prerendered
// HTML for every route and locale on its public listener, and the
// operational endpoints /healthz, /readyz and /metrics on a second,
// administrative listener, reached only from inside the cluster.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ostendorf-cloud/govue/httpserver"
	"github.com/ostendorf-cloud/govue/logging"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/embedfs"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/webapp"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	logger := logging.New(os.Stdout, cfg.LogLevel)

	dist, err := embedfs.Subtree(distFS, "dist")
	if err != nil {
		return fmt.Errorf("website: %w", err)
	}
	prerendered, err := embedfs.Subtree(prerenderedFS, "prerendered")
	if err != nil {
		return fmt.Errorf("website: %w", err)
	}

	app, err := webapp.New(cfg, logger, dist, prerendered)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The feed service sync loop, which stops with ctx. It does not run
	// for a deployment that configured no feed service, and it cannot
	// keep the server from serving.
	app.Start(ctx)

	runErr := httpserver.RunAdmin(ctx, httpserver.Config{
		Addr:    cfg.ListenAddr,
		Handler: app.Handler(),
		Logger:  logger,
		// Failing the readiness probe before the drain delay starts is
		// what lets a load balancer stop sending new requests before the
		// server stops accepting connections at all. This runs once, for
		// both listeners, however many are running.
		BeforeShutdown: app.Health().Drain,
	}, httpserver.AdminConfig{
		Addr:    cfg.AdminListenAddr,
		Handler: app.AdminHandler(),
	})

	// The servers have stopped, so what is still queued for Umami is sent
	// now, within a bound, on a fresh context: ctx has ended already.
	closeCtx, cancel := context.WithTimeout(context.Background(), viewsCloseTimeout)
	defer cancel()
	return errors.Join(runErr, closeResult(logger, app.Close(closeCtx)))
}

// closeResult turns a missed shutdown deadline into a warning: views that
// did not reach Umami in time are lost, which is no failure, and a slow
// Umami must not turn a normal SIGTERM into exit 1. Other errors pass.
func closeResult(logger *slog.Logger, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		logger.Warn("page views still queued at shutdown were not sent", "error", err)
		return nil
	}
	return err
}

// viewsCloseTimeout bounds how long shutdown waits for queued page views.
const viewsCloseTimeout = 5 * time.Second
