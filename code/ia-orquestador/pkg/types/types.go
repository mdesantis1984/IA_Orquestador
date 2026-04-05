// Package types defines core domain types for MCP Orchestrator
package types

import (
	"encoding/json"
	"time"
)

// SkillType represents the skill execution model
type SkillType string

const (
	SkillTypeSDD    SkillType = "sdd"
	SkillTypeDotNet SkillType = "dotnet"
	SkillTypeHTTP   SkillType = "http"
	SkillTypeWASM   SkillType = "wasm"
)

// SkillStatus represents skill lifecycle state
type SkillStatus string

const (
	SkillStatusInactive   SkillStatus = "inactive"
	SkillStatusActive     SkillStatus = "active"
	SkillStatusDeprecated SkillStatus = "deprecated"
)

// Skill represents a registered MCP skill
type Skill struct {
	ID         string          `json:"id" db:"id"`
	Name       string          `json:"name" db:"name"`
	Version    string          `json:"version" db:"version"`
	Type       SkillType       `json:"type" db:"type"`
	Entrypoint string          `json:"entrypoint" db:"entrypoint"`
	Path       string          `json:"path,omitempty" db:"path"`
	Metadata   json.RawMessage `json:"metadata" db:"metadata"`
	Status     SkillStatus     `json:"status" db:"status"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}

// SkillMetadata contains skill capabilities and descriptors
type SkillMetadata struct {
	Capabilities []string          `json:"capabilities"`
	Tags         []string          `json:"tags"`
	Description  string            `json:"description"`
	Extra        map[string]string `json:"extra,omitempty"`
}

// Session represents an active MCP client session
type Session struct {
	ID              string          `json:"id" db:"id"`
	ClientID        string          `json:"client_id" db:"client_id"`
	ProtocolVersion string          `json:"protocol_version" db:"protocol_version"`
	Capabilities    json.RawMessage `json:"capabilities" db:"capabilities"`
	TopicKey        string          `json:"topic_key,omitempty" db:"topic_key"`
	StartedAt       time.Time       `json:"started_at" db:"started_at"`
	LastSeenAt      time.Time       `json:"last_seen_at" db:"last_seen_at"`
	State           string          `json:"state" db:"state"`
}

// SessionState represents session lifecycle state
type SessionState string

const (
	SessionStateActive  SessionState = "active"
	SessionStateEnded   SessionState = "ended"
	SessionStateExpired SessionState = "expired"
)

// Token represents a cached OBO/downstream token
type Token struct {
	TokenID   string    `json:"token_id" db:"token_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	TokenHash string    `json:"-" db:"token_hash"`
	Scopes    string    `json:"scopes" db:"scopes"`
	Audience  string    `json:"audience,omitempty" db:"audience"`
	SkillID   string    `json:"skill_id,omitempty" db:"skill_id"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// AuditLog represents an immutable audit trail entry
type AuditLog struct {
	ID        int64           `json:"id" db:"id"`
	Timestamp time.Time       `json:"timestamp" db:"timestamp"`
	UserID    string          `json:"user_id,omitempty" db:"user_id"`
	ClientID  string          `json:"client_id,omitempty" db:"client_id"`
	Action    string          `json:"action" db:"action"`
	Resource  string          `json:"resource,omitempty" db:"resource"`
	RequestID string          `json:"request_id,omitempty" db:"request_id"`
	Metadata  json.RawMessage `json:"metadata,omitempty" db:"metadata"`
}
