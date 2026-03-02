---
todo: "107 (part 1)"
---

# Phase 7 — Hardcode `max_completion_overflow` and `merge_backpressure_threshold`

Back to [[plans/102-config-cleanup/overview]]

## Goal

Remove `max_completion_overflow` and `merge_backpressure_threshold` from user-facing config. These are internal plumbing (channel buffer size, merge queue backpressure). Hardcode the current defaults (1024, 128) as constants where they're consumed.

## Changes

### `config/types_defaults.go`
- Remove `MaxCompletionOverflow int` from `ConcurrencyConfig`
- Remove `MergeBackpressureThreshold int` from `ConcurrencyConfig`
- Remove from `DefaultConfig()`

### `config/parse.go`
- Remove default-setting logic for both fields
- Remove validation for both fields (`> 0` checks)

### `config/config_test.go`
- Remove tests asserting these defaults and validation

### `loop/loop.go`
- Replace `maxCompletionOverflow()` config read with hardcoded constant: `const completionBufferSize = 1024`

### `loop/loop_cycle_pipeline.go`
- Replace `mergeBackpressureThreshold` config read with hardcoded constant: `const mergeBackpressureLimit = 128`

### `docs/reference/configuration.md`
- Remove both fields from concurrency section docs

### `generate/skill_noodle.go`
- Remove field descriptions for both

## Data Structures

`ConcurrencyConfig` shrinks to `MaxConcurrency` + `ShutdownTimeout` (shutdown removed in phase 8).

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical deletion + constant extraction |

## Verification

### Static
- `pnpm check` passes (full suite)
- No references to `MaxCompletionOverflow`, `MergeBackpressureThreshold`, `max_completion_overflow`, or `merge_backpressure_threshold` remain

### Runtime
- `go test ./loop/...` — loop tests pass with hardcoded values
- `go test ./config/...` — config tests pass
