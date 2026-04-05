package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestMessage_Marshaling(t *testing.T) {
	msg := &Message{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test.method",
		Params:  json.RawMessage(`{"key":"value"}`),
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	var decoded Message
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	
	if decoded.Method != msg.Method {
		t.Errorf("Method mismatch: got %s, want %s", decoded.Method, msg.Method)
	}
}

func TestNewRequest(t *testing.T) {
	params := map[string]string{"key": "value"}
	
	msg, err := NewRequest(1, "test.method", params)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	
	if msg.JSONRPC != "2.0" {
		t.Errorf("JSONRPC version mismatch: got %s, want 2.0", msg.JSONRPC)
	}
	
	if msg.Method != "test.method" {
		t.Errorf("Method mismatch: got %s, want test.method", msg.Method)
	}
	
	if msg.ID != 1 {
		t.Errorf("ID mismatch: got %v, want 1", msg.ID)
	}
	
	if msg.Params == nil {
		t.Error("Params should not be nil")
	}
}

func TestNewRequest_NilParams(t *testing.T) {
	msg, err := NewRequest(1, "test.method", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	
	if msg.Params != nil {
		t.Error("Params should be nil")
	}
}

func TestNewResponse(t *testing.T) {
	result := map[string]string{"status": "ok"}
	
	msg, err := NewResponse(1, result)
	if err != nil {
		t.Fatalf("NewResponse failed: %v", err)
	}
	
	if msg.JSONRPC != "2.0" {
		t.Errorf("JSONRPC version mismatch: got %s, want 2.0", msg.JSONRPC)
	}
	
	if msg.ID != 1 {
		t.Errorf("ID mismatch: got %v, want 1", msg.ID)
	}
	
	if msg.Result == nil {
		t.Error("Result should not be nil")
	}
	
	var decoded map[string]string
	err = json.Unmarshal(msg.Result, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal result failed: %v", err)
	}
	
	if decoded["status"] != "ok" {
		t.Errorf("Result mismatch: got %s, want ok", decoded["status"])
	}
}

func TestNewError(t *testing.T) {
	data := map[string]interface{}{"extra": "info"}
	
	msg := NewError(1, -32600, "Invalid Request", data)
	
	if msg.JSONRPC != "2.0" {
		t.Errorf("JSONRPC version mismatch: got %s, want 2.0", msg.JSONRPC)
	}
	
	if msg.ID != 1 {
		t.Errorf("ID mismatch: got %v, want 1", msg.ID)
	}
	
	if msg.Error == nil {
		t.Fatal("Error should not be nil")
	}
	
	if msg.Error.Code != -32600 {
		t.Errorf("Error code mismatch: got %d, want -32600", msg.Error.Code)
	}
	
	if msg.Error.Message != "Invalid Request" {
		t.Errorf("Error message mismatch: got %s, want Invalid Request", msg.Error.Message)
	}
	
	if msg.Error.Data["extra"] != "info" {
		t.Errorf("Error data mismatch")
	}
}

func TestNewNotification(t *testing.T) {
	params := map[string]int{"count": 42}
	
	msg, err := NewNotification("test.notification", params)
	if err != nil {
		t.Fatalf("NewNotification failed: %v", err)
	}
	
	if msg.JSONRPC != "2.0" {
		t.Errorf("JSONRPC version mismatch: got %s, want 2.0", msg.JSONRPC)
	}
	
	if msg.ID != nil {
		t.Error("Notification should not have ID")
	}
	
	if msg.Method != "test.notification" {
		t.Errorf("Method mismatch: got %s, want test.notification", msg.Method)
	}
}

func TestReadMessage(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"test.method","params":{"key":"value"}}`
	
	reader := bytes.NewBufferString(input)
	
	msg, err := ReadMessage(reader)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	
	if msg.Method != "test.method" {
		t.Errorf("Method mismatch: got %s, want test.method", msg.Method)
	}
}

func TestReadMessage_Invalid(t *testing.T) {
	input := `{invalid json`
	
	reader := bytes.NewBufferString(input)
	
	_, err := ReadMessage(reader)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestWriteMessage(t *testing.T) {
	msg := &Message{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test.method",
		Params:  json.RawMessage(`{"key":"value"}`),
	}
	
	var buf bytes.Buffer
	
	err := WriteMessage(&buf, msg)
	if err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}
	
	// Read back
	decoded, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	
	if decoded.Method != msg.Method {
		t.Errorf("Method mismatch after write/read cycle")
	}
}

func TestHandlerFunc(t *testing.T) {
	called := false
	
	handler := HandlerFunc(func(ctx context.Context, msg *Message) (*Message, error) {
		called = true
		return NewResponse(msg.ID, "ok")
	})
	
	msg := &Message{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
	}
	
	resp, err := handler.Handle(nil, msg)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}
	
	if !called {
		t.Error("Handler function not called")
	}
	
	if resp == nil {
		t.Error("Response should not be nil")
	}
}

func TestErrorObject_Marshaling(t *testing.T) {
	errObj := &ErrorObject{
		Code:    -32600,
		Message: "Invalid Request",
		Data: map[string]interface{}{
			"detail": "missing method field",
		},
	}
	
	data, err := json.Marshal(errObj)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	var decoded ErrorObject
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	
	if decoded.Code != errObj.Code {
		t.Errorf("Code mismatch: got %d, want %d", decoded.Code, errObj.Code)
	}
	
	if decoded.Message != errObj.Message {
		t.Errorf("Message mismatch: got %s, want %s", decoded.Message, errObj.Message)
	}
}

func TestMessage_IDTypes(t *testing.T) {
	tests := []struct {
		name string
		id   interface{}
	}{
		{"string ID", "request-123"},
		{"integer ID", 42},
		{"float ID", 3.14},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := NewRequest(tt.id, "test.method", nil)
			if err != nil {
				t.Fatalf("NewRequest failed: %v", err)
			}
			
			// Marshal and unmarshal
			data, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			
			var decoded Message
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			
			// Type might change (int -> float in JSON), so just check non-nil
			if decoded.ID == nil {
				t.Error("ID should not be nil after round-trip")
			}
		})
	}
}
