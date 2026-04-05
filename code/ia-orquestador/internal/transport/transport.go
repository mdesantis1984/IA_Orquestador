// Package transport defines transport abstractions for MCP
package transport

import (
	"context"
	"encoding/json"
	"io"
)

// Message represents a JSON-RPC 2.0 message
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`      // string or number
	Method  string          `json:"method,omitempty"`  // for requests
	Params  json.RawMessage `json:"params,omitempty"`  // for requests
	Result  json.RawMessage `json:"result,omitempty"`  // for responses
	Error   *ErrorObject    `json:"error,omitempty"`   // for errors
}

// ErrorObject represents a JSON-RPC 2.0 error object
type ErrorObject struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// Transport defines the interface for MCP transports
type Transport interface {
	// Start initializes the transport
	Start(ctx context.Context) error
	
	// Stop gracefully shuts down the transport
	Stop(ctx context.Context) error
	
	// Send writes a message to the transport
	Send(msg *Message) error
	
	// Receive reads messages from the transport
	Receive() <-chan *Message
	
	// Name returns the transport name
	Name() string
}

// Handler processes incoming messages
type Handler interface {
	Handle(ctx context.Context, msg *Message) (*Message, error)
}

// HandlerFunc is a function adapter for Handler
type HandlerFunc func(ctx context.Context, msg *Message) (*Message, error)

func (f HandlerFunc) Handle(ctx context.Context, msg *Message) (*Message, error) {
	return f(ctx, msg)
}

// NewRequest creates a new JSON-RPC request
func NewRequest(id interface{}, method string, params interface{}) (*Message, error) {
	var paramsRaw json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		paramsRaw = data
	}
	
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsRaw,
	}, nil
}

// NewResponse creates a new JSON-RPC response
func NewResponse(id interface{}, result interface{}) (*Message, error) {
	var resultRaw json.RawMessage
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		resultRaw = data
	}
	
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultRaw,
	}, nil
}

// NewError creates a new JSON-RPC error response
func NewError(id interface{}, code int, message string, data map[string]interface{}) *Message {
	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ErrorObject{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// NewNotification creates a new JSON-RPC notification (no ID)
func NewNotification(method string, params interface{}) (*Message, error) {
	var paramsRaw json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		paramsRaw = data
	}
	
	return &Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsRaw,
	}, nil
}

// ReadMessage reads and decodes a JSON-RPC message from a reader
func ReadMessage(r io.Reader) (*Message, error) {
	dec := json.NewDecoder(r)
	var msg Message
	if err := dec.Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// WriteMessage encodes and writes a JSON-RPC message to a writer
func WriteMessage(w io.Writer, msg *Message) error {
	enc := json.NewEncoder(w)
	return enc.Encode(msg)
}
