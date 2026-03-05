Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 1: SessionOutcome Type

## Goal

Define the foundational data type that replaces string-based session status. `SessionOutcome` is a structured type that captures not just pass/fail but the classification reason and whether a deliverable was produced.

## Changes

**`dispatcher/types.go`** — Add `SessionOutcome` type:
- `SessionOutcome` with fields: `Status` (enum: completed/failed/cancelled/killed), `Reason` (human-readable classification reason), `HasDeliverable` (bool — did the session produce actionable output), `ExitCode` (int — raw process exit code for diagnostics)
- `SessionStatus` enum type with constants: `StatusCompleted`, `StatusFailed`, `StatusCancelled`, `StatusKilled`
- Method `IsTerminal() bool` on `SessionStatus`

**`dispatcher/types.go`** — Update `Session` interface:
- Add `Outcome() SessionOutcome` method (returns zero value while running, populated after `Done()` closes)
- Keep `Status() string` temporarily for backward compat during migration (removed in phase 10)

## Data Structures

- `SessionStatus` — string enum with defined constants
- `SessionOutcome` — struct with Status, Reason, HasDeliverable, ExitCode

## Routing

- Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

- `go build ./dispatcher/...` — types compile
- `go vet ./dispatcher/...` — no issues
- Unit tests: `SessionOutcome` zero value is non-terminal, `IsTerminal()` returns correct values for each status
