Back to [[plans/27-remote-dispatchers/overview]]

# Phase 6: CursorBackend stub

**Routing:** `codex` / `gpt-5.3-codex` — mechanical stub implementation

## References

- Cursor Cloud Agent API: https://cursor.com/docs/cloud-agent/api/endpoints

## Goal

Create a stub `CursorBackend` that satisfies the `PollingBackend` interface at compile time. All methods return "not implemented" errors. The stub is **not** registered in the factory or advertised in `available_runtimes` — it exists only to prove the `PollingBackend` interface compiles with a real type and to provide a starting point for a future Cursor implementation plan.

The MVP focus is the Sprites streaming path. The polling dispatcher and its interface are still fully implemented (Phase 4) — this phase just defers the Cursor-specific HTTP plumbing.

## Data structures

- `CursorBackend` struct — holds `CursorConfig`

## Changes

**`dispatcher/cursor_backend.go` (new)**

Stub implementation — every method returns `fmt.Errorf("cursor backend not implemented")` and logs to stderr. Satisfies `PollingBackend` at compile time only. Not registered in the factory (Phase 2/11) or advertised in `available_runtimes` (Phase 8).

## Verification

### Static
- Compiles, passes vet
- `var _ PollingBackend = (*CursorBackend)(nil)` compile-time check

### Runtime
- Unit test: each method returns an error containing "not implemented"
