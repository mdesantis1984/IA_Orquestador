// Package transport/stdio implements STDIO transport for local MCP
package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// STDIOTransport implements Transport over stdin/stdout
type STDIOTransport struct {
	handler  Handler
	input    io.Reader
	output   io.Writer
	msgChan  chan *Message
	errChan  chan error
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	stopped  bool
}

// NewSTDIOTransport creates a new STDIO transport
func NewSTDIOTransport(handler Handler) *STDIOTransport {
	return &STDIOTransport{
		handler:  handler,
		input:    os.Stdin,
		output:   os.Stdout,
		msgChan:  make(chan *Message, 100),
		errChan:  make(chan error, 10),
		stopChan: make(chan struct{}),
	}
}

// Name returns the transport name
func (t *STDIOTransport) Name() string {
	return "stdio"
}

// Start initializes the STDIO transport
func (t *STDIOTransport) Start(ctx context.Context) error {
	log.Println("[STDIO] Starting transport")

	t.wg.Add(2)

	// Reader goroutine
	go func() {
		defer t.wg.Done()
		t.readLoop()
	}()

	// Handler goroutine
	go func() {
		defer t.wg.Done()
		t.handleLoop(ctx)
	}()

	return nil
}

// Stop gracefully shuts down the transport
func (t *STDIOTransport) Stop(ctx context.Context) error {
	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		return nil
	}
	t.stopped = true
	t.mu.Unlock()

	log.Println("[STDIO] Stopping transport")
	close(t.stopChan)

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("[STDIO] Transport stopped gracefully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}

// Send writes a message to stdout
func (t *STDIOTransport) Send(msg *Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return fmt.Errorf("transport is stopped")
	}

	return WriteMessage(t.output, msg)
}

// Receive returns the message channel
func (t *STDIOTransport) Receive() <-chan *Message {
	return t.msgChan
}

// readLoop reads messages from stdin
func (t *STDIOTransport) readLoop() {
	scanner := bufio.NewScanner(t.input)
	scanner.Split(bufio.ScanLines)

	for {
		select {
		case <-t.stopChan:
			log.Println("[STDIO] Read loop stopped")
			return
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					log.Printf("[STDIO] Read error: %v", err)
				}
				log.Println("[STDIO] EOF reached, stopping")
				close(t.stopChan)
				return
			}

			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var msg Message
			if err := json.Unmarshal(line, &msg); err != nil {
				log.Printf("[STDIO] Failed to parse message: %v", err)
				continue
			}
			if msg.JSONRPC == "" {
				msg.JSONRPC = "2.0"
			}

			select {
			case t.msgChan <- &msg:
			case <-t.stopChan:
				return
			}
		}
	}
}

// handleLoop processes incoming messages
func (t *STDIOTransport) handleLoop(ctx context.Context) {
	for {
		select {
		case <-t.stopChan:
			log.Println("[STDIO] Handle loop stopped")
			return
		case msg := <-t.msgChan:
			if msg == nil {
				continue
			}

			// Process message with handler
			resp, err := t.handler.Handle(ctx, msg)
			if err != nil {
				log.Printf("[STDIO] Handler error: %v", err)
				// Send error response if request had ID
				if msg.ID != nil {
					errResp := NewError(msg.ID, -32603, err.Error(), nil)
					if sendErr := t.Send(errResp); sendErr != nil {
						log.Printf("[STDIO] Failed to send error: %v", sendErr)
					}
				}
				continue
			}

			// Send response if present
			if resp != nil {
				if err := t.Send(resp); err != nil {
					log.Printf("[STDIO] Failed to send response: %v", err)
				}
			}
		}
	}
}
