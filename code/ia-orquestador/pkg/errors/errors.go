// Package errors defines standard error codes and types for MCP
package errors

import "fmt"

// MCPError represents a structured MCP error
type MCPError struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

func (e *MCPError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// Standard JSON-RPC 2.0 errors
var (
	ErrInvalidRequest = &MCPError{Code: -32600, Message: "Invalid Request"}
	ErrMethodNotFound = &MCPError{Code: -32601, Message: "Method not found"}
	ErrInvalidParams  = &MCPError{Code: -32602, Message: "Invalid params"}
	ErrInternal       = &MCPError{Code: -32603, Message: "Internal error"}
)

// Custom MCP errors
var (
	ErrRateLimited       = &MCPError{Code: -32001, Message: "Rate limited"}
	ErrAuthFailed        = &MCPError{Code: -32002, Message: "Authentication failed"}
	ErrPermissionDenied  = &MCPError{Code: -32003, Message: "Permission denied"}
	ErrSkillExecution    = &MCPError{Code: -32004, Message: "Skill execution error"}
	ErrSkillNotFound     = &MCPError{Code: -32005, Message: "Skill not found"}
	ErrSessionNotFound   = &MCPError{Code: -32006, Message: "Session not found"}
	ErrSessionExpired    = &MCPError{Code: -32007, Message: "Session expired"}
	ErrInvalidToken      = &MCPError{Code: -32008, Message: "Invalid token"}
	ErrTokenExpired      = &MCPError{Code: -32009, Message: "Token expired"}
	ErrInsufficientScope = &MCPError{Code: -32010, Message: "Insufficient scope"}
)

// NewMCPError creates a new MCP error with optional data
func NewMCPError(base *MCPError, data map[string]interface{}) *MCPError {
	return &MCPError{
		Code:    base.Code,
		Message: base.Message,
		Data:    data,
	}
}

// WithMessage creates a new error with custom message
func WithMessage(base *MCPError, msg string) *MCPError {
	return &MCPError{
		Code:    base.Code,
		Message: msg,
		Data:    base.Data,
	}
}
