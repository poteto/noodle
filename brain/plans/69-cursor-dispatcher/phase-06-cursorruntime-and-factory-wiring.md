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
- `Recover(ctx) ([]RecoveredSession, error)` — scans session directories for `spawn.json` files with `runtime: "cursor"` and a `remote_id`. For each: calls `backend.PollStatus(remoteID)` to check if the agent is still alive. If RUNNING/CREATING: creates an adopted `pollingSession` that resumes polling. If FINISHED: writes SyncResult and returns as completed. If ERROR/EXPIRED/not-found: returns as failed. Registers recovered sessions in the polling registry. **Must populate `RecoveredSession.OrderID`** from `spawn.json` metadata (order identity is persisted at dispatch time alongside `remote_id`). Without this, reconcile cannot map recovered sessions to orders and stages remain stuck.
  - Transient errors during recovery polls (429/5xx) — treat as "still alive" and adopt for continued polling, rather than failing the session based on a temporary API error.
- Wraps `PollingDispatcher` via `DispatcherRuntime` for dispatch, adds recovery on top.

**`config/config.go`**
- `AvailableRuntimes()` — add cursor when `cursorDefined` and `APIKey()` is non-empty and `Repository` is non-empty (both required for functional cursor runtime)
- Add `WebhookSecretEnv string` to `CursorConfig` (follows env-key pattern like `APIKeyEnv`)
- Add `WebhookSecret()` accessor method (reads env var, defaults to `CURSOR_WEBHOOK_SECRET`)
- Add config validation in `Validate()`: warn if cursor is defined but repository is empty

**`loop/reconcile.go`**
- `refreshAdoptedTargets` currently checks `SessionPIDAlive` to determine if adopted sessions are still live. This is PID-specific — remote polling sessions have no local PID. Add runtime-aware liveness: if `spawn.json` has `runtime: "cursor"` (or any non-process runtime), check heartbeat recency instead of PID liveness. This prevents recovered cursor sessions from being pruned and re-dispatched while the remote agent is still running.

**`dispatcher/dispatch_metadata.go`**
- Persist `order_id` in `spawn.json` at dispatch time (already has `session_id`, `runtime`, `remote_id`). This enables `CursorRuntime.Recover` to populate `RecoveredSession.OrderID`.

**`loop/defaults.go`**
- In `defaultDependencies()`, add cursor runtime block (after sprites block):
  - Guard: `runtimeEnabled(cfg.AvailableRuntimes(), "cursor")`
  - Create: `NewCursorBackend(apiKey, baseURL, repository)` → `NewPollingDispatcher(cfg)` → `NewCursorRuntime(dispatcher, backend, runtimeDir, maxConcurrent)`
  - Store dispatcher reference for webhook notifier wiring
  - Register: `runtimes["cursor"] = cursorRuntime`
- Expose `SessionNotifier` from `defaultDependencies` return (add to `Dependencies` struct or return alongside)

**`cmd_start.go`**
- Thread `SessionNotifier` from dependencies into `server.Options`. Concrete wiring path: `defaultDependencies()` returns notifier → `runWebServer()` receives it as parameter → passes to `server.Options{SessionNotifier: notifier}`. This is the missing boundary contract that reviewers flagged.

**`server/server.go`**
- Add `SessionNotifier` field to `server.Options` struct (interface from phase 4: `Nudge(remoteID string)`). Used by webhook handler in phase 7.

**`config/config_test.go`**
- Test: `AvailableRuntimes()` includes "cursor" when cursor section defined, API key set, and repository non-empty
- Test: `AvailableRuntimes()` excludes "cursor" when API key is missing
- Test: `AvailableRuntimes()` excludes "cursor" when repository is empty
- Test: `WebhookSecret()` reads from configured env var

**`runtime/cursor_test.go` (new)**
- Test: Recover finds session with remote_id, agent still RUNNING → adopted session polling, OrderID populated from spawn.json
- Test: Recover finds session with remote_id, agent FINISHED → completed with SyncResult
- Test: Recover finds session with remote_id, agent ERROR → failed
- Test: Recover finds session with remote_id, agent not found (404) → failed
- Test: Recover with no cursor sessions → empty result
- Test: Recover with transient 429 error during poll → session adopted (not failed)
- Test: Recover populates OrderID from spawn.json metadata

**`loop/reconcile_test.go`**
- Test: `refreshAdoptedTargets` keeps cursor sessions alive when heartbeat is recent (no PID check)
- Test: `refreshAdoptedTargets` prunes cursor sessions when heartbeat is stale

## Verification

### Static
- `go vet ./...`
- `go build ./...`

### Runtime
- `go test ./config/... -run TestAvailableRuntimes -race`
- `go test ./runtime/... -run TestCursorRuntime -race`
- `go test ./loop/... -race`
- Full: `go test ./... && sh scripts/lint-arch.sh`
