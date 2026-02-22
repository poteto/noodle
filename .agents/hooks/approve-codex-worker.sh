#!/bin/bash
# PreToolUse hook for the codex-worker agent.
# Auto-approves most Bash commands but structurally blocks sleep commands,
# which waste turns and cause stalled polling loops.
input=$(cat)
command=$(printf '%s' "$input" | grep -o '"command":"[^"]*"' | head -1 | sed 's/"command":"//;s/"$//')

# Block sleep commands — use TaskOutput with block: true instead
if printf '%s' "$command" | grep -qE '(^|[;&|] *)sleep '; then
  printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"sleep commands are blocked. Use TaskOutput with block: true, timeout: 600000 to wait for background tasks."}}'
  exit 0
fi

printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","permissionDecisionReason":"Auto-approved for codex-worker"}}'
