Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 2: CompletionTracker

## Goal

Build the state machine that tracks session lifecycle incrementally from the canonical event stream. This replaces post-hoc file scanning with real-time state tracking.

## Changes

**New file `dispatcher/completion_tracker.go`**:
- `CompletionTracker` struct — consumes `parse.CanonicalEvent` values and maintains session state
- `Observe(event parse.CanonicalEvent)` — state transition on each event
- `Resolve(exitCode int, ctxCancelled bool) SessionOutcome` — called once when process exits; combines accumulated state with exit code to produce the final outcome
- Internal state: `sawInit`, `sawAction`, `sawResult`, `sawComplete`, `sawError`, `lastEventType`, `hasYield` (tracks stage_yield from event stream)

## Data Structures

- `CompletionTracker` — struct with boolean flags and a mutex for concurrent safety (stamp processor goroutine calls `Observe`, main goroutine calls `Resolve`)

## Design Notes

The tracker's `Resolve()` method encodes the decision logic that currently lives in `terminalStatus()`, but with richer input:

1. Context cancelled → `StatusCancelled`
2. Exit code < 0 (signal) AND no `sawComplete` AND no yield → `StatusKilled`
3. `sawComplete` or `sawResult` → `StatusCompleted` (explicit completion signal)
4. `sawAction` (agent did work but no result/complete) → `StatusCompleted` with reason "exited after doing work"
5. `sawInit` only (started but did nothing) → `StatusFailed` with reason "no work produced"
6. No events at all → `StatusFailed` with reason "no events emitted"

`HasDeliverable` is true when `sawComplete` or `sawResult` — meaning at least one turn finished with output.

## Routing

- Provider: `claude`, Model: `claude-opus-4-6` (state machine design requires judgment)

## Verification

- Unit tests for `CompletionTracker` covering every path through `Resolve()`:
  - Clean completion (EventComplete seen)
  - Turn-based completion (EventResult but no EventComplete — Claude pattern)
  - Crash after work (EventAction but no Result/Complete)
  - Crash before any events
  - Signal termination (exit code < 0)
  - Context cancellation override
  - Stage yield override (exit non-zero but yield seen)
- Verify thread safety: concurrent `Observe` + `Resolve` calls
- `go test ./dispatcher/... -race`
