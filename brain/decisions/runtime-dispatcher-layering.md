# Decision: Keep runtime/ as a separate layer from dispatcher/

## Date
2026-02-26

## Context

During the Go structural cleanup (#72, phase 7), we evaluated whether `runtime/` should merge into `dispatcher/` or remain a distinct package.

## Decision

Keep the layers separate. Remove dead weight (`Start`, `Close`, health channels, `WrapDispatcherSession`).

## Rationale

The runtime layer earns its existence through three responsibilities that don't belong in `dispatcher/`:

1. **Per-runtime concurrency limiting** — `DispatcherRuntime` wraps any `dispatcher.Dispatcher` with a `maxConcurrent` cap. This is a runtime-level policy, not a backend concern. Individual dispatchers (tmux, sprites, cursor) shouldn't each implement their own concurrency control.

2. **Tmux-specific recovery** — `tmuxRuntime.Recover()` is ~60 lines of real logic: scanning live tmux sessions, reconciling with session directories, killing orphans. This is platform-specific lifecycle management, not dispatch. Putting it in `dispatcher/` would pollute a package that's otherwise backend-agnostic.

3. **SessionHandle abstraction** — Adds `VerdictPath()` (derived from runtime directory + session ID) on top of `dispatcher.Session`. This join of runtime context + session identity doesn't belong in the dispatcher layer.

## What was removed

- `Start()` / `Close()` — no-ops for all runtimes (YAGNI)
- `TerminalHealth()` / `InfoHealth()` — channels created but never written to
- `HealthEvent` / `HealthEventType` — dead types
- `WrapDispatcherSession` — never called externally
- `drainRuntimeHealth()` — always hit the `default` case
- `sessionHealth` map — populated by dead drain, never read
- `startRuntimes()` — called `Start()` which was always a no-op

## Resulting interface

```go
type Runtime interface {
    Dispatch(ctx context.Context, req DispatchRequest) (SessionHandle, error)
    Kill(handle SessionHandle) error
    Recover(ctx context.Context) ([]RecoveredSession, error)
}
```
