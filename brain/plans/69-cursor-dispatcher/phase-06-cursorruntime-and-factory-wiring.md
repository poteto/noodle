Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 6: CursorRuntime and Factory Wiring

**Routing:** `codex` / `gpt-5.3-codex` — mechanical wiring following existing patterns

## Goal

Wire the Cursor dispatcher into Noodle's runtime system. Add `CursorRuntime` with polling-based recovery (re-attach to live Cursor agents after restart), update `AvailableRuntimes()` to include cursor when configured, wire into `defaultDependencies()`, and validate cursor config at the boundary.

## Data structures

- `CursorRuntime` struct — extends `DispatcherRuntime` with `Recover()` implementation

## Changes

**`runtime/cursor.go` (new)**
- `NewCursorRuntime(d *dispatcher.PollingDispatcher, backend dispatcher.PollingBackend, runtimeDir string, maxConcurrent int) *CursorRuntime`
- `Recover(ctx) ([]RecoveredSession, error)` — scans session directories for `spawn.json` files with `runtime: "cursor"` and a `remote_id`. For each: calls `backend.PollStatus(remoteID)` to check if the agent is still alive. If RUNNING/CREATING: creates an adopted `pollingSession` that resumes polling. If FINISHED: writes SyncResult and returns as completed. If ERROR/EXPIRED/not-found: returns as failed. Registers recovered sessions in the polling registry.
- Wraps `PollingDispatcher` via `DispatcherRuntime` for dispatch, adds recovery on top.

**`config/config.go`**
- `AvailableRuntimes()` — add cursor when `cursorDefined` and `APIKey()` is non-empty and `Repository` is non-empty (both required for functional cursor runtime)
- Add `WebhookSecretEnv string` to `CursorConfig` (follows env-key pattern like `APIKeyEnv`)
- Add `WebhookSecret()` accessor method (reads env var, defaults to `CURSOR_WEBHOOK_SECRET`)
- Add config validation in `Validate()`: warn if cursor is defined but repository is empty

**`loop/defaults.go`**
- In `defaultDependencies()`, add cursor runtime block (after sprites block):
  - Guard: `runtimeEnabled(cfg.AvailableRuntimes(), "cursor")`
  - Create: `NewCursorBackend(apiKey, baseURL, repository)` → `NewPollingDispatcher(cfg)` → `NewCursorRuntime(dispatcher, backend, runtimeDir, maxConcurrent)`
  - Store dispatcher reference for webhook notifier wiring (expose `SessionNotifier` via Dependencies or server options)
  - Register: `runtimes["cursor"] = cursorRuntime`
- Wire `SessionNotifier` into server options so webhook endpoint can reach live sessions

**`config/config_test.go`**
- Test: `AvailableRuntimes()` includes "cursor" when cursor section defined, API key set, and repository non-empty
- Test: `AvailableRuntimes()` excludes "cursor" when API key is missing
- Test: `AvailableRuntimes()` excludes "cursor" when repository is empty
- Test: `WebhookSecret()` reads from configured env var

**`runtime/cursor_test.go` (new)**
- Test: Recover finds session with remote_id, agent still RUNNING → adopted session polling
- Test: Recover finds session with remote_id, agent FINISHED → completed with SyncResult
- Test: Recover finds session with remote_id, agent ERROR → failed
- Test: Recover finds session with remote_id, agent not found (404) → failed
- Test: Recover with no cursor sessions → empty result

## Verification

### Static
- `go vet ./...`
- `go build ./...`

### Runtime
- `go test ./config/... -run TestAvailableRuntimes -race`
- `go test ./runtime/... -run TestCursorRuntime -race`
- `go test ./loop/... -race`
- Full: `go test ./... && sh scripts/lint-arch.sh`
