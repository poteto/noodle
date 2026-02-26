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

**`loop/defaults.go`** — Build the runtime map from config. Call `Start()` on each runtime during loop init. Call `Close()` during shutdown. Preserve fallback: if a non-tmux dispatch fails, retry via tmux runtime (existing `factory.go` behavior).

## Data structures

- `Runtime` interface — `Start(ctx)`, `Dispatch(ctx, req) → (SessionHandle, error)`, `Kill(SessionHandle) error`, `Recover(ctx) → ([]RecoveredSession, error)`, `Close() error`
- `SessionHandle` — `ID() string`, `Done() <-chan struct{}`, `Status() string`, `TotalCost() float64`
- `RecoveredSession` — `OrderID string`, `SessionHandle SessionHandle`
- Runtime map: `map[string]Runtime` keyed by runtime name ("tmux", "sprites", "cursor")

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — interface design with lifecycle, adoption migration, and fallback semantics requires judgment

## Verification

### Static
- `go test ./...` — all tests pass
- `dispatcher.Dispatcher` interface no longer referenced from loop package
- `adoptedTargets`/`adoptedSessions` removed from Loop struct
- Loop imports `runtime` package, not `dispatcher` directly
- Every Runtime implementation's `Close()` stops all background goroutines

### Runtime
- Integration test: dispatch via TmuxRuntime, verify session starts and Done() fires
- Test: dispatch via mock runtime, verify loop processes completion correctly
- Test: stage with `runtime: "sprites"` routes to SpritesRuntime
- Test: kill loop, restart, verify `Recover()` finds orphaned sessions and maps them to active orders
- Test: context cancellation causes `Close()` to stop all goroutines within 5s
