#!/bin/bash
# Register a skill in the orchestrator database

DB_PATH="${1:-./orchestrator.db}"
SKILL_MANIFEST="${2:-./skills/echo-skill/manifest.json}"

if [ ! -f "$SKILL_MANIFEST" ]; then
  echo "Error: Skill manifest not found: $SKILL_MANIFEST"
  exit 1
fi

# Parse manifest
SKILL_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')
NAME=$(jq -r '.name' "$SKILL_MANIFEST")
VERSION=$(jq -r '.version' "$SKILL_MANIFEST")
TYPE=$(jq -r '.type' "$SKILL_MANIFEST")
ENTRYPOINT=$(jq -r '.entrypoint' "$SKILL_MANIFEST")
PATH_VAL=$(jq -r '.path' "$SKILL_MANIFEST")
METADATA=$(jq -c '.metadata' "$SKILL_MANIFEST")
STATUS=$(jq -r '.status // "inactive"' "$SKILL_MANIFEST")
NOW=$(date +%s)

echo "Registering skill:"
echo "  ID:         $SKILL_ID"
echo "  Name:       $NAME"
echo "  Version:    $VERSION"
echo "  Type:       $TYPE"
echo "  Entrypoint: $ENTRYPOINT"
echo "  Status:     $STATUS"
echo ""

# Insert into database
sqlite3 "$DB_PATH" <<SQL
INSERT INTO skills (id, name, version, type, entrypoint, path, metadata, status, created_at, updated_at)
VALUES (
  '$SKILL_ID',
  '$NAME',
  '$VERSION',
  '$TYPE',
  '$ENTRYPOINT',
  '$PATH_VAL',
  '$METADATA',
  '$STATUS',
  $NOW,
  $NOW
);
SQL

if [ $? -eq 0 ]; then
  echo "✓ Skill registered successfully!"
  echo "  Skill ID: $SKILL_ID"
else
  echo "✗ Failed to register skill"
  exit 1
fi
