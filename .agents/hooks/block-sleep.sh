#!/bin/sh
# block-sleep.sh — PreToolUse:Bash hook
# Blocks bare sleep commands that waste agent turns.

input=$(cat)
command=$(printf '%s' "$input" | grep -o '"command":"[^"]*"' | head -1 | sed 's/"command":"//;s/"$//')

case "$command" in
  sleep\ *|sleep)
    printf '{"decision":"block","reason":"sleep commands waste agent turns — use run_in_background or TaskOutput instead"}\n'
    ;;
esac
