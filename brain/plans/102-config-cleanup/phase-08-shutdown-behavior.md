---
todo: "107 (part 2)"
---

# Phase 8 — Replace `shutdown_timeout` With SIGTERM + 2s Deadline

Back to [[plans/102-config-cleanup/overview]]

## Goal

Remove `shutdown_timeout` from config. Users expect `Ctrl-C` to stop Noodle immediately, not wait 30 seconds. New behavior: send SIGTERM to all agents, wait 2 seconds, then SIGKILL any survivors. No configurable timeout.

## Changes

### `config/types_defaults.go`
- Remove `ShutdownTimeout string` from `ConcurrencyConfig`
- `ConcurrencyConfig` now has only `MaxConcurrency` — evaluate whether the `[concurrency]` section name still makes sense (keep it; one field is fine)
- Remove from `DefaultConfig()`

### `config/parse.go`
- Remove `shutdown_timeout` default-setting logic
- Remove `shutdown_timeout` duration validation

### `config/config_test.go`
- Remove shutdown_timeout tests

### Session interface — `dispatcher/types.go`
- Split `Kill()` into two explicit primitives:
  - `Terminate()` — send SIGTERM only, non-blocking, returns immediately
  - `ForceKill()` — send SIGKILL only, non-blocking, returns immediately
- The loop orchestrates the global deadline over these primitives; sessions no longer own the TERM→wait→KILL sequence internally

### `loop/loop.go`
- Guard `Shutdown()` with `sync.Once` — currently invoked twice on Ctrl-C (once in `shutdownAndDrain()`, once by deferred `runtimeLoop.Shutdown()` in `cmd_start.go`). Without `sync.Once`, the second call re-runs the kill sequence.
- Replace the shutdown grace period logic with global two-phase shutdown:
  - New: hardcode `const shutdownDeadline = 2 * time.Second`
  - **Step A:** Call `Terminate()` on all sessions in parallel (broadcast SIGTERM)
  - **Step B:** Call `Terminate()` on all adopted PIDs in parallel (via pid_observer)
  - **Step C:** Single global 2s wait via `time.After(shutdownDeadline)`
  - **Step D:** Call `ForceKill()` on all surviving sessions + adopted PIDs in parallel
  - Total wall-clock: always ~2s regardless of session count

### `dispatcher/process.go`
- Implement `Terminate()` (SIGTERM to process group, return immediately)
- Implement `ForceKill()` (SIGKILL to process group, return immediately)
- Remove the old `Kill()` method's internal 5s wait+KILL sequence

### `dispatcher/sprites_session.go`
- Implement `Terminate()` and `ForceKill()` matching the new interface

### `monitor/pid_observer.go`
- Keep `KillSessionByPID` non-blocking and SIGTERM-only — this is used by the monitor repair path during normal operation, not just shutdown. Do NOT add SIGKILL or waits here.
- SIGKILL for adopted PIDs during shutdown is orchestrated by the loop (Step D above), not by the monitor

### `docs/reference/configuration.md`
- Remove `shutdown_timeout` from concurrency section docs
- Add a note about shutdown behavior if appropriate

### `generate/skill_noodle.go`
- Remove `concurrency.shutdown_timeout` field description

### `.agents/skills/noodle/SKILL.md`
- Remove `concurrency.shutdown_timeout` entry from the config field table

## Data Structures

`ConcurrencyConfig` loses `ShutdownTimeout`. Shutdown behavior becomes a hardcoded constant, not configurable.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Behavior change in shutdown path, needs judgment about edge cases |

## Verification

### Static
- `pnpm check` passes (full suite)
- No references to `ShutdownTimeout`, `shutdown_timeout`, or old `Kill()` method remain

### Runtime
- `go test ./loop/...` — shutdown tests pass with 2s deadline
- `go test ./dispatcher/...` — process dispatcher tests pass with new `Terminate()`/`ForceKill()` interface
- **Idempotency test:** assert `Shutdown()` called twice executes the kill sequence only once (`sync.Once`)
- **Wall-clock bound test:** assert shutdown completes within 3s (2s deadline + 1s margin) regardless of session count
- Manual test: run `noodle start`, spawn a cook, hit Ctrl-C. Noodle should exit within ~2s. Agent processes should not linger.
- **Multi-session test:** spawn 3+ concurrent cooks, hit Ctrl-C. Total shutdown time should still be ~2s (not N*2s). Verify no orphaned agent processes via `ps`.
- **E2e smoke test:** `noodle start` → enqueue an order → cook runs to completion → `Ctrl-C` exits within ~2s. Full config-parse → loop → cook → shutdown path with the reduced config surface.
