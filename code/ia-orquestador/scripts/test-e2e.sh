#!/bin/bash
# test-e2e.sh
# End-to-end validation of the MCP Orchestrator:
#   1. Health check
#   2. Initialize session
#   3. List tools (assert > 0)
#   4. Call echo-skill (local-exec backend)
#   5. Call sdd-init (skill-content backend)
#   6. Call non-existent tool (assert error)
#
# Usage:
#   ./scripts/test-e2e.sh [HOST]
# Default HOST: http://localhost:7440

set -euo pipefail

HOST="${1:-http://localhost:7440}"
PASS=0
FAIL=0

# ── helpers ───────────────────────────────────────────────────────────────────

check() {
  local label="$1"
  local result="$2"   # "pass" | "fail"
  local detail="${3:-}"
  if [ "$result" = "pass" ]; then
    echo "  ✓ $label"
    PASS=$((PASS + 1))
  else
    echo "  ✗ $label${detail:+: $detail}"
    FAIL=$((FAIL + 1))
  fi
}

mcp_call() {
  local method="$1"
  local params="$2"
  local id="${3:-1}"
  curl -s -X POST "$HOST/mcp/jsonrpc" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"id\":$id,\"method\":\"$method\",\"params\":$params}"
}

# ── 1. Health check ───────────────────────────────────────────────────────────
echo ""
echo "=== E2E Test: $HOST ==="
echo ""
echo "--- 1. Health check ---"
HEALTH=$(curl -s -o /dev/null -w "%{http_code}" "$HOST/healthz")
[ "$HEALTH" = "200" ] && check "GET /healthz → 200" pass || check "GET /healthz → 200" fail "HTTP $HEALTH"

# ── 2. Initialize session ─────────────────────────────────────────────────────
echo ""
echo "--- 2. mcp.initialize ---"
INIT=$(mcp_call "mcp.initialize" '{"clientId":"e2e-test","protocolVersion":"2024-11-05"}')
SESSION_ID=$(echo "$INIT" | jq -r '.result.sessionId // empty')
SERVER_VER=$(echo "$INIT" | jq -r '.result.serverVersion // empty')

[ -n "$SESSION_ID" ] && check "sessionId returned" pass || check "sessionId returned" fail "$(echo "$INIT" | jq -r '.error.message // "no sessionId"')"
[ -n "$SERVER_VER" ] && check "serverVersion returned" pass || check "serverVersion returned" fail

# ── 3. List tools ─────────────────────────────────────────────────────────────
echo ""
echo "--- 3. mcp.tools.list ---"
LIST=$(mcp_call "mcp.tools.list" '{"paging":{"limit":50}}')
TOTAL=$(echo "$LIST" | jq -r '.result.total // 0')
TOOLS_COUNT=$(echo "$LIST" | jq '.result.tools | length // 0')

[ "$TOTAL" -gt 0 ] && check "total > 0 ($TOTAL skills)" pass || check "total > 0" fail "got $TOTAL"
[ "$TOOLS_COUNT" -gt 0 ] && check "tools array not empty ($TOOLS_COUNT items)" pass || check "tools array not empty" fail

# ── 4. Call echo-skill (local-exec) ───────────────────────────────────────────
echo ""
echo "--- 4. mcp.tools.call: echo-skill ---"
ECHO_ID=$(mcp_call "mcp.tools.list" '{}' 2 | jq -r '.result.tools[] | select(.name == "echo-skill") | .id' | head -1)

if [ -n "$ECHO_ID" ]; then
  ECHO_RESP=$(mcp_call "mcp.tools.call" "{\"toolId\":\"$ECHO_ID\",\"sessionId\":\"$SESSION_ID\",\"input\":{\"text\":\"hello-e2e\"}}" 3)
  ECHO_MODE=$(echo "$ECHO_RESP" | jq -r '.result.mode // empty')
  ECHO_STATUS=$(echo "$ECHO_RESP" | jq -r '.result.status // empty')

  [ "$ECHO_STATUS" = "completed" ] && check "echo-skill status=completed" pass || check "echo-skill status=completed" fail "$ECHO_STATUS"
  [ "$ECHO_MODE" = "local-exec" ] && check "echo-skill mode=local-exec" pass || check "echo-skill mode (got $ECHO_MODE)" fail
else
  check "echo-skill found in registry" fail "not in DB — run bulk-register-skills.sh first"
fi

# ── 5. Call sdd-init (skill-content) ─────────────────────────────────────────
echo ""
echo "--- 5. mcp.tools.call: sdd-init ---"
SDD_ID=$(mcp_call "mcp.tools.list" '{}' 4 | jq -r '.result.tools[] | select(.name == "sdd-init") | .id' | head -1)

if [ -n "$SDD_ID" ]; then
  SDD_RESP=$(mcp_call "mcp.tools.call" "{\"toolId\":\"$SDD_ID\",\"sessionId\":\"$SESSION_ID\",\"input\":{\"project_root\":\"/opt/ia-orquestador\"}}" 5)
  SDD_MODE=$(echo "$SDD_RESP" | jq -r '.result.mode // empty')
  SDD_STATUS=$(echo "$SDD_RESP" | jq -r '.result.status // empty')
  SDD_CONTENT=$(echo "$SDD_RESP" | jq -r '.result.output.content // empty' | wc -c)

  [ "$SDD_STATUS" = "completed" ] && check "sdd-init status=completed" pass || check "sdd-init status=completed" fail "$SDD_STATUS"
  { [ "$SDD_MODE" = "skill-content" ] || [ "$SDD_MODE" = "skill-metadata" ]; } && \
    check "sdd-init mode=$SDD_MODE" pass || check "sdd-init mode (got $SDD_MODE)" fail
  [ "$SDD_CONTENT" -gt 10 ] && check "sdd-init content not empty (${SDD_CONTENT} chars)" pass || check "sdd-init content empty" fail
else
  check "sdd-init found in registry" fail "not in DB — run bulk-register-skills.sh first"
fi

# ── 6. Call non-existent tool (assert error) ──────────────────────────────────
echo ""
echo "--- 6. mcp.tools.call: non-existent skill ---"
BAD_RESP=$(mcp_call "mcp.tools.call" '{"toolId":"00000000-0000-0000-0000-000000000000","sessionId":"fake"}' 6)
BAD_ERR=$(echo "$BAD_RESP" | jq -r '.error.message // empty')
[ -n "$BAD_ERR" ] && check "unknown toolId returns JSON-RPC error" pass || check "unknown toolId returns JSON-RPC error" fail "no error in response"

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "=================================="
echo "  PASSED: $PASS"
echo "  FAILED: $FAIL"
echo "=================================="
echo ""

[ "$FAIL" -eq 0 ] && echo "ALL TESTS PASSED ✓" && exit 0
echo "SOME TESTS FAILED ✗" && exit 1
