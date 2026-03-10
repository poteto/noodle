---
todo: "104 (part 1)"
---

# Phase 3 — Remove Stuck Detection + `stuck_threshold`

Back to [[plans/102-config-cleanup/overview]]

## Goal

Remove the "stuck session" concept entirely. Long-running tasks aren't stuck — they're working. The config field `monitor.stuck_threshold` is defined but never wired (monitor uses its own hardcoded constant). Remove both the config field and the stuck detection logic.

## Changes

### `config/types_defaults.go`
- Remove `StuckThreshold string` from `MonitorConfig`

### `config/parse.go`
- Remove `monitor.stuck_threshold` default-setting logic
- Remove `stuck_threshold` duration validation

### `config/config_test.go`
- Remove tests asserting stuck_threshold defaults and parsing

### `monitor/derive.go`
- Remove the stuck detection check (`stuck := observation.Alive && stuckThreshold > 0 && ...`)
- Remove `SessionStatusStuck` assignment
- Remove health degradation for stuck sessions (Yellow at half-threshold, Red at threshold)

### `monitor/types.go`
- Remove `defaultStuckThreshold` constant
- Remove `SessionStatusStuck` from status enum if defined there

### `monitor/monitor.go`
- Remove `stuckThreshold` parameter/field if passed through

### `docs/reference/configuration.md`
- Remove `stuck_threshold` from monitor section docs

### `generate/skill_noodle.go`
- Remove `monitor.stuck_threshold` field description

## Data Structures

Remove `SessionStatusStuck` from session status enum. Sessions are either alive, completed, or failed — not "stuck."

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Deletion with clear spec, touches monitor internals |

## Verification

### Static
- `pnpm check` passes (full suite)
- No references to `StuckThreshold`, `stuck_threshold`, `SessionStatusStuck`, or `defaultStuckThreshold` remain

### Runtime
- `go test ./monitor/...` — monitor tests pass without stuck assertions
- `go test ./config/...` — config tests pass
