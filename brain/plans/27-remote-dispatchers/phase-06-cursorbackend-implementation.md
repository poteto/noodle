Back to [[plans/27-remote-dispatchers/overview]]

# Phase 6: CursorBackend stub

**Routing:** `codex` / `gpt-5.3-codex` — mechanical stub implementation

## References

- Cursor Cloud Agent API: https://cursor.com/docs/cloud-agent/api/endpoints

## Goal

Create a stub `CursorBackend` that satisfies the `PollingBackend` interface. All methods log "cursor backend not implemented" and return errors. This validates the polling infrastructure from Phase 4 has a real type wired through the factory, while deferring the actual Cursor API integration to a future plan.

The MVP focus is the Sprites streaming path. The polling dispatcher and its interface are still fully implemented (Phase 4) — this phase just defers the Cursor-specific HTTP plumbing.

## Data structures

- `CursorBackend` struct — holds `CursorConfig`

## Changes

**`dispatcher/cursor_backend.go` (new)**

Stub implementation — every method returns `fmt.Errorf("cursor backend not implemented")` and logs to stderr. Satisfies `PollingBackend` at compile time so the factory can register it and the wiring compiles end-to-end.

## Verification

### Static
- Compiles, passes vet
- `var _ PollingBackend = (*CursorBackend)(nil)` compile-time check

### Runtime
- Unit test: each method returns an error containing "not implemented"
