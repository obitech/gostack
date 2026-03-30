// Package database provides database connectivity with migrations.
package database

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Required for postgres driver registration
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Database holds the connection pool.
type Database struct {
	pool *pgxpool.Pool
}

// New creates a new database connection, runs migrations, and returns a Database.
// It configures OpenTelemetry instrumentation via otelpgx for metrics collection.
func New(ctx context.Context, databaseURL string) (*Database, error) {
	if err := runMigrations(databaseURL); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing connection config: %w", err)
	}

	cfg.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	if err := otelpgx.RecordStats(pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("recording pool stats: %w", err)
	}

	return &Database{pool: pool}, nil
}

// Pool returns the underlying connection pool for direct queries.
func (d *Database) Pool() *pgxpool.Pool {
	return d.pool
}

// Close closes the database connection pool.
func (d *Database) Close() {
	d.pool.Close()
}

func runMigrations(databaseURL string) error {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return fmt.Errorf("creating migration instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}
