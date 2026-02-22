#!/bin/bash
# Inject brain index at session start so the agent knows what knowledge is available.

BRAIN_INDEX="$CLAUDE_PROJECT_DIR/brain/index.md"

if [ -f "$BRAIN_INDEX" ]; then
  echo "Brain vault index — read the relevant files before acting:"
  echo ""
  cat "$BRAIN_INDEX"
fi
