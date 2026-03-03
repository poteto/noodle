# Claude Code NDJSON Protocol

Claude Code CLI supports programmatic control via NDJSON (newline-delimited JSON). Two transports are available:

## Transports

1. **stdin/stdout pipes** — spawn `claude` as a child process with `--input-format stream-json --output-format stream-json --print`
2. **WebSocket via `--sdk-url`** — hidden flag (`.hideHelp()` in Commander.js). CLI becomes a WebSocket client connecting to the specified URL. Same NDJSON protocol over WebSocket frames.

Both transports use the exact same message format — the same protocol used internally by `@anthropic-ai/claude-agent-sdk`.

## Key Flags

```bash
claude --output-format stream-json \   # NDJSON on stdout
       --input-format stream-json \    # Accept NDJSON on stdin (multi-turn)
       --verbose \                     # Include stream_event messages
       --print \                       # Non-interactive (no TUI)
       --permission-prompt-tool stdio \ # Route permissions through stdin/stdout
       -p ""                           # Empty initial prompt
```

## Message Types

| Type | Direction | Purpose |
|------|-----------|---------|
| `system` (subtype `init`) | CLI → app | Session ID, tools, model |
| `user` | app → CLI | Send prompt |
| `assistant` | CLI → app | LLM response with content blocks |
| `stream_event` | CLI → app | Token-by-token streaming (with `--verbose`) |
| `result` | CLI → app | Turn complete — cost, usage, success/error |
| `control_request` (subtype `can_use_tool`) | CLI → app | Permission request for tool use |
| `control_response` | app → CLI | Approve/deny tool use |
| `keep_alive` | bidirectional | Heartbeat |

## Interrupt / Abort

There is **no NDJSON "interrupt" message type**. The SDK's `query.interrupt()` uses `AbortController.abort()` which kills the child process. To resume:

1. Kill the process
2. Keep the `session_id` (from the `system` init message)
3. Re-spawn with `--resume <session_id>` on next send

This means interrupt = kill + resume. The conversation context is preserved server-side by session ID.

## Line Size Gotcha

Individual NDJSON lines can be very large — multi-MB — especially when:
- Codex workers return large results (entire file contents in tool_result)
- Context compaction produces large system messages
- Subagent output is embedded in tool_result blocks

**The noodle `stamp` sidecar uses `bufio.Scanner` with a 64MB buffer** (`64<<20`). The original 1MB buffer caused `token too long` errors that killed the pipe and crashed manager sessions. Any code processing Claude stream-json output must handle arbitrarily large lines.

## Noodle Uses

Noodle uses stdin/stdout pipes to spawn and communicate with Claude sessions. The `stamp` sidecar monitors the NDJSON stream for cost, tool use, and outcome data.

See also [[principles/boundary-discipline]], [[codebase/claude-print-flag-gotchas]]
