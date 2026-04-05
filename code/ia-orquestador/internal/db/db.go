// Package db handles database initialization and migrations.
// By default the SQLite driver is used. Build with -tags postgres
// to use PostgreSQL instead.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// Config holds database configuration.
type Config struct {
	// Path is the SQLite file path (ignored when Driver=="postgres").
	Path string
	// Driver selects the backend: "sqlite" (default) or "postgres".
	Driver string
	// DSN is the full connection string for Postgres.
	// Example: "postgres://user:pass@host:5432/dbname?sslmode=require"
	DSN string
}

// Open opens a database connection using the configured driver.
func Open(cfg Config) (*sql.DB, error) {
	switch strings.ToLower(cfg.Driver) {
	case "postgres", "postgresql":
		return openPostgres(cfg)
	default:
		return openSQLite(cfg)
	}
}

// Initialize runs initial setup and migrations.
// driver is the same string passed to Open ("sqlite" or "postgres").
func Initialize(ctx context.Context, database *sql.DB, driver string) error {
	log.Println("[DB] Initializing database schema")

	pg := strings.ToLower(driver) == "postgres" || strings.ToLower(driver) == "postgresql"

	if pg {
		if err := postgresInit(ctx, database); err != nil {
			return err
		}
	} else {
		// SQLite WAL pragmas
		for _, pragma := range []string{
			"PRAGMA journal_mode = WAL",
			"PRAGMA synchronous = NORMAL",
			"PRAGMA temp_store = MEMORY",
			"PRAGMA cache_size = -2000",
			"PRAGMA foreign_keys = ON",
		} {
			if _, err := database.ExecContext(ctx, pragma); err != nil {
				return fmt.Errorf("failed to set pragma: %w", err)
			}
		}
	}

	if err := createMigrationsTable(ctx, database); err != nil {
		return err
	}

	if err := runMigrations(ctx, database, pg); err != nil {
		return err
	}

	log.Println("[DB] Database initialized successfully")
	return nil
}

// createMigrationsTable creates the schema_migrations table.
func createMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT   PRIMARY KEY,
			applied_at BIGINT NOT NULL
		)
	`)
	return err
}

// runMigrations applies all pending migrations.
func runMigrations(ctx context.Context, db *sql.DB, pg bool) error {
	type migVersion struct {
		version string
		up      func(context.Context, *sql.Tx, bool) error
	}
	migrations := []migVersion{
		{"v1_init", migrationV1Init},
		{"v2_api_keys", migrationV2ApiKeys},
	}

	recordSQL := "INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)"
	checkSQL := "SELECT COUNT(*) FROM schema_migrations WHERE version = ?"
	if pg {
		recordSQL = "INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)"
		checkSQL = "SELECT COUNT(*) FROM schema_migrations WHERE version = $1"
	}

	for _, m := range migrations {
		var count int
		if err := db.QueryRowContext(ctx, checkSQL, m.version).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", m.version, err)
		}
		if count > 0 {
			log.Printf("[DB] Migration %s already applied", m.version)
			continue
		}

		log.Printf("[DB] Applying migration %s", m.version)

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", m.version, err)
		}

		if err := m.up(ctx, tx, pg); err != nil {
			tx.Rollback() //nolint:errcheck
			return fmt.Errorf("migration %s failed: %w", m.version, err)
		}

		if _, err = tx.ExecContext(ctx, recordSQL, m.version, nowUnix()); err != nil {
			tx.Rollback() //nolint:errcheck
			return fmt.Errorf("record migration %s: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m.version, err)
		}

		log.Printf("[DB] Migration %s applied successfully", m.version)
	}

	return nil
}

// migration represents a database migration
type migration struct {
	version string
	up      func(context.Context, *sql.Tx) error
}

// migrationV1Init creates the initial schema (portable SQL for SQLite and Postgres).
func migrationV1Init(ctx context.Context, tx *sql.Tx, pg bool) error {
	auditID := "id INTEGER PRIMARY KEY AUTOINCREMENT"
	if pg {
		auditID = "id BIGSERIAL PRIMARY KEY"
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS skills (
			id         TEXT   PRIMARY KEY,
			name       TEXT   NOT NULL,
			version    TEXT   NOT NULL,
			type       TEXT   NOT NULL,
			entrypoint TEXT   NOT NULL,
			path       TEXT,
			metadata   TEXT   NOT NULL,
			status     TEXT   NOT NULL DEFAULT 'inactive',
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_skills_name         ON skills(name)`,
		`CREATE INDEX IF NOT EXISTS idx_skills_name_version ON skills(name, version)`,
		`CREATE INDEX IF NOT EXISTS idx_skills_status       ON skills(status)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id               TEXT   PRIMARY KEY,
			client_id        TEXT   NOT NULL,
			protocol_version TEXT   NOT NULL,
			capabilities     TEXT   NOT NULL,
			topic_key        TEXT,
			started_at       BIGINT NOT NULL,
			last_seen_at     BIGINT NOT NULL,
			state            TEXT   NOT NULL DEFAULT 'active'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_client    ON sessions(client_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_last_seen ON sessions(last_seen_at)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_state     ON sessions(state)`,
		`CREATE TABLE IF NOT EXISTS tokens (
			token_id   TEXT   PRIMARY KEY,
			user_id    TEXT   NOT NULL,
			token_hash TEXT   NOT NULL,
			scopes     TEXT   NOT NULL,
			audience   TEXT,
			skill_id   TEXT,
			expires_at BIGINT NOT NULL,
			created_at BIGINT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_user_expires ON tokens(user_id, expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_expires      ON tokens(expires_at)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS audit_logs (
			%s,
			timestamp  BIGINT NOT NULL,
			user_id    TEXT,
			client_id  TEXT,
			action     TEXT   NOT NULL,
			resource   TEXT,
			request_id TEXT,
			metadata   TEXT
		)`, auditID),
		`CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_user      ON audit_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_action    ON audit_logs(action)`,
	}

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("v1_init stmt failed: %w\nSQL: %s", err, stmt)
		}
	}
	return nil
}

// nowUnix returns current Unix timestamp
func nowUnix() int64 {
	return nowTime().Unix()
}

// nowTime returns current time (mockable for tests)
var nowTime = func() time.Time {
	return time.Now()
}

// migrationV2ApiKeys adds the api_keys table for HTTP auth.
func migrationV2ApiKeys(ctx context.Context, tx *sql.Tx, _ bool) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS api_keys (
			id         TEXT    PRIMARY KEY,
			name       TEXT    NOT NULL,
			key_hash   TEXT    NOT NULL UNIQUE,
			scopes     TEXT    NOT NULL DEFAULT 'admin',
			created_at BIGINT  NOT NULL,
			expires_at BIGINT,
			revoked    INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_hash    ON api_keys(key_hash)`,
		`CREATE INDEX        IF NOT EXISTS idx_api_keys_revoked ON api_keys(revoked)`,
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("v2_api_keys stmt failed: %w\nSQL: %s", err, stmt)
		}
	}
	return nil
}
