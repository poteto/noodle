#!/bin/bash
# Detect user interrupts by checking the transcript.
# If the last assistant message has a tool_use without a matching
# tool_result, the turn was interrupted (Stop hook never fired).
# Fires on: UserPromptSubmit

INPUT=$(cat)
TRANSCRIPT=$(echo "$INPUT" | jq -r '.transcript_path // empty')

[ -z "$TRANSCRIPT" ] || [ ! -f "$TRANSCRIPT" ] && exit 0

# Get the type of the last transcript entry
LAST_TYPE=$(tail -1 "$TRANSCRIPT" | jq -r '.type // empty' 2>/dev/null)

# If the last entry is an assistant message, the agent was mid-turn
# when it stopped. Normal stops are followed by user messages or
# tool results — an assistant message at the end means interrupt.
if [ "$LAST_TYPE" = "assistant" ]; then
  echo "Note: You were interrupted during your previous turn. After responding, add a brief todo reflecting on why intervention was needed — system gap (agent should have known) or context gap (agent lacked information the user had)."
fi
