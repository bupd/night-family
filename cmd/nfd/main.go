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
	"github.com/bupd/night-family/internal/provider"
	"github.com/bupd/night-family/internal/runner"
	"github.com/bupd/night-family/internal/schedule"
	"github.com/bupd/night-family/internal/server"
	"github.com/bupd/night-family/internal/storage"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:7337", "listen address")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	dbPath := flag.String("db", "", "path to SQLite database (default ~/.local/share/night-family/nf.db; use ':memory:' for ephemeral)")
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

	sched := schedule.Default()
	if err := sched.Validate(); err != nil {
		fatal(logger, "default schedule invalid: %v", err)
	}
	logger.Info("schedule", "window", sched.WindowStart+"-"+sched.WindowEnd, "tz", sched.TimeZone)

	db, err := openStorage(context.Background(), *dbPath, logger)
	if err != nil {
		fatal(logger, "open storage: %v", err)
	}
	defer db.Close()

	prov := provider.NewMock()
	logger.Info("provider", "name", prov.Name())
	run, err := runner.New(runner.Deps{
		Family:   fam,
		Duties:   duties,
		Storage:  db,
		Provider: prov,
		Logger:   logger,
	})
	if err != nil {
		fatal(logger, "runner: %v", err)
	}

	srv, err := server.New(server.Config{
		Addr:     *addr,
		Logger:   logger,
		Family:   fam,
		Duties:   duties,
		Schedule: &sched,
		Storage:  db,
		Runner:   run,
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

// openStorage resolves the DB path (falling back to the XDG state
// directory under night-family/nf.db) and calls storage.Open.
func openStorage(ctx context.Context, explicit string, logger *slog.Logger) (*storage.DB, error) {
	dsn := explicit
	if dsn == "" {
		stateDir := os.Getenv("XDG_STATE_HOME")
		if stateDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			stateDir = home + "/.local/share"
		}
		dir := stateDir + "/night-family"
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		dsn = "file:" + dir + "/nf.db"
	}
	logger.Info("storage", "dsn", dsn)
	return storage.Open(ctx, dsn)
}

func fatal(logger *slog.Logger, format string, args ...any) {
	logger.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}
