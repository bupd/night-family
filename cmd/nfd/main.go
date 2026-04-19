// Command nfd is the night-family daemon.
//
// In this skeleton it binds an HTTP server exposing /healthz, /readyz,
// /version, and a minimal HTMX-rendered landing page at /. Graceful
// shutdown is wired to SIGINT / SIGTERM.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bupd/night-family/internal/duty"
	"github.com/bupd/night-family/internal/family"
	"github.com/bupd/night-family/internal/server"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:7337", "listen address")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

	logger := newLogger(*logLevel)
	slog.SetDefault(logger)

	fam := family.NewStore()
	defaults, err := family.LoadDefaults()
	if err != nil {
		fatal(logger, "load default family: %v", err)
	}
	fam.Seed(defaults)
	logger.Info("family seeded", "count", fam.Len())

	duties := duty.NewBuiltinRegistry()
	logger.Info("duties loaded", "count", duties.Len())

	srv, err := server.New(server.Config{
		Addr:   *addr,
		Logger: logger,
		Family: fam,
		Duties: duties,
	})
	if err != nil {
		fatal(logger, "server init: %v", err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			fatal(logger, "listen: %v", err)
		}
	case s := <-sig:
		logger.Info("shutdown signal", "signal", s.String())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("shutdown", "err", err)
			os.Exit(1)
		}
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}

func fatal(logger *slog.Logger, format string, args ...any) {
	logger.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}
