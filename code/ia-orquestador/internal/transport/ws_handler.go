// Package transport — WebSocket handler for HTTPTransport.
// Registers at /mcp/ws and provides full-duplex JSON-RPC 2.0 over WebSocket.
package transport

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	// Allow any origin; enforce auth at the X-Api-Key middleware layer.
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// handleWebSocket upgrades an HTTP connection to WebSocket and processes
// JSON-RPC 2.0 messages bidirectionally until the client disconnects.
func (t *HTTPTransport) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	remoteAddr := r.RemoteAddr
	log.Printf("[WS] Client connected: %s", remoteAddr)
	defer log.Printf("[WS] Client disconnected: %s", remoteAddr)

	for {
		// Block until a message arrives or the connection closes.
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[WS] Read error from %s: %v", remoteAddr, err)
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			errMsg := NewError(nil, -32700, "Parse error", nil)
			writeWSMessage(conn, errMsg, remoteAddr)
			continue
		}

		resp, err := t.handler.Handle(r.Context(), &msg)
		if err != nil {
			log.Printf("[WS] Handler error: %v", err)
			errMsg := NewError(msg.ID, -32603, err.Error(), nil)
			writeWSMessage(conn, errMsg, remoteAddr)
			continue
		}

		if resp != nil {
			writeWSMessage(conn, resp, remoteAddr)
		}
	}
}

func writeWSMessage(conn *websocket.Conn, msg *Message, remoteAddr string) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] Marshal error to %s: %v", remoteAddr, err)
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("[WS] Write error to %s: %v", remoteAddr, err)
	}
}
