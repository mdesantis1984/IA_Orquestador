package jsonrpc

import (
	"context"
	"encoding/json"
	"testing"
	
	"github.com/thiscloud/ia-orquestador/internal/transport"
	mcperrors "github.com/thiscloud/ia-orquestador/pkg/errors"
)

func TestDispatcher_Register(t *testing.T) {
	d := NewDispatcher()
	
	called := false
	handler := func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		called = true
		return "ok", nil
	}
	
	d.Register("test.method", handler)
	
	if !d.HasMethod("test.method") {
		t.Fatal("Method not registered")
	}
	
	// Test call
	msg := &transport.Message{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test.method",
		Params:  json.RawMessage(`{}`),
	}
	
	resp, err := d.Handle(context.Background(), msg)
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	
	if !called {
		t.Fatal("Handler not called")
	}
	
	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}
}

func TestDispatcher_MethodNotFound(t *testing.T) {
	d := NewDispatcher()
	
	msg := &transport.Message{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown.method",
		Params:  json.RawMessage(`{}`),
	}
	
	resp, err := d.Handle(context.Background(), msg)
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	
	if resp.Error == nil {
		t.Fatal("Expected error response")
	}
	
	if resp.Error.Code != mcperrors.ErrMethodNotFound.Code {
		t.Fatalf("Wrong error code: got %d, want %d", 
			resp.Error.Code, mcperrors.ErrMethodNotFound.Code)
	}
}

func TestDispatcher_Notification(t *testing.T) {
	d := NewDispatcher()
	
	called := false
	handler := func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		called = true
		return nil, nil
	}
	
	d.Register("test.notification", handler)
	
	// Notification (no ID)
	msg := &transport.Message{
		JSONRPC: "2.0",
		Method:  "test.notification",
		Params:  json.RawMessage(`{}`),
	}
	
	resp, err := d.Handle(context.Background(), msg)
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	
	if !called {
		t.Fatal("Handler not called")
	}
	
	// Notifications should not return response
	if resp != nil {
		t.Fatal("Notification should not return response")
	}
}

func TestDispatcher_InvalidJSONRPC(t *testing.T) {
	d := NewDispatcher()
	
	msg := &transport.Message{
		JSONRPC: "1.0", // Wrong version
		ID:      1,
		Method:  "test.method",
	}
	
	resp, err := d.Handle(context.Background(), msg)
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	
	if resp.Error == nil {
		t.Fatal("Expected error response")
	}
	
	if resp.Error.Code != mcperrors.ErrInvalidRequest.Code {
		t.Fatalf("Wrong error code: got %d, want %d",
			resp.Error.Code, mcperrors.ErrInvalidRequest.Code)
	}
}
