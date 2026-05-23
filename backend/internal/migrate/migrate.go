// Package migrate wraps golang-migrate with our migrations embedded
// and exposes Up / Down / Status helpers used by the CLI subcommands.
package migrate

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed all:migrations
var migrationsFS embed.FS

// new constructs a *migrate.Migrate against the given Postgres DSN.
func newMigrate(dsn string) (*migrate.Migrate, error) {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("subfs: %w", err)
	}
	src, err := iofs.New(sub, ".")
	if err != nil {
		return nil, fmt.Errorf("iofs: %w", err)
	}
	// golang-migrate's pgx/v5 driver registers under the "pgx5" scheme.
	// Accept the more familiar "postgres://" / "postgresql://" forms by
	// rewriting the scheme transparently.
	dsn = normaliseDSN(dsn)
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return nil, fmt.Errorf("migrate new: %w", err)
	}
	return m, nil
}

func normaliseDSN(dsn string) string {
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if len(dsn) >= len(prefix) && dsn[:len(prefix)] == prefix {
			return "pgx5://" + dsn[len(prefix):]
		}
	}
	return dsn
}

// Up applies all pending migrations.
func Up(_ context.Context, dsn string, logger *slog.Logger) error {
	m, err := newMigrate(dsn)
	if err != nil {
		return err
	}
	defer closeMigrate(m, logger)
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	v, dirty, _ := m.Version()
	logger.Info("migrate up complete", "version", v, "dirty", dirty)
	return nil
}

// Down rolls back the last applied migration (or all of them when n<=0).
func Down(_ context.Context, dsn string, n int, logger *slog.Logger) error {
	m, err := newMigrate(dsn)
	if err != nil {
		return err
	}
	defer closeMigrate(m, logger)
	if n <= 0 {
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migrate down all: %w", err)
		}
	} else {
		if err := m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migrate down %d: %w", n, err)
		}
	}
	v, dirty, _ := m.Version()
	logger.Info("migrate down complete", "version", v, "dirty", dirty)
	return nil
}

// Status reports the current schema version and lists every migration in
// the embedded source.
func Status(_ context.Context, dsn string, logger *slog.Logger) error {
	m, err := newMigrate(dsn)
	if err != nil {
		return err
	}
	defer closeMigrate(m, logger)

	v, dirty, err := m.Version()
	switch {
	case errors.Is(err, migrate.ErrNilVersion):
		logger.Info("schema is empty (no migrations applied yet)")
	case err != nil:
		return fmt.Errorf("version: %w", err)
	default:
		logger.Info("current schema version", "version", v, "dirty", dirty)
	}

	files, err := listMigrations()
	if err != nil {
		return err
	}
	for _, f := range files {
		logger.Info("available", "file", f)
	}
	return nil
}

func closeMigrate(m *migrate.Migrate, logger *slog.Logger) {
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		logger.Warn("migrate source close error", "err", srcErr)
	}
	if dbErr != nil {
		logger.Warn("migrate db close error", "err", dbErr)
	}
}

func listMigrations() ([]string, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read embed dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
