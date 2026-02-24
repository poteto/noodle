---
id: 31
created: 2026-02-24
status: ready
---

# Structured Loop Logging

## Context

The loop package produces zero structured output. All lifecycle events — dispatch, completion, retry, failure, state transitions, control commands, runtime repair — are silent. The only output is 3 ad-hoc `fmt.Fprintf(os.Stderr, ...)` calls: a session failure message in `cook.go`, a nil-entry warning in `pending_review.go`, and a sprite config warning in `defaults.go`. Errors return up the call stack and most are handled by the runtime repair system, but operators have no way to observe what the loop is doing without reading filesystem state.

This plan adds structured logging to the loop using Go's stdlib `log/slog`. Every significant lifecycle event gets a log line with consistent key-value attributes. The logger writes to stderr in text format by default. The 3 existing `fmt.Fprintf` calls are replaced with slog equivalents.

## Scope

**In:**
- `log/slog.Logger` field on the `Loop` struct, injected via `Dependencies`
- Default to `slog.New(slog.NewTextHandler(os.Stderr, nil))` when not provided
- Log lines for: dispatch (cook, prioritize, runtime repair), completion (success, retry, failure, merge, park-for-review), queue mutations (consume-next, bootstrap-prioritize, skip, normalize), state transitions, control commands, runtime repair lifecycle
- Replace the 3 existing `fmt.Fprintf(os.Stderr, ...)` calls with slog
- Tests that verify log output for key events

**Out:**
- JSON output format (text is sufficient for stderr; JSON can be a follow-up if needed for log aggregation)
- Log levels configuration / runtime log-level switching
- TUI integration (the TUI reads filesystem state, not log output)
- Logging in packages outside `loop/` (dispatcher, mise, monitor, etc.)
- File-based log output (stderr only)

## Constraints

- Use `log/slog` from Go stdlib. No new dependencies.
- Output to stderr. Do not create log files.
- Logger must be injectable for testing (pass `*slog.Logger` or use `slog.New` with a custom handler to capture output in tests).
- Keep log volume reasonable: one line per event, not per iteration. Don't log "nothing to do" on idle cycles.
- Use consistent attribute keys across all log lines: `item`, `session`, `state`, `scope`, `action`, `attempt`, `reason`, `worktree`.

## Applicable skills

- `go-best-practices` — Go patterns, slog usage, testing conventions

## Phases

- [[plans/31-structured-loop-logging/phase-01-logger-type-and-interface]]
- [[plans/31-structured-loop-logging/phase-02-inject-logger-into-loop]]
- [[plans/31-structured-loop-logging/phase-03-log-dispatch-and-completion-events]]
- [[plans/31-structured-loop-logging/phase-04-log-queue-mutations-and-state-transitions]]
- [[plans/31-structured-loop-logging/phase-05-log-retry-failure-and-repair-events]]
- [[plans/31-structured-loop-logging/phase-06-tests]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```
