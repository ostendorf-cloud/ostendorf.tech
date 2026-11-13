// Command feed owns the Instagram access token through OpenBao, keeps a
// cache of the business account's latest posts synced from Meta, and
// serves that cache, plus /healthz and /readyz, to the website over an
// internal-only listener. It never serves a token, and it never falls back
// to one: OpenBao is the only source, and without a usable token and feed
// the process runs and reports unready rather than crash-looping or
// starting to serve stale or absent data as if nothing were wrong.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ostendorf-cloud/govue/httpserver"
	"github.com/ostendorf-cloud/govue/logging"

	"github.com/ostendorf-cloud/ostendorf.tech/internal/feed"
	"github.com/ostendorf-cloud/ostendorf.tech/internal/feedconfig"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := feedconfig.Load(os.Getenv)
	if err != nil {
		return err
	}

	logger := logging.New(os.Stdout, cfg.LogLevel)

	app := feed.New(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
