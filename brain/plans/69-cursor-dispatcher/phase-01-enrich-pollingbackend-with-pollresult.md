Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 1: Enrich PollingBackend with PollResult

**Routing:** `codex` / `gpt-5.3-codex` — mechanical interface migration, clear spec

## Goal

Change `PollingBackend.PollStatus` from returning `(RemoteStatus, error)` to `(PollResult, error)`. The `PollResult` struct carries status plus optional branch and summary fields that Cursor (and future polling backends) need to communicate completion metadata.

## Data structures

- `PollResult` struct — `Status RemoteStatus`, `Branch string`, `Summary string`

## Changes

**`dispatcher/backend_types.go`**
- Add `PollResult` struct after `RemoteStatus` constants

**`dispatcher/backend.go`**
- Change `PollStatus` signature: `PollStatus(ctx context.Context, remoteID string) (PollResult, error)`

**`dispatcher/cursor_backend.go`**
- Update stub to return `PollResult{Status: RemoteStatusUnknown}, err`

**`dispatcher/backend_test.go`**
- Update `pollingBackendStub.PollStatus` to return `PollResult{Status: RemoteStatusRunning}, nil`

## Verification

### Static
- `go vet ./dispatcher/...`
- All existing tests pass
- `CursorBackend` still satisfies `PollingBackend` (compile-time check via `var _ PollingBackend = (*CursorBackend)(nil)`)

### Runtime
- `go test ./dispatcher/... -race`
- Existing `cursor_backend_test.go` passes with updated return type
