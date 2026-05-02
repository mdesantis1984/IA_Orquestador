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
	ctxKeyName   ctxKey = "auth_name"
	ctxKeyID     ctxKey = "auth_id"
	ctxKeyScopes ctxKey = "auth_scopes"
)

const ContextKeyUserID ctxKey = "auth_id"

type apiKeyRecord struct {
	ID     string
	Name   string
	Scopes string
	Revoked bool
}

// Validator validates API keys against the database.
type Validator struct {
	db *sql.DB
}

// NewValidator creates a new Validator backed by the given database.
func NewValidator(db *sql.DB) *Validator {
	return &Validator{db: db}
}

// Middleware returns an HTTP middleware that enforces API-key authentication.
func (v *Validator) Middleware(next http.Handler) http.Handler {
	return v.authorize(next)
}

// WithScopes returns middleware that enforces API-key auth plus the required scopes.
func (v *Validator) WithScopes(requiredScopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return v.authorize(next, requiredScopes...)
	}
}

func (v *Validator) authorize(next http.Handler, requiredScopes ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractKey(r)
		if key == "" {
			writeJSON(w, http.StatusUnauthorized, `{"error":"missing_credentials"}`)
			return
		}

		record, err := v.lookup(r.Context(), key)
		if err != nil || record == nil {
			writeJSON(w, http.StatusUnauthorized, `{"error":"invalid_credentials"}`)
			return
		}
		if record.Revoked {
			writeJSON(w, http.StatusForbidden, `{"error":"revoked_credentials"}`)
			return
		}
		if len(requiredScopes) > 0 && !hasScopes(record.Scopes, requiredScopes...) {
			writeJSON(w, http.StatusForbidden, `{"error":"insufficient_scope"}`)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyName, record.Name)
		ctx = context.WithValue(ctx, ctxKeyID, record.ID)
		ctx = context.WithValue(ctx, ctxKeyScopes, record.Scopes)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (v *Validator) lookup(ctx context.Context, plainKey string) (*apiKeyRecord, error) {
	hash := hashKey(plainKey)
	var record apiKeyRecord
	var revoked int
	var expiresAt sql.NullInt64
	err := v.db.QueryRowContext(ctx, `
		SELECT id, name, scopes, revoked, expires_at FROM api_keys
		WHERE key_hash = $1
	`, hash).Scan(&record.ID, &record.Name, &record.Scopes, &revoked, &expiresAt)
	if err != nil {
		return nil, fmt.Errorf("key not found")
	}
	record.Revoked = revoked != 0
	if record.Revoked {
		return &record, nil
	}
	if expiresAt.Valid && expiresAt.Int64 <= time.Now().Unix() {
		return nil, fmt.Errorf("key expired")
	}
	return &record, nil
}

// Generate creates a new API key, stores its SHA-256 hash in the DB, and
// returns the plaintext key exactly once. The plaintext cannot be recovered.
// scopes is a comma-separated list of permissions (e.g. "admin,read,write").
// If empty, defaults to "admin".
func (v *Validator) Generate(ctx context.Context, name string, scopes string) (string, error) {
	if scopes == "" {
		scopes = "admin"
	}
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
		VALUES ($1, $2, $3, $4, $5)
	`, id, name, hash, scopes, now)
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
	_, err := v.db.ExecContext(ctx, `UPDATE api_keys SET revoked = 1 WHERE id = $1`, keyID)
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

// ScKey is the context key for user ID.
const scKey ctxKey = "auth_id"

// ScopesFromContext returns the authenticated scopes stored in the context.
func ScopesFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyScopes).(string)
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

func extractKey(r *http.Request) string {
	key := r.Header.Get("X-Api-Key")
	if key != "" {
		return key
	}
	if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

func hasScopes(actual string, required ...string) bool {
	if len(required) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, s := range strings.FieldsFunc(strings.ToLower(actual), func(r rune) bool { return r == ',' || r == ' ' || r == ';' }) {
		if s != "" {
			set[s] = struct{}{}
		}
	}
	if _, ok := set["owner"]; ok {
		return true
	}
	for _, req := range required {
		if _, ok := set[strings.ToLower(req)]; ok {
			return true
		}
	}
	return false
}
