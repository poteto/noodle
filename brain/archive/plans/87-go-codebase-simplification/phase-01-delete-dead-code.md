Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 1: Delete Dead Backend Abstractions

## Goal

Remove unused backend interfaces and stub implementations from the dispatcher package. These were scaffolded for future Cursor support but are never referenced in production code. Per subtract-before-you-add, delete before building.

Note: Todo 69 (Cursor dispatcher) will redesign these from first principles when implemented. Keeping dead stubs constrains future design.

## Changes

**Critical: `SyncResult` in `backend_types.go` is live production code.** It's used in `sprites_dispatcher.go:313`, `sprites_session.go:118`, and aliased in `runtime/runtime.go:20`. Do NOT delete `backend_types.go` wholesale.

- **`dispatcher/backend_types.go`** — relocate `SyncResult` and its constants to `dispatcher/sync_result.go`. Then delete the remaining dead types (`StreamStartConfig`, `StreamHandle`, `RemoteStatus`, `ConversationMessage`).
- **`dispatcher/backend.go`** — delete entirely. `StreamingBackend`, `PollingBackend`, `StreamingSyncBacker` interfaces have zero production callers.
- **`dispatcher/cursor_backend.go`** — delete entirely. Stub `CursorBackend` where every method returns "not implemented."
- **`dispatcher/backend_test.go`** and **`dispatcher/cursor_backend_test.go`** — delete. Tests for dead code.

## Data Structures

`SyncResult` relocated, not deleted. Everything else is pure deletion.

## Routing

- Provider: `codex`
- Model: `gpt-5.4`
- Mechanical deletion with no judgment calls.

## Verification

### Static
- `go build ./dispatcher/...` — confirms no remaining references
- `go test ./dispatcher/...` — remaining tests still pass
- `go vet ./...` — clean

### Runtime
- `go test ./...` — full suite passes
