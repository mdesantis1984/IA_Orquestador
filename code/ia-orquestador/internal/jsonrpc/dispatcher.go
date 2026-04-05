// Package jsonrpc implements the JSON-RPC 2.0 dispatcher for MCP
package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"github.com/thiscloud/ia-orquestador/internal/transport"
	mcperrors "github.com/thiscloud/ia-orquestador/pkg/errors"
)

// MethodHandler is a function that handles a JSON-RPC method
type MethodHandler func(ctx context.Context, params json.RawMessage) (interface{}, error)

// Dispatcher routes JSON-RPC requests to registered handlers
type Dispatcher struct {
	handlers map[string]MethodHandler
	mu       sync.RWMutex
}

// NewDispatcher creates a new JSON-RPC dispatcher
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string]MethodHandler),
	}
}

// Register adds a method handler
func (d *Dispatcher) Register(method string, handler MethodHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[method] = handler
	log.Printf("[JSONRPC] Registered method: %s", method)
}

// Unregister removes a method handler
func (d *Dispatcher) Unregister(method string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.handlers, method)
	log.Printf("[JSONRPC] Unregistered method: %s", method)
}

// Handle processes a JSON-RPC message
func (d *Dispatcher) Handle(ctx context.Context, msg *transport.Message) (*transport.Message, error) {
	// Validate JSON-RPC version
	if msg.JSONRPC != "2.0" {
		return transport.NewError(msg.ID, mcperrors.ErrInvalidRequest.Code, 
			"Invalid JSON-RPC version", nil), nil
	}
	
	// Handle notification (no response needed)
	if msg.ID == nil && msg.Method != "" {
		return d.handleNotification(ctx, msg)
	}
	
	// Handle request (requires response)
	if msg.Method != "" {
		return d.handleRequest(ctx, msg)
	}
	
	// Invalid message
	return transport.NewError(msg.ID, mcperrors.ErrInvalidRequest.Code,
		"Invalid JSON-RPC message", nil), nil
}

// handleRequest processes a JSON-RPC request
func (d *Dispatcher) handleRequest(ctx context.Context, msg *transport.Message) (*transport.Message, error) {
	tracer := otel.Tracer("ia-orquestador/jsonrpc")
	ctx, span := tracer.Start(ctx, "jsonrpc."+msg.Method)
	span.SetAttributes(attribute.String("rpc.method", msg.Method))
	defer span.End()
	d.mu.RLock()
	handler, exists := d.handlers[msg.Method]
	d.mu.RUnlock()
	
	if !exists {
		log.Printf("[JSONRPC] Method not found: %s", msg.Method)
		return transport.NewError(msg.ID, mcperrors.ErrMethodNotFound.Code,
			fmt.Sprintf("Method not found: %s", msg.Method), nil), nil
	}
	
	// Execute handler
	log.Printf("[JSONRPC] Handling request: method=%s id=%v", msg.Method, msg.ID)
	
	result, err := handler(ctx, msg.Params)
	if err != nil {
		log.Printf("[JSONRPC] Handler error: method=%s error=%v", msg.Method, err)
		
		// Check if error is MCPError
		if mcpErr, ok := err.(*mcperrors.MCPError); ok {
			return transport.NewError(msg.ID, mcpErr.Code, mcpErr.Message, mcpErr.Data), nil
		}
		
		// Generic internal error
		return transport.NewError(msg.ID, mcperrors.ErrInternal.Code, err.Error(), nil), nil
	}
	
	// Create success response
	resp, err := transport.NewResponse(msg.ID, result)
	if err != nil {
		log.Printf("[JSONRPC] Failed to create response: %v", err)
		return transport.NewError(msg.ID, mcperrors.ErrInternal.Code,
			"Failed to create response", nil), nil
	}
	
	return resp, nil
}

// handleNotification processes a JSON-RPC notification (no response)
func (d *Dispatcher) handleNotification(ctx context.Context, msg *transport.Message) (*transport.Message, error) {
	d.mu.RLock()
	handler, exists := d.handlers[msg.Method]
	d.mu.RUnlock()
	
	if !exists {
		log.Printf("[JSONRPC] Notification method not found: %s", msg.Method)
		return nil, nil // Notifications don't return errors
	}
	
	log.Printf("[JSONRPC] Handling notification: method=%s", msg.Method)
	
	// Execute handler (ignore result for notifications)
	_, err := handler(ctx, msg.Params)
	if err != nil {
		log.Printf("[JSONRPC] Notification handler error: method=%s error=%v", msg.Method, err)
	}
	
	return nil, nil // No response for notifications
}

// ListMethods returns all registered method names
func (d *Dispatcher) ListMethods() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	methods := make([]string, 0, len(d.handlers))
	for method := range d.handlers {
		methods = append(methods, method)
	}
	return methods
}

// HasMethod checks if a method is registered
func (d *Dispatcher) HasMethod(method string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, exists := d.handlers[method]
	return exists
}
