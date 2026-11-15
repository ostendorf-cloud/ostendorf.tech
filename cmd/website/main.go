// Command website serves ostendorf.tech: the built Vue SPA and prerendered
// HTML for every route and locale on its public listener, and the
// operational endpoints /healthz, /readyz and /metrics on a second,
// administrative listener, reached only from inside the cluster.
package main

import (
	"context"
	"fmt"
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

	dist, err := embeddedSubtree(distFS, "dist")
	if err != nil {
		return fmt.Errorf("website: %w", err)
	}
	prerendered, err := embeddedSubtree(prerenderedFS, "prerendered")
	if err != nil {
		return fmt.Errorf("website: %w", err)
	}

	app, err := webapp.New(cfg, logger, dist, prerendered)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The feed service sync loop and the Umami reporter, both of which
	// stop with ctx. Neither runs for a deployment that configured no
	// feed service and no Umami, and neither can keep the server from
	// serving.
	app.Start(ctx)

	return httpserver.RunAdmin(ctx, httpserver.Config{
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
}
