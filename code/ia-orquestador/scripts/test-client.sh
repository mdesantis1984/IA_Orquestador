#!/bin/bash
# Example MCP client script to test the orchestrator

ORCHESTRATOR_URL="http://localhost:8080"

echo "=== MCP Orchestrator Test Client ==="
echo ""

# 1. Initialize session
echo "→ Initializing session..."
SESSION_RESP=$(curl -s -X POST "${ORCHESTRATOR_URL}/mcp/jsonrpc" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "mcp.initialize",
    "params": {
      "clientId": "test-client-bash",
      "protocolVersion": "1.0",
      "clientCapabilities": {"sse": true, "streaming": true}
    }
  }')

echo "$SESSION_RESP" | jq .

SESSION_ID=$(echo "$SESSION_RESP" | jq -r '.result.sessionId')
echo "✓ Session ID: $SESSION_ID"
echo ""

# 2. List available tools
echo "→ Listing available tools..."
TOOLS_RESP=$(curl -s -X POST "${ORCHESTRATOR_URL}/mcp/jsonrpc" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "mcp.tools.list",
    "params": {
      "paging": {"limit": 10, "offset": 0}
    }
  }')

echo "$TOOLS_RESP" | jq .
echo ""

# 3. Call a tool (if any available)
TOOL_ID=$(echo "$TOOLS_RESP" | jq -r '.result.tools[0].id // empty')

if [ -n "$TOOL_ID" ]; then
  echo "→ Calling tool: $TOOL_ID"
  CALL_RESP=$(curl -s -X POST "${ORCHESTRATOR_URL}/mcp/jsonrpc" \
    -H "Content-Type: application/json" \
    -d "{
      \"jsonrpc\": \"2.0\",
      \"id\": 3,
      \"method\": \"mcp.tools.call\",
      \"params\": {
        \"toolId\": \"$TOOL_ID\",
        \"sessionId\": \"$SESSION_ID\",
        \"input\": {\"text\": \"Hello from test client\"},
        \"callOptions\": {\"stream\": false}
      }
    }")
  
  echo "$CALL_RESP" | jq .
  echo ""
else
  echo "⚠ No tools available to call"
  echo ""
fi

# 4. Health check
echo "→ Health check..."
HEALTH=$(curl -s "${ORCHESTRATOR_URL}/healthz")
echo "Health: $HEALTH"
echo ""

echo "=== Test Complete ==="
