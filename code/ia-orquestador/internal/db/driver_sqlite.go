//go:build !postgres

// Package db — SQLite driver registration (default build).
package db

import (
	"fmt"
	"log"

	"database/sql"

	_ "modernc.org/sqlite"
)

// openSQLite opens a SQLite connection with WAL-friendly settings.
func openSQLite(cfg Config) (*sql.DB, error) {
	path := cfg.Path
	if path == "" {
		path = "./orchestrator.db"
	}
	dsn := fmt.Sprintf("file:%s?_journal=WAL&_timeout=5000&_fk=true", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}
	db.SetMaxOpenConns(1) // single writer — WAL safe
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}
	log.Printf("[DB] Opened SQLite database: %s", path)
	return db, nil
}

// openPostgres is not available in the default (non-postgres) build.
func openPostgres(_ Config) (*sql.DB, error) {
	return nil, fmt.Errorf("postgres driver not compiled in; rebuild with -tags postgres")
}

// postgresInit is a no-op in the default build.
func postgresInit(_ interface{}, _ *sql.DB) error {
	return fmt.Errorf("postgres not compiled in")
}
