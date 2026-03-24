package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migpostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

// Embed the whole `migrations/` directory so the io/fs driver can resolve it
// reliably across Go versions and filesystem implementations.
//go:embed migrations
var migrationsFS embed.FS

// RunMigrations runs all pending golang-migrate migrations against the provided DSN.
// It is safe to call on startup.
func RunMigrations(dsn string) error {
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("migrations: open postgres: %w", err)
	}
	defer sqlDB.Close()

	// Ensure the DB is reachable early (avoid failing later during migration execution).
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("migrations: ping postgres: %w", err)
	}

	dbDriver, err := migpostgres.WithInstance(sqlDB, &migpostgres.Config{})
	if err != nil {
		return fmt.Errorf("migrations: init postgres driver: %w", err)
	}

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migrations: init iofs source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("migrations: new migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	// Safety net: if the migration bookkeeping says the DB is at latest
	// version but the expected table is missing (e.g., forced/dirty state),
	// re-run migrations from scratch.
	var urlsExists bool
	if err := sqlDB.QueryRow(`SELECT to_regclass('public.urls') IS NOT NULL`).Scan(&urlsExists); err != nil {
		return fmt.Errorf("migrations: check urls table existence: %w", err)
	}
	if !urlsExists {
		if err := m.Force(-1); err != nil {
			return fmt.Errorf("migrations: force reset to base: %w", err)
		}
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("migrations: rerun up after missing table: %w", err)
		}
		return nil
	}

	if err := m.Up(); err != nil {
		// If a previous run crashed mid-migration, golang-migrate marks the
		// migration version as "dirty" and refuses to continue.
		// Reset the dirty state and retry.
		var dirty migrate.ErrDirty
		if errors.As(err, &dirty) {
			// If we are dirty at version N, migration N likely failed part-way.
			// Force to N-1 so `Up()` will re-apply migration N.
			targetVersion := dirty.Version - 1
			if targetVersion < -1 {
				targetVersion = -1
			}
			if forceErr := m.Force(targetVersion); forceErr != nil {
				return fmt.Errorf("migrations: force version %d (from dirty %d): %w", targetVersion, dirty.Version, forceErr)
			}
			if upErr := m.Up(); upErr != nil && upErr != migrate.ErrNoChange {
				return fmt.Errorf("migrations: retry migrate up: %w", upErr)
			}
		} else if err != migrate.ErrNoChange {
			return fmt.Errorf("migrations: migrate up: %w", err)
		}
	}

	return nil
}

