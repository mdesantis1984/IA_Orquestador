//go:build postgres

// Package db — PostgreSQL driver registration (build with -tags postgres).
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// openPostgres opens a PostgreSQL connection pool.
func openPostgres(cfg Config) (*sql.DB, error) {
	dsn := cfg.DSN
	if dsn == "" {
		return nil, fmt.Errorf("postgres DSN is required (-db-dsn flag)")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx open: %w", err)
	}

	// Production-ready pool settings.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pgx ping: %w", err)
	}
	log.Printf("[DB] Opened PostgreSQL connection (pool max=25)")
	return db, nil
}

// openSQLite is not available in the postgres build.
func openSQLite(_ Config) (*sql.DB, error) {
	return nil, fmt.Errorf("sqlite driver not compiled in; rebuild without -tags postgres")
}

// postgresInit sets session-level defaults for performance and safety.
func postgresInit(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `SET timezone = 'UTC'`)
	return err
}
