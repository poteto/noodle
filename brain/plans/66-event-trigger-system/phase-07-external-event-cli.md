Back to [[plans/66-event-trigger-system/overview]]

# Phase 7 — External Event CLI

## Goal

Let external tools inject events into the loop's event stream via `noodle event emit <type> [payload]`. CI pipelines, GitHub Actions, webhooks, cron jobs, or any process that can run a shell command can now surface events to the schedule agent. The agent decides what to do — same as internal lifecycle events.

This makes the event system extensible without adding an HTTP server, webhook receiver, or adapter hooks. The CLI command appends to the same `loop-events.ndjson` file, using the same `EventWriter` with sequence assignment and file locking. External events flow through the mise brief identically to internal events.

## Changes

- **`cmd_event.go`** (new) — `noodle event` parent command with `emit` subcommand:
  - `noodle event emit <type> [--payload <json>]` — validates that type is a non-empty string, marshals the optional payload, calls `EventWriter.Emit()`.
  - The command reads `runtimeDir` from the project's `.noodle/` directory (same as other commands).
  - No type validation beyond non-empty — external events use arbitrary types (e.g., `ci.failed`, `deploy.completed`, `pr.merged`). The schedule agent interprets them.
- **`root.go`** — add `newEventCmd(&app)` to the command tree.
- **`cmd_event_test.go`** (new) — test that `noodle event emit ci.failed --payload '{"url":"..."}'` appends a valid NDJSON line with correct type, payload, and monotonic sequence.

## Data Structures

No new types — reuses `EventWriter` and `LoopEvent` from phase 1. External events have the same shape as internal events. The type field is an arbitrary string, not constrained to the `LoopEventType` enum — this is intentional.

## Routing

Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

```bash
go test ./... && go vet ./...
```

- `noodle event emit ci.failed` appends a valid line to `loop-events.ndjson`
- `noodle event emit deploy.completed --payload '{"env":"prod"}'` includes the payload
- The emitted event appears in the next mise brief's `recent_events`
- Sequence numbers remain monotonic when mixing internal and external events
- Missing `.noodle/` directory or uninitialized project produces a clear error
