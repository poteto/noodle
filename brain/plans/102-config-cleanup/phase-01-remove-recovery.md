---
todo: 103
---

# Phase 1 — Remove `[recovery]` Config Section

Back to [[plans/102-config-cleanup/overview]]

## Goal

Delete the `[recovery]` config section and `RecoveryConfig` type. This field (`max_retries = 3`) is defined and validated but never read by production code. Also delete the unused internal fallback `defaultRetryMaxAttempts = 2` and implicit default retry policy wiring in `internal/dispatch` so retries remain scheduler-owned.

## Changes

### `config/types_defaults.go`
- Delete `RecoveryConfig` struct
- Remove `Recovery RecoveryConfig` field from `Config`
- Remove recovery defaults from `DefaultConfig()`

### `config/parse.go`
- Remove `recovery.max_retries` default-setting logic
- Remove `recovery.max_retries >= 0` validation
- **Add removed-key detection (warning-only):** after TOML decode, check undecoded keys that match removed config paths (`recovery.*`). Emit a clear warning ("recovery section was removed; ignoring field and using defaults"), then continue parsing. Subsequent phases should add their removed keys to this same warning path.

### `config/config_test.go`
- Remove tests asserting recovery defaults and parsing
- Remove recovery validation tests

### `docs/reference/configuration.md`
- Remove `[recovery]` section documentation

### `generate/skill_noodle.go`
- Remove `recovery.max_retries` field description

### `internal/dispatch/dispatch.go`
- Delete `defaultRetryMaxAttempts`
- Delete `defaultRetryPolicy()` if it is only used as an implicit fallback
- Remove implicit retry-policy fallback wiring and require explicit policy at call sites if this package remains in use

### `internal/dispatch/dispatch_test.go`
- Update tests to pass explicit retry policy inputs where needed
- Remove assertions that depend on implicit default retry policy behavior

### Test fixtures
- Remove `recovery` fields from any `.noodle.toml` fixtures (check `loop/testdata/`)
- Remove `Recovery` references in `loop/fixture_test.go`, `loop/log_test.go`, `loop/loop_event_integration_test.go`

## Data Structures

Remove `RecoveryConfig` entirely — no replacement type needed.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical deletion against a clear spec |

## Verification

### Static
- `pnpm check` passes (full suite)
- No references to `RecoveryConfig` remain
- No references to `defaultRetryMaxAttempts` or `defaultRetryPolicy()` remain

### Runtime
- Parse a `.noodle.toml` without `[recovery]` — should work (already the common case)
- Parse a `.noodle.toml` WITH `[recovery]` — should still parse, with a clear warning and defaults applied
- **Test the removed-key check:** assert that a `.noodle.toml` containing `[recovery]` or `recovery.max_retries` returns warning(s), not an error
