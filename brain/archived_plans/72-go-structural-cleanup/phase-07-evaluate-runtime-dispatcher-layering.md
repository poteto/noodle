Back to [[plans/72-go-structural-cleanup/overview]]

# Phase 7: Evaluate runtime/dispatcher layering

## Goal

Investigate whether the `runtime` package earns its existence as a separate layer over `dispatcher`, and document the decision. This phase is an investigation — it may or may not result in code changes.

## Context

Currently:
- `dispatcher/` defines `Dispatcher` interface (just `Dispatch`) and `Session` interface
- `runtime/` defines `Runtime` interface (Start, Dispatch, Kill, Recover, TerminalHealth, InfoHealth, Close) and `SessionHandle` interface
- `runtime/runtime.go` type-aliases `DispatchRequest` and `SyncResult` from dispatcher so that loop doesn't need to import dispatcher
- `runtime/dispatcher_runtime.go` wraps `dispatcher.Dispatcher` into a `Runtime`, adding per-runtime concurrency limits and health channels
- Both sprites and tmux runtimes use `DispatcherRuntime`. Base `Start` and `Close` are no-ops. `Kill` delegates to `handle.Kill()` (not a no-op). `tmuxRuntime` overrides `Recover` with ~60 lines of real recovery logic (scanning live tmux sessions, reconciling with session dirs, killing orphans)
- Health channels are created with buffer sizes (64, 256) but nothing ever writes to them. `drainRuntimeHealth()` in `loop/health.go` runs every cycle and always hits the `default` case
- `WrapDispatcherSession` is exported but never called outside the package — dead code
- Loop imports `runtime` (aliased as `loopruntime`), never imports `dispatcher` directly. No other package imports `runtime`
- Cursor exists only as a dispatcher-level backend stub (`dispatcher/cursor_backend.go`) — no runtime-layer code

The type aliases create an indirection: `loopruntime.DispatchRequest` is secretly `dispatcher.DispatchRequest`. This works but adds cognitive overhead.

## Questions to answer

1. **Does the Runtime interface pull its weight?** `Start` and `Close` are no-ops for both runtimes. But tmux `Recover` is real (~60 lines of session reconciliation), and `Kill` delegates to actual session termination. Do these two justify the interface, or should they live elsewhere?

2. **Could concurrency limiting live on Dispatcher instead?** The main value `DispatcherRuntime` adds over `Dispatcher` is the `maxConcurrent` cap. If `Dispatcher` grew a concurrency parameter, would the runtime wrapper be unnecessary?

3. **Would collapsing the layers simplify the codebase?** If loop imported dispatcher directly (through an interface defined in loop — per Go idiom), would we lose anything? The type aliases would disappear. The health channels should be deleted regardless (dead infrastructure).

4. **What about the dead code?** `WrapDispatcherSession` is never called externally. Health channels are created but never written to. These should be removed regardless of the layering decision.

## Changes (unconditional)

Regardless of the layering decision:
- Delete `WrapDispatcherSession` — dead code
- Delete health channel infrastructure: `TerminalHealth`/`InfoHealth` channels, `HealthEvent` type, `HealthEventType` enum and its four constants (`HealthHealthy`, `HealthIdle`, `HealthStuck`, `HealthDead`), `drainRuntimeHealth()` in `loop/health.go`, `sessionHealth` map in Loop — none of this is ever written to
- Update `loop/mock_runtime_test.go` to remove `TerminalHealth()`/`InfoHealth()` implementations

## Changes (conditional)

If the evaluation concludes the layers should merge:
- Move the `Runtime` interface to `loop/types.go` (consumer-defined)
- Move `DispatcherRuntime` + concurrency limiting into `dispatcher/`
- Move tmux recovery logic to `dispatcher/`
- Delete `runtime/` package
- Update loop imports from `loopruntime` to `dispatcher`

If the evaluation concludes to keep the layers:
- Document why in a brain note
- Remove `Start` and `Close` from the interface (no-ops for all runtimes — YAGNI), delete `startRuntimes()` in `loop/loop.go`, and remove the two `runtime.Close()` calls from the shutdown paths

## Verification

- If changes made: `go test ./... && go vet ./...`
- If no changes: a brain note at `brain/decisions/runtime-dispatcher-layering.md` documenting the rationale
