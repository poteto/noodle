---
todo: "104 (part 2)"
---

# Phase 5 — Hardcode `poll_interval` to 1s, Remove `[monitor]` Section

Back to [[plans/102-config-cleanup/overview]]

## Goal

After phases 2-3 removed `ticket_stale` and `stuck_threshold`, `poll_interval` is the last field in `MonitorConfig`. It's an implementation detail (lightweight local I/O check), not a user knob. Hardcode it to 1s and delete the entire `[monitor]` config section.

## Changes

### `config/types_defaults.go`
- Delete `MonitorConfig` struct entirely
- Remove `Monitor MonitorConfig` field from `Config`
- Remove monitor defaults from `DefaultConfig()`

### `config/parse.go`
- Remove `monitor.poll_interval` default-setting logic
- Remove `poll_interval` duration validation
- Remove any remaining monitor-related parsing

### `config/config_test.go`
- Remove monitor-related tests (defaults, parsing, validation)

### `loop/util.go`
- Replace `pollInterval()` method: instead of reading from config, return hardcoded `1 * time.Second`
- Or replace with a package-level constant `const defaultPollInterval = 1 * time.Second`

### `monitor/monitor.go`
- Replace `defaultPollInterval` with `1 * time.Second` if it was `5 * time.Second`

### `docs/reference/configuration.md`
- Remove entire `[monitor]` section documentation

### `generate/skill_noodle.go`
- Remove all `monitor.*` field descriptions

### `.agents/skills/noodle/SKILL.md`
- Remove `monitor.*` entries from the config field table

## Data Structures

Delete `MonitorConfig` entirely — no replacement.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical deletion + one hardcoded constant |

## Verification

### Static
- `pnpm check` passes (full suite)
- No references to `MonitorConfig`, `monitor.poll_interval`, or `[monitor]` remain

### Runtime
- `go test ./loop/...` — loop tests pass with 1s poll interval
- `go test ./monitor/...` — monitor tests pass
- `go test ./config/...` — config tests pass without monitor section
- **Cycle cost check:** verify that a no-op cycle (no state changes) is cheap at 1s frequency. The loop should short-circuit when nothing changed — confirm that mise rebuild and external sync scripts are not invoked unconditionally every tick. If they are, add a dirty-state guard before hardcoding 1s.
