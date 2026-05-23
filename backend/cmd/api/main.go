// Package main is the entry point for the Tansiq Information Portal API.
// It dispatches to subcommands: serve (default), migrate, seed.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/me-nazim/criminal-archive/backend/internal/admincmd"
	"github.com/me-nazim/criminal-archive/backend/internal/config"
	"github.com/me-nazim/criminal-archive/backend/internal/db"
	apimigrate "github.com/me-nazim/criminal-archive/backend/internal/migrate"
	"github.com/me-nazim/criminal-archive/backend/internal/router"
	"github.com/me-nazim/criminal-archive/backend/internal/seed"
)

func main() {
	logger := buildLogger()
	slog.SetDefault(logger)

	cmd := "serve"
	args := os.Args[1:]
	if len(args) > 0 && !isFlag(args[0]) {
		cmd = args[0]
		args = args[1:]
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	switch cmd {
	case "serve":
		if err := runServe(ctx, cfg, logger); err != nil {
			logger.Error("serve failed", "err", err)
			os.Exit(1)
		}
	case "migrate":
		if err := runMigrate(ctx, cfg, logger, args); err != nil {
			logger.Error("migrate failed", "err", err)
			os.Exit(1)
		}
	case "seed":
		if err := runSeed(ctx, cfg, logger, args); err != nil {
			logger.Error("seed failed", "err", err)
			os.Exit(1)
		}
	case "admin":
		if err := runAdmin(ctx, cfg, logger, args); err != nil {
			logger.Error("admin failed", "err", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n%s", cmd, usage())
		os.Exit(2)
	}
}

func isFlag(s string) bool { return len(s) > 0 && s[0] == '-' }

func usage() string {
	return `usage:
  api serve                       # run the HTTP API (default)
  api migrate up|down|status      # apply or roll back schema migrations
    api migrate down --n 1        #   roll back the last N migrations
  api seed [--reset]              # load reference data into the DB
  api admin bootstrap             # create or promote a super-admin account
    --email user@example.com      #   required
    --role super_admin            #   role to grant (default super_admin)
    --name "Full Name"            #   required when creating a new user
    --password "<long-password>"  #   required when creating a new user
`
}

func buildLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func runServe(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Warn("database connection failed at startup; continuing", "err", err)
	} else {
		defer pool.Close()
	}

	r := router.New(cfg, pool, logger)
	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("api listening", "port", cfg.AppPort, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down api")
	case err := <-errCh:
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	return nil
}

func runMigrate(ctx context.Context, cfg *config.Config, logger *slog.Logger, args []string) error {
	if len(args) == 0 {
		return errors.New("migrate requires a verb: up | down | status")
	}
	verb := args[0]
	fs := flag.NewFlagSet("migrate "+verb, flag.ContinueOnError)
	steps := fs.Int("n", 0, "number of steps (down only; 0 = all)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	switch verb {
	case "up":
		return apimigrate.Up(ctx, cfg.DatabaseURL, logger)
	case "down":
		return apimigrate.Down(ctx, cfg.DatabaseURL, *steps, logger)
	case "status":
		return apimigrate.Status(ctx, cfg.DatabaseURL, logger)
	default:
		return fmt.Errorf("unknown migrate verb %q", verb)
	}
}

func runSeed(ctx context.Context, cfg *config.Config, logger *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	reset := fs.Bool("reset", false, "delete existing seeded rows before re-loading")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer pool.Close()
	return seed.Run(ctx, pool, *reset, logger)
}

func runAdmin(ctx context.Context, cfg *config.Config, logger *slog.Logger, args []string) error {
	if len(args) == 0 {
		return errors.New("admin requires a subcommand: bootstrap")
	}
	verb := args[0]
	if verb != "bootstrap" {
		return fmt.Errorf("unknown admin subcommand %q", verb)
	}
	fs := flag.NewFlagSet("admin bootstrap", flag.ContinueOnError)
	email := fs.String("email", "", "user email (required)")
	role := fs.String("role", "super_admin", "role to grant")
	name := fs.String("name", "", "full name (required when creating a new user)")
	password := fs.String("password", "", "password (required when creating a new user)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer pool.Close()
	return admincmd.Run(ctx, pool, admincmd.CreateOrPromoteParams{
		Email:      *email,
		FullName:   *name,
		Password:   *password,
		Role:       *role,
		BcryptCost: cfg.BcryptCost,
	}, logger)
}
