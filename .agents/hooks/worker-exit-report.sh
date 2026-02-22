#!/bin/sh
# Stop hook: writes a JSON exit report when an agent (worker or manager) exits.
#
# Workers write to:  ~/.director/workers/<agent>-<timestamp>.json
# Managers write to: ~/.director/sessions/<session>/ (derived from hook input)
#
# Reads hook input JSON from stdin. Fails silently if stdin is empty or malformed.
# POSIX-compatible — no bash 4+ features.

set -e

# Read all of stdin; if empty, exit silently
INPUT=$(cat 2>/dev/null) || true
if [ -z "$INPUT" ]; then
  exit 0
fi

# Extract fields from the hook input JSON using lightweight parsing.
# The Stop hook receives JSON with session metadata.
# We use parameter expansion and sed — no jq dependency required.

# Helper: extract a JSON string value by key (simple flat-object parser)
json_str() {
  printf '%s' "$INPUT" | sed -n 's/.*"'"$1"'"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1
}

AGENT_NAME=$(json_str "agent_name")
SESSION_ID=$(json_str "session_id")

# Fallback: if agent_name is empty, try to derive from environment or use "unknown"
if [ -z "$AGENT_NAME" ]; then
  AGENT_NAME="${CLAUDE_AGENT_NAME:-unknown}"
fi

# ISO-8601 timestamp (POSIX date)
EXITED_AT=$(date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date +"%Y-%m-%dT%H:%M:%S" 2>/dev/null || echo "unknown")

# Build the exit report JSON
REPORT=$(printf '{
  "agent": "%s",
  "exited_at": "%s",
  "last_action": "normal exit",
  "errors": []
}' "$AGENT_NAME" "$EXITED_AT")

# Determine output path.
# If we have a session ID and the session directory exists, write there (manager path).
# Otherwise, write to CWD (worker path).
SESSION_DIR=""
if [ -n "$SESSION_ID" ]; then
  SESSION_DIR="$HOME/.director/sessions/$SESSION_ID"
fi

if [ -n "$SESSION_DIR" ] && [ -d "$SESSION_DIR" ]; then
  # Manager exit report: write to session directory
  OUTPUT_PATH="$SESSION_DIR/mgr-${AGENT_NAME}-exit.json"
else
  # Worker exit report: write to ~/.director/workers/
  WORKER_DIR="$HOME/.director/workers"
  mkdir -p "$WORKER_DIR" 2>/dev/null || true
  OUTPUT_PATH="${WORKER_DIR}/${AGENT_NAME}-$(date +%s 2>/dev/null || echo $$).json"
fi

# Write the report, fail silently on error
printf '%s\n' "$REPORT" > "$OUTPUT_PATH" 2>/dev/null || true
