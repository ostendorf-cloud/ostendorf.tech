// Command website serves ostendorf.tech: the built Vue SPA, prerendered
// HTML for every route and locale, and the operational endpoints
// /healthz, /readyz and /metrics.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"syscall"

	"github.com/ostendorf-cloud/govue/httpserver"
	"github.com/ostendorf-cloud/govue/logging"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/config"
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

	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		return fmt.Errorf("website: %w", err)
	}
	prerendered, err := fs.Sub(prerenderedFS, "prerendered")
	if err != nil {
		return fmt.Errorf("website: %w", err)
	}

	app, err := webapp.New(cfg, logger, dist, prerendered)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The Instagram feed's sync loop and the Umami reporter, both of which
	// stop with ctx. Neither runs for a deployment that configured no
	// token and no Umami, and neither can keep the server from serving.
	app.Start(ctx)

	return httpserver.Run(ctx, httpserver.Config{
		Addr:    cfg.ListenAddr,
		Handler: app.Handler(),
		Logger:  logger,
		// Failing the readiness probe before the drain delay starts is
		// what lets a load balancer stop sending new requests before the
		// server stops accepting connections at all.
		BeforeShutdown: app.Health().Drain,
	})
}
