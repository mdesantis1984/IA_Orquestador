// Package auth implements API key authentication for the MCP HTTP transport.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ctxKey string

const (
	ctxKeyName ctxKey = "auth_name"
	ctxKeyID   ctxKey = "auth_id"
)

// Validator validates API keys against the database.
type Validator struct {
	db *sql.DB
}

// NewValidator creates a new Validator backed by the given database.
func NewValidator(db *sql.DB) *Validator {
	return &Validator{db: db}
}

// Middleware returns an HTTP middleware that enforces X-Api-Key or Bearer authentication.
func (v *Validator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Api-Key")
		if key == "" {
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				key = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		if key == "" {
			writeJSON(w, http.StatusUnauthorized, `{"error":"missing X-Api-Key or Authorization: Bearer token"}`)
			return
		}

		keyID, name, err := v.validate(r.Context(), key)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, `{"error":"invalid or revoked API key"}`)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyName, name)
		ctx = context.WithValue(ctx, ctxKeyID, keyID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (v *Validator) validate(ctx context.Context, plainKey string) (id, name string, err error) {
	hash := hashKey(plainKey)
	now := time.Now().Unix()
	err = v.db.QueryRowContext(ctx, `
		SELECT id, name FROM api_keys
		WHERE key_hash = ? AND revoked = 0 AND (expires_at IS NULL OR expires_at > ?)
	`, hash, now).Scan(&id, &name)
	if err != nil {
		return "", "", fmt.Errorf("key not found")
	}
	return id, name, nil
}

// Generate creates a new API key, stores its SHA-256 hash in the DB, and
// returns the plaintext key exactly once. The plaintext cannot be recovered.
func (v *Validator) Generate(ctx context.Context, name string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	plaintext := "mcp_" + base64.RawURLEncoding.EncodeToString(raw)
	hash := hashKey(plaintext)
	id := uuid.New().String()
	now := time.Now().Unix()

	_, err := v.db.ExecContext(ctx, `
		INSERT INTO api_keys (id, name, key_hash, scopes, created_at)
		VALUES (?, ?, ?, 'admin', ?)
	`, id, name, hash, now)
	if err != nil {
		return "", fmt.Errorf("store api key: %w", err)
	}
	return plaintext, nil
}

// ListKeys returns all non-revoked API keys (hash is never returned).
func (v *Validator) ListKeys(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := v.db.QueryContext(ctx, `
		SELECT id, name, scopes, created_at, expires_at
		FROM api_keys
		WHERE revoked = 0
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []map[string]interface{}
	for rows.Next() {
		var id, name, scopes string
		var createdAt int64
		var expiresAt sql.NullInt64
		if err := rows.Scan(&id, &name, &scopes, &createdAt, &expiresAt); err != nil {
			continue
		}
		k := map[string]interface{}{
			"id":         id,
			"name":       name,
			"scopes":     scopes,
			"created_at": time.Unix(createdAt, 0).Format(time.RFC3339),
		}
		if expiresAt.Valid {
			k["expires_at"] = time.Unix(expiresAt.Int64, 0).Format(time.RFC3339)
		}
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []map[string]interface{}{}
	}
	return keys, nil
}

// Revoke marks an API key as revoked by its ID.
func (v *Validator) Revoke(ctx context.Context, keyID string) error {
	_, err := v.db.ExecContext(ctx, `UPDATE api_keys SET revoked = 1 WHERE id = ?`, keyID)
	return err
}

// CountKeys returns the number of active (non-revoked) API keys.
func (v *Validator) CountKeys(ctx context.Context) (int, error) {
	var count int
	err := v.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_keys WHERE revoked = 0`).Scan(&count)
	return count, err
}

// NameFromContext returns the authenticated key name stored in the context.
func NameFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyName).(string)
	return v
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func writeJSON(w http.ResponseWriter, code int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write([]byte(body)) //nolint:errcheck
}
