#!/bin/bash
# bulk-register-skills.sh
# Registers ALL skills from /opt/ia-orquestador/skills/ into the SQLite DB.
# Handles two manifest formats:
#   manifest.json  — echo-skill style (has entrypoint, status fields)
#   metadata.json  — SDD/DotNet style (has phase, description, author fields)
#
# Usage:
#   ./scripts/bulk-register-skills.sh [DB_PATH] [SKILLS_BASE_DIR]
#
# Defaults:
#   DB_PATH       = ./orchestrator.db
#   SKILLS_BASE   = /opt/ia-orquestador/skills

set -euo pipefail

DB_PATH="${1:-./orchestrator.db}"
SKILLS_BASE="${2:-/opt/ia-orquestador/skills}"

if [ ! -f "$DB_PATH" ]; then
  echo "ERROR: Database not found at $DB_PATH"
  exit 1
fi

if [ ! -d "$SKILLS_BASE" ]; then
  echo "ERROR: Skills directory not found at $SKILLS_BASE"
  exit 1
fi

command -v jq >/dev/null 2>&1 || { echo "ERROR: jq not installed"; exit 1; }
command -v uuidgen >/dev/null 2>&1 || { echo "ERROR: uuidgen not installed"; exit 1; }
command -v sqlite3 >/dev/null 2>&1 || { echo "ERROR: sqlite3 not installed"; exit 1; }

echo "=== Bulk Skill Registration ==="
echo "DB:     $DB_PATH"
echo "Skills: $SKILLS_BASE"
echo ""

registered=0
skipped=0
failed=0

register_skill() {
  local manifest="$1"
  local skill_dir
  skill_dir=$(dirname "$manifest")
  local manifest_file
  manifest_file=$(basename "$manifest")

  local NAME VERSION TYPE ENTRYPOINT PATH_VAL STATUS METADATA NOW SKILL_ID

  # ── manifest.json format (echo-skill) ─────────────────────────────────────
  if [ "$manifest_file" = "manifest.json" ]; then
    NAME=$(jq -r '.name // empty' "$manifest")
    VERSION=$(jq -r '.version // "1.0.0"' "$manifest")
    TYPE=$(jq -r '.type // "sdd"' "$manifest")
    ENTRYPOINT=$(jq -r '.entrypoint // ""' "$manifest")
    PATH_VAL=$(jq -r '.path // ""' "$manifest")
    STATUS=$(jq -r '.status // "active"' "$manifest")
    METADATA=$(jq -c '.metadata // {}' "$manifest")

  # ── metadata.json format (SDD / DotNet) ───────────────────────────────────
  elif [ "$manifest_file" = "metadata.json" ]; then
    NAME=$(jq -r '.name // empty' "$manifest")
    VERSION=$(jq -r '.version // "1.0.0"' "$manifest")

    # Infer skill type from directory path
    if echo "$skill_dir" | grep -q "/sdd/"; then
      TYPE="sdd"
    elif echo "$skill_dir" | grep -q "/dotnet/"; then
      TYPE="dotnet"
    else
      TYPE="sdd"
    fi

    # For SDD/DotNet skills the entrypoint is the SKILL.md file
    if [ -f "$skill_dir/SKILL.md" ]; then
      ENTRYPOINT="$skill_dir/SKILL.md"
    else
      ENTRYPOINT=""
    fi
    PATH_VAL="$skill_dir"
    STATUS="active"

    # Build metadata JSON from metadata.json fields
    DESCRIPTION=$(jq -r '.description // ""' "$manifest")
    TAGS=$(jq -c '.tags // []' "$manifest")
    PHASE=$(jq -r '.phase // ""' "$manifest")
    AUTHOR=$(jq -r '.author // ""' "$manifest")

    METADATA=$(jq -cn \
      --arg desc "$DESCRIPTION" \
      --argjson tags "$TAGS" \
      --arg phase "$PHASE" \
      --arg author "$AUTHOR" \
      '{
        capabilities: ($tags | if length > 0 then . else [] end),
        tags: $tags,
        description: $desc,
        extra: ({ phase: $phase, author: $author } | with_entries(select(.value != "")))
      }')
  else
    echo "  SKIP: Unknown manifest format: $manifest"
    skipped=$((skipped + 1))
    return
  fi

  # Validate mandatory fields
  if [ -z "$NAME" ]; then
    echo "  SKIP: Missing name in $manifest"
    skipped=$((skipped + 1))
    return
  fi

  # Check if skill already exists (by name + version)
  EXISTING=$(sqlite3 "$DB_PATH" "SELECT id FROM skills WHERE name='$NAME' AND version='$VERSION' LIMIT 1;")
  if [ -n "$EXISTING" ]; then
    echo "  SKIP (already registered): $NAME v$VERSION"
    skipped=$((skipped + 1))
    return
  fi

  SKILL_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')
  NOW=$(date +%s)

  sqlite3 "$DB_PATH" <<SQL
INSERT INTO skills (id, name, version, type, entrypoint, path, metadata, status, created_at, updated_at)
VALUES (
  '$SKILL_ID',
  '$(echo "$NAME" | sed "s/'/''/g")',
  '$VERSION',
  '$TYPE',
  '$(echo "$ENTRYPOINT" | sed "s/'/''/g")',
  '$(echo "$PATH_VAL" | sed "s/'/''/g")',
  '$(echo "$METADATA" | sed "s/'/''/g")',
  '$STATUS',
  $NOW,
  $NOW
);
SQL

  if [ $? -eq 0 ]; then
    echo "  OK: $NAME v$VERSION [$TYPE] → $SKILL_ID"
    registered=$((registered + 1))
  else
    echo "  FAIL: $NAME"
    failed=$((failed + 1))
  fi
}

# ── Discover and register all skills ─────────────────────────────────────────

# Priority: manifest.json first, then metadata.json
for manifest in "$SKILLS_BASE"/*/manifest.json "$SKILLS_BASE"/*/*/manifest.json; do
  [ -f "$manifest" ] && register_skill "$manifest"
done

for manifest in "$SKILLS_BASE"/*/metadata.json "$SKILLS_BASE"/*/*/metadata.json; do
  [ -f "$manifest" ] && register_skill "$manifest"
done

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "=== Done ==="
echo "  Registered : $registered"
echo "  Skipped    : $skipped (already in DB or unknown format)"
echo "  Failed     : $failed"
echo ""
echo "Active skills in DB:"
sqlite3 "$DB_PATH" "SELECT name, version, type, status FROM skills WHERE status='active' ORDER BY type, name;" | column -t -s '|'
