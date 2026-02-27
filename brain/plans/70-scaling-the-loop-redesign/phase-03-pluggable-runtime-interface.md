Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 3: Pluggable Runtime interface

## Goal

Extract the dispatch + observation responsibilities into a `Runtime` interface with explicit lifecycle management. `TmuxRuntime` wraps the existing tmux dispatcher + tmux observer. `SpritesRuntime` wraps the sprites dispatcher + heartbeat observer. The loop dispatches through the Runtime, which handles session lifecycle details internally.

## Changes

**`runtime/runtime.go`** (new package) — Define the `Runtime` interface with full lifecycle:
- `Start(ctx context.Context) error` — start background goroutines (observation, etc.)
- `Dispatch(ctx context.Context, req DispatchRequest) (SessionHandle, error)` — create a session
- `Kill(handle SessionHandle) error` — cancel a session
- `Recover(ctx context.Context) ([]RecoveredSession, error)` — discover pre-existing sessions from a previous loop run (replaces `adoptedTargets`/`adoptedSessions`)
- `Close() error` — stop background goroutines, clean up

The `Done()` channel on `SessionHandle` **must** close exactly once on completion, cancellation, or runtime shutdown. This is a contract requirement — leaking watcher goroutines is a correctness bug.

**`runtime/tmux.go`** — `TmuxRuntime` wrapping the existing `TmuxDispatcher`. `Recover()` scans `.noodle/sessions/` + `tmux list-sessions` to find orphaned sessions (migrating logic from `reconcile.go`). Preserves existing tmux-to-tmux fallback semantics from `dispatcher/factory.go`.

**`runtime/sprites.go`** — `SpritesRuntime` wrapping the existing `SpritesDispatcher`. `Recover()` checks sprites API for running sessions.

**`loop/types.go`** — Replace `Dispatcher` interface with a map of runtime name → `Runtime`. Remove `adoptedTargets` and `adoptedSessions` — replaced by `Runtime.Recover()`. The loop selects runtime based on the stage's `Runtime` field (falling back to config default).

**`loop/cook.go`** — `spawnCook()` looks up the runtime from the stage, dispatches through it. The goroutine watching `Done()` (from phase 2) works identically regardless of runtime.

**`loop/reconcile.go`** — Rewrite to call `Runtime.Recover()` on each registered runtime during startup. Map recovered sessions back to active orders.

**`loop/defaults.go`** — Build the runtime map from config. Lifecycle anchoring: call `Runtime.Start()` for each runtime in `Loop.Run()` before reconcile (current startup is `Run` → reconcile → cycle at `loop/loop.go:152-169`). Call `Runtime.Close()` in `Loop.Shutdown()` after killing active sessions (current shutdown at `loop/loop.go:141-149`) — runtime close must happen after session kill so `Done()` channels fire before the runtime tears down its observation goroutines. Preserve fallback: if a non-tmux dispatch fails, retry via tmux runtime (existing `factory.go` behavior).

**`dispatcher/`** — Delete the `dispatcher` package after all callers are migrated. The `factory.go` fallback logic moves into `loop/defaults.go` (runtime map construction). Per migrate-callers-then-delete, the full import inventory is:
- `loop/` — cook.go, control.go (session type refs), schedule.go, types.go (Dispatcher interface + Session interface), defaults.go (factory construction)
- `main.go:281` — `startRepairSession` direct dispatch
- `cmd_dispatch.go:10` — CLI dispatch command
- `app_helpers.go:43` — `ProviderConfigs` type from dispatcher
- `cook.go:368,393` — sync-result parsing uses dispatcher session types

Split into sub-steps: (a) extract shared types (`Session`, `DispatchRequest`, provider configs) into `runtime/` or a shared types package; (b) migrate loop callers to `Runtime` interface; (c) migrate `main.go`, `cmd_dispatch.go`, `app_helpers.go` to new types; (d) delete `dispatcher/` package. Note: `server/` has no direct dispatcher imports — skip it.

**Per-runtime concurrency**: Each `Runtime` implementation enforces its own `MaxConcurrent` limit, returning a "concurrency limit reached" error from `Dispatch()`. The loop's global `MaxCooks` (currently `config.Concurrency.MaxCooks` at `config/config.go:76`) remains as a ceiling. This requires adding per-runtime config fields (`config.Runtime.Tmux.MaxConcurrent`, `config.Runtime.Sprites.MaxConcurrent`) with defaults and validation before the runtime implementations can enforce caps. This allows e.g. 50 cloud agents + 4 tmux agents within a global cap of 54.

**Quality verdicts**: `SessionHandle` gains a `VerdictPath() string` method returning the path where the quality verdict for this session is written. For local runtimes, this is `.noodle/quality/<session-id>.json`. For cloud runtimes, the verdict is fetched on completion and written to the same local path. The loop's quality gate reads from this path uniformly.

**Recovered-session watcher registration**: When `Runtime.Recover()` returns sessions, the loop must register a watcher goroutine for each (same pattern as dispatch from phase 2). The existing `collectAdoptedCompletions()` path and `adoptedTargets`/`adoptedSessions` maps are removed — `Recover()` replaces them entirely. Unknown sessions (empty `OrderID` in `RecoveredSession`) should be killed via `Runtime.Kill()` with a log warning.

**Internal sequencing**: (a) Define `Runtime` interface + `SessionHandle` + `DispatchRequest` types; (b) add per-runtime config fields with defaults; (c) implement `TmuxRuntime` wrapping existing dispatcher; (d) rewrite `reconcile.go` to use `Runtime.Recover()` + register watcher goroutines; (e) implement `SpritesRuntime`; (f) extract shared types from `dispatcher/`; (g) migrate all `dispatcher` imports (loop, main.go, cmd_dispatch.go, app_helpers.go); (h) delete `dispatcher/` package.

## Data structures

- `Runtime` interface — `Start(ctx)`, `Dispatch(ctx, req) → (SessionHandle, error)`, `Kill(SessionHandle) error`, `Recover(ctx) → ([]RecoveredSession, error)`, `Close() error`
- `SessionHandle` — `ID() string`, `Done() <-chan struct{}`, `Status() SessionStatus` (typed constant, not bare string), `TotalCost() float64`, `VerdictPath() string`
- `RecoveredSession` — `OrderID string` (optional — may be empty for sessions that can't be mapped to an order), `SessionHandle SessionHandle`, `RuntimeName string`, `Reason string` (how the session was discovered)
- Runtime map: `map[string]Runtime` keyed by runtime name ("tmux", "sprites", "cursor")

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — interface design with lifecycle, adoption migration, and fallback semantics requires judgment

## Verification

### Static
- `go test ./...` — all tests pass
- `dispatcher.Dispatcher` interface no longer referenced from loop package
- `adoptedTargets`/`adoptedSessions` removed from Loop struct
- Loop imports `runtime` package, not `dispatcher` directly
- `dispatcher/` package deleted — no remaining imports anywhere in the codebase
- Every Runtime implementation's `Close()` stops all background goroutines

### Runtime
- Integration test: dispatch via TmuxRuntime, verify session starts and Done() fires
- Test: dispatch via mock runtime, verify loop processes completion correctly
- Test: stage with `runtime: "sprites"` routes to SpritesRuntime
- Test: kill loop, restart, verify `Recover()` finds orphaned sessions and maps them to active orders
- Test: `Recover()` returns session with empty OrderID — session is killed, not adopted
- Test: recovered sessions get watcher goroutines registered in `watcherWG`
- Test: context cancellation causes `Close()` to stop all goroutines — verify via `watcherWG.Wait()` completing within 5s
- Test: per-runtime concurrency cap — dispatch returns error when limit reached, global MaxCooks still enforced
