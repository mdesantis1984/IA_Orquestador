// Package transport/http implements HTTP/SSE transport for MCP
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/r3labs/sse/v2"
)

// HTTPTransport implements Transport over HTTP with SSE support
type HTTPTransport struct {
	handler        Handler
	server         *http.Server
	sseServer      *sse.Server
	addr           string
	msgChan        chan *Message
	stopChan       chan struct{}
	wg             sync.WaitGroup
	mu             sync.Mutex
	stopped        bool
	clients        map[string]*sseClient
	clientsMu      sync.RWMutex
	routeRegistrar func(*http.ServeMux) // called in Start() to add extra routes
}

type sseClient struct {
	id        string
	sessionID string
	msgChan   chan *Message
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(addr string, handler Handler) *HTTPTransport {
	sseServer := sse.New()
	sseServer.AutoReplay = false
	sseServer.AutoStream = true

	return &HTTPTransport{
		handler:   handler,
		addr:      addr,
		sseServer: sseServer,
		msgChan:   make(chan *Message, 100),
		stopChan:  make(chan struct{}),
		clients:   make(map[string]*sseClient),
	}
}

// Name returns the transport name
func (t *HTTPTransport) Name() string {
	return "http"
}

// OnRoutes registers a callback that is invoked during Start() with the HTTP
// mux, allowing callers to add extra routes (admin API, metrics, etc.)
// before the server begins accepting connections.
func (t *HTTPTransport) OnRoutes(fn func(*http.ServeMux)) {
	t.routeRegistrar = fn
}

// Start initializes the HTTP transport
func (t *HTTPTransport) Start(ctx context.Context) error {
	log.Printf("[HTTP] Starting transport on %s", t.addr)

	mux := http.NewServeMux()

	// JSON-RPC endpoint
	mux.HandleFunc("/mcp/jsonrpc", t.handleJSONRPC)

	// SSE stream endpoint
	mux.HandleFunc("/mcp/stream", t.handleSSE)

	// WebSocket endpoint (full-duplex JSON-RPC)
	mux.HandleFunc("/mcp/ws", t.handleWebSocket)

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Extra routes (admin API, metrics, etc.)
	if t.routeRegistrar != nil {
		t.routeRegistrar(mux)
	}

	t.server = &http.Server{
		Addr:         t.addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[HTTP] Server error: %v", err)
		}
	}()

	log.Printf("[HTTP] Transport started on %s", t.addr)
	return nil
}

// Stop gracefully shuts down the transport
func (t *HTTPTransport) Stop(ctx context.Context) error {
	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		return nil
	}
	t.stopped = true
	t.mu.Unlock()

	log.Println("[HTTP] Stopping transport")

	// Close SSE server
	t.sseServer.Close()

	// Shutdown HTTP server
	if err := t.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	close(t.stopChan)

	// Wait for goroutines
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("[HTTP] Transport stopped gracefully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}

// Send writes a message (broadcasts to all SSE clients)
func (t *HTTPTransport) Send(msg *Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return fmt.Errorf("transport is stopped")
	}

	// Serialize message
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Broadcast to all active SSE clients.
	t.clientsMu.RLock()
	defer t.clientsMu.RUnlock()

	for _, client := range t.clients {
		select {
		case client.msgChan <- msg:
		default:
			log.Printf("[HTTP] SSE client channel full: %s", client.id)
		}
	}

	// Keep the serialized payload available for potential future pub/sub wiring.
	_ = data

	return nil
}

// SendToClient sends a message to a specific SSE client
func (t *HTTPTransport) SendToClient(clientID string, msg *Message) error {
	t.clientsMu.RLock()
	client, exists := t.clients[clientID]
	t.clientsMu.RUnlock()

	if !exists {
		return fmt.Errorf("client not found: %s", clientID)
	}

	select {
	case client.msgChan <- msg:
		return nil
	default:
		return fmt.Errorf("client channel full")
	}
}

// Receive returns the message channel
func (t *HTTPTransport) Receive() <-chan *Message {
	return t.msgChan
}

// handleJSONRPC processes JSON-RPC requests
func (t *HTTPTransport) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse message
	msg, err := ReadMessage(r.Body)
	if err != nil {
		log.Printf("[HTTP] Failed to parse request: %v", err)
		resp := NewError(nil, -32700, "Parse error", nil)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Handle message
	resp, err := t.handler.Handle(r.Context(), msg)
	if err != nil {
		log.Printf("[HTTP] Handler error: %v", err)
		errResp := NewError(msg.ID, -32603, err.Error(), nil)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errResp)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[HTTP] Failed to encode response: %v", err)
	}
}

// handleSSE processes SSE stream connections
func (t *HTTPTransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Extract session ID from query or header
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = r.Header.Get("X-Session-ID")
	}

	if sessionID == "" {
		http.Error(w, "Missing session_id", http.StatusBadRequest)
		return
	}

	// Register client
	clientID := fmt.Sprintf("%s-%d", sessionID, time.Now().UnixNano())
	client := &sseClient{
		id:        clientID,
		sessionID: sessionID,
		msgChan:   make(chan *Message, 100),
	}

	t.clientsMu.Lock()
	t.clients[clientID] = client
	t.clientsMu.Unlock()

	defer func() {
		t.clientsMu.Lock()
		delete(t.clients, clientID)
		close(client.msgChan)
		t.clientsMu.Unlock()
		log.Printf("[HTTP] SSE client disconnected: %s", clientID)
	}()

	log.Printf("[HTTP] SSE client connected: %s (session: %s)", clientID, sessionID)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Stream events
	for {
		select {
		case <-r.Context().Done():
			return
		case <-t.stopChan:
			return
		case msg := <-client.msgChan:
			if msg == nil {
				continue
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("[HTTP] Failed to marshal SSE message: %v", err)
				continue
			}

			eventType := "mcp.event"
			if msg.Error != nil {
				eventType = "mcp.error"
			} else if msg.Result != nil {
				eventType = "mcp.response"
			}

			fmt.Fprintf(w, "event: %s\n", eventType)
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			flusher.Flush()
		}
	}
}
