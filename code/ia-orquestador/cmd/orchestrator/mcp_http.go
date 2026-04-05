// MCP Streamable HTTP transport for IA_Orquestador.
// Exposes registered skills as standard MCP tools so VS Code / AI agents
// can call them via the unified MCP protocol (2024-11-05 / 2025-03-26).
//
//	POST   /mcp  → JSON-RPC 2.0 (single request, plain JSON or SSE response)
//	DELETE /mcp  → session termination (no-op, returns 200)
//	OPTIONS /mcp → CORS preflight
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/thiscloud/ia-orquestador/internal/executor"
	"github.com/thiscloud/ia-orquestador/pkg/types"
)

// ─────────────────────────────────────────────────────────────────
// JSON-RPC 2.0 wire types
// ─────────────────────────────────────────────────────────────────

type mcpRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *mcpRPCError `json:"error,omitempty"`
}

type mcpRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ─────────────────────────────────────────────────────────────────
// RegisterMCPRoutes mounts the standard MCP endpoint
// ─────────────────────────────────────────────────────────────────

// RegisterMCPRoutes adds POST/DELETE/OPTIONS /mcp to the given mux.
func (o *Orchestrator) RegisterMCPRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/mcp", o.handleMCPDispatch)
}

// handleMCPDispatch routes by HTTP method.
func (o *Orchestrator) handleMCPDispatch(w http.ResponseWriter, r *http.Request) {
	setMCPCORSHeaders(w, r)
	switch r.Method {
	case http.MethodPost:
		o.handleMCPPost(w, r)
	case http.MethodDelete:
		// Stateless server — session termination is a no-op.
		w.WriteHeader(http.StatusOK)
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleMCPPost handles a single JSON-RPC 2.0 request.
func (o *Orchestrator) handleMCPPost(w http.ResponseWriter, r *http.Request) {
	var req mcpRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMCPJSON(w, r, http.StatusBadRequest, mcpError(nil, -32700, "parse error"))
		return
	}

	isNotification := req.ID == nil

	var resp *mcpRPCResponse

	switch req.Method {
	case "initialize":
		if isNotification {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		resp = o.mcpInitialize(&req)

	case "ping":
		if isNotification {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		resp = mcpOK(req.ID, map[string]interface{}{})

	case "tools/list", "mcp.tools.list":
		if isNotification {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		resp = o.mcpToolsList(r.Context(), &req)

	case "tools/call", "mcp.tools.call":
		if isNotification {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		resp = o.mcpToolsCall(r.Context(), &req)

	case "notifications/initialized",
		"notifications/cancelled",
		"notifications/progress":
		// Silently accept all notifications per spec.
		w.WriteHeader(http.StatusAccepted)
		return

	default:
		if isNotification {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		resp = mcpError(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}

	// Echo session ID header if provided (stateless — we don't enforce it).
	if sid := r.Header.Get("Mcp-Session-Id"); sid != "" {
		w.Header().Set("Mcp-Session-Id", sid)
	}

	code := http.StatusOK
	if resp.Error != nil {
		code = http.StatusUnprocessableEntity
	}
	writeMCPJSON(w, r, code, resp)
}

// ─────────────────────────────────────────────────────────────────
// MCP method handlers
// ─────────────────────────────────────────────────────────────────

func (o *Orchestrator) mcpInitialize(req *mcpRPCRequest) *mcpRPCResponse {
	return mcpOK(req.ID, map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]interface{}{
			"name":    "ia-orquestador",
			"version": "0.2.0",
		},
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{"listChanged": false},
		},
	})
}

func (o *Orchestrator) mcpToolsList(ctx context.Context, req *mcpRPCRequest) *mcpRPCResponse {
	skillsList, err := o.skills.List(types.SkillStatusActive, "", 100, 0)
	if err != nil {
		return mcpError(req.ID, -32603, err.Error())
	}

	tools := make([]map[string]interface{}, 0, len(skillsList))
	for _, skill := range skillsList {
		var meta types.SkillMetadata
		_ = json.Unmarshal(skill.Metadata, &meta)

		desc := meta.Description
		if desc == "" {
			desc = fmt.Sprintf("Skill: %s v%s", skill.Name, skill.Version)
		}

		tools = append(tools, map[string]interface{}{
			"name":        skill.Name,
			"description": desc,
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{
						"type":        "string",
						"description": "Input context or instructions for the skill",
					},
					"version": map[string]interface{}{
						"type":        "string",
						"description": "Specific skill version to use (optional, defaults to latest)",
					},
					"timeoutMs": map[string]interface{}{
						"type":        "integer",
						"description": "Execution timeout in milliseconds (default 60000)",
					},
				},
				"required": []string{"input"},
			},
		})
	}

	return mcpOK(req.ID, map[string]interface{}{"tools": tools})
}

func (o *Orchestrator) mcpToolsCall(ctx context.Context, req *mcpRPCRequest) *mcpRPCResponse {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return mcpError(req.ID, -32602, "invalid params: "+err.Error())
	}
	if params.Name == "" {
		return mcpError(req.ID, -32602, "name is required")
	}

	version, _ := params.Arguments["version"].(string)
	skill, err := o.skills.GetByName(params.Name, version)
	if err != nil {
		return mcpError(req.ID, -32602, fmt.Sprintf("skill not found: %s", params.Name))
	}

	// Build input JSON from arguments.
	inputJSON, _ := json.Marshal(params.Arguments)

	timeoutMs := 60000
	if t, ok := params.Arguments["timeoutMs"].(float64); ok && t > 0 {
		timeoutMs = int(t)
	}

	res, err := executor.Execute(ctx, skill, inputJSON, timeoutMs)
	if err != nil {
		return mcpError(req.ID, -32603, err.Error())
	}

	outputStr := ""
	switch v := res.Output.(type) {
	case string:
		outputStr = v
	default:
		b, _ := json.MarshalIndent(res.Output, "", "  ")
		outputStr = string(b)
	}

	return mcpOK(req.ID, map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": outputStr},
		},
	})
}

// ─────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────

func mcpOK(id interface{}, result interface{}) *mcpRPCResponse {
	return &mcpRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func mcpError(id interface{}, code int, msg string) *mcpRPCResponse {
	return &mcpRPCResponse{JSONRPC: "2.0", ID: id, Error: &mcpRPCError{Code: code, Message: msg}}
}

// writeMCPJSON writes a JSON response, or SSE-wrapped if the client wants SSE.
func writeMCPJSON(w http.ResponseWriter, r *http.Request, code int, v interface{}) {
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/event-stream") {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		b, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", b)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func setMCPCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Api-Key, Mcp-Session-Id")
	w.Header().Set("Access-Control-Max-Age", "86400")
}
