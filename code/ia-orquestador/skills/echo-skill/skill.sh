#!/bin/bash
# Echo Skill - Simple example skill that returns input

# Read JSON input from stdin
INPUT=$(cat)

# Extract text field
TEXT=$(echo "$INPUT" | jq -r '.text // "No text provided"')

# Output JSON response
cat <<EOF
{
  "status": "success",
  "output": {
    "echoed": "$TEXT",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "skill": "echo-skill"
  }
}
EOF
