Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 1: Enrich PollingBackend with PollResult

**Routing:** `codex` / `gpt-5.4` — mechanical interface migration, clear spec

## Goal

Change `PollingBackend.PollStatus` from returning `(RemoteStatus, error)` to `(PollResult, error)`. The `PollResult` struct carries status plus optional branch and summary fields that Cursor (and future polling backends) need to communicate completion metadata.

## Data structures

- `PollResult` struct — `Status RemoteStatus`, `Branch string`, `Summary string`
- `LaunchResult` struct — `RemoteID string`, `TargetBranch string`
- `APIError` struct — `StatusCode int`, `Message string`, `Retryable bool`, `RetryAfter time.Duration`

## Changes

**`dispatcher/backend_types.go`**
- Add `PollResult` struct after `RemoteStatus` constants
- Add `LaunchResult` struct — immutable metadata returned from Launch
- Add `APIError` struct with `Retryable` classification and `RetryAfter time.Duration` — used by HTTP client, consumed by pollingSession to distinguish terminal vs retryable errors and honor server backoff hints

**`dispatcher/backend.go`**
- Change `Launch` signature: `Launch(ctx context.Context, config PollLaunchConfig) (LaunchResult, error)`
- Change `PollStatus` signature: `PollStatus(ctx context.Context, remoteID string) (PollResult, error)`

**`dispatcher/cursor_backend.go`**
- Update stub to return `LaunchResult{}, err` and `PollResult{Status: RemoteStatusUnknown}, err`

**`dispatcher/backend_test.go`**
- Update `pollingBackendStub.Launch` to return `LaunchResult{RemoteID: "test-id"}, nil`
- Update `pollingBackendStub.PollStatus` to return `PollResult{Status: RemoteStatusRunning}, nil`

## Verification

### Static
- `go vet ./dispatcher/...`
- All existing tests pass
- `CursorBackend` still satisfies `PollingBackend` (compile-time check via `var _ PollingBackend = (*CursorBackend)(nil)`)

### Runtime
- `go test ./dispatcher/... -race`
- Existing `cursor_backend_test.go` passes with updated return types
- Test: `APIError.Retryable` classification for 429, 5xx (retryable) vs 401, 403, 404, 410 (terminal)
- Test: `APIError.RetryAfter` populated from 429 response, zero for other errors
