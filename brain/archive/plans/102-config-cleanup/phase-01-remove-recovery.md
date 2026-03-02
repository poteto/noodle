---
todo: 103
---

# Phase 1 — Remove `[recovery]` Config Section

Back to [[plans/102-config-cleanup/overview]]

## Goal

Delete the `[recovery]` config section and `RecoveryConfig` type. This field (`max_retries = 3`) is defined and validated but never read by production code. Also delete the unused internal fallback `defaultRetryMaxAttempts = 2`, remove loop auto-retry-on-dispatch-failure behavior in runtime paths, and clean up implicit retry branches from `internal/dispatch` if still unused so retries remain scheduler-owned.

## Changes

### `config/types_defaults.go`
- Delete `RecoveryConfig` struct
- Remove `Recovery RecoveryConfig` field from `Config`
- Remove recovery defaults from `DefaultConfig()`

### `config/parse.go`
- Remove `recovery.max_retries` default-setting logic
- Remove `recovery.max_retries >= 0` validation
- **Add parse warning infrastructure:** normalize invalid/removed values to defaults at parse boundary and emit structured warnings instead of returning validation errors.
- **Add removed-key detection (warning-only):** after TOML decode, inspect `metadata.Undecoded()` and match against an explicit removed-prefix list (`recovery.*` initially). Emit a clear warning ("recovery section was removed; ignoring field and using defaults"), then continue parsing. Subsequent phases should add their removed prefixes to this shared matcher.

### `config/config_test.go`
- Remove tests asserting recovery defaults and parsing
- Remove recovery validation tests

### `docs/reference/configuration.md`
- Remove `[recovery]` section documentation

### `generate/skill_noodle.go`
- Remove `recovery.max_retries` field description

### `internal/dispatch/dispatch.go`
- Delete `defaultRetryMaxAttempts`
- Delete `defaultRetryPolicy()`
- In `RouteCompletion()`, remove the failed-attempt retry branch; failed/cancelled attempts should route directly to failure events (no implicit stage reset to pending).
- In `RouteFailure()`, remove retry-policy status branching; failed stages should mark the order failed unless explicitly reactivated via scheduler control commands.
- Delete `RetryCandidate` and retry-reason constants if no call sites remain.

### `internal/dispatch/types.go`
- Delete `RetryPolicy` type if no call sites remain after dispatch retry-branch removal.

### `internal/dispatch/dispatch_test.go`
- Update tests to pass explicit retry policy inputs where needed
- Remove assertions that depend on implicit default retry policy behavior

### `loop/cook_spawn.go`
- Remove the retryable-path special case in `handleCookDispatchFailure` that resets stage status to pending
- On dispatch failure, always mark stage/order failed, emit failure event context, and forward to scheduler

### `loop/schedule.go`
- Remove the retryable-path special case in `spawnSchedule` that silently resets schedule stage to pending
- Keep failure handling explicit (scheduler sees failure and decides what to do)

### Runtime precedence note
- Treat `loop/*` retry-removal work as the runtime-authoritative behavior change.
- `internal/dispatch/*` changes are cleanup-only unless a concrete runtime caller exists.

### `loop/dispatch_failure_envelope.go`
- Remove `AgentStartFailureClassRetryable` and retryability-driven branching used only for loop auto-retry decisions.
- Preserve runtime fallback and unrecoverable classification paths (`AgentStartFailureClassFallback`, `AgentStartFailureClassUnrecoverable`).
- In `classifyAgentStartFailure()`, make the default class explicit: `unrecoverable` unless a known fallback flow is being emitted.

### `internal/mode/gate.go` + `internal/mode/mode_test.go`
- Remove `ActionRetry`, `CanRetry`, and retry-specific blocked reason text (no runtime callers should remain)
- Preserve schedule/dispatch/merge gating behavior

### `docs/concepts/modes.md`
- Remove language implying automatic loop retries; document that retries are scheduler-directed

### Bootstrap retry scope
- `spawnBootstrapIfNeeded()`'s `bootstrapAttempts` retry loop is **out of scope** for Plan 102.
- Rationale: bootstrap retries are onboarding resilience for schedule-skill creation, not workload-stage retry policy.

### `generate/skill_noodle.go`
- Update mode contract table and prose to remove "Auto-retry" claims and retry-gate wording tied to implicit loop retries.
- Update dispatch section text if it still claims `RouteCompletion` performs automatic retry routing.

### `.agents/skills/noodle/SKILL.md` + `.agents/skills/noodle/references/config-schema.md`
- Remove `recovery.max_retries` config docs.
- Update mode tables/text to remove "Auto-retry" claims and keep scheduler-directed retry wording.

### Test fixtures
- Remove `recovery` fields from any `.noodle.toml` fixtures (check `loop/testdata/`)
- Remove `Recovery` references in `loop/fixture_test.go`, `loop/log_test.go`, `loop/loop_event_integration_test.go`
- Update or delete retry-specific loop/dispatch tests (`loop/loop_test.go`, `loop/cook_spawn_test.go`, `internal/dispatch/dispatch_test.go`) that assert implicit auto-retry behavior.

## Data Structures

Remove `RecoveryConfig` entirely — no replacement type needed.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Includes dispatch state-machine behavior changes and fallback classification boundaries |

## Verification

### Static
- `pnpm check` passes (full suite)
- No references to `RecoveryConfig` remain
- No references to `defaultRetryMaxAttempts` or `defaultRetryPolicy()` remain
- No implicit retry branches remain in `RouteCompletion()` / `RouteFailure()`
- No loop/runtime branches that auto-reset failed dispatches to pending based on retryable classification
- No references to `ActionRetry` / `CanRetry` remain
- `classifyAgentStartFailure()` has an explicit non-retryable default and preserved fallback path behavior

### Runtime
- Parse a `.noodle.toml` without `[recovery]` — should work (already the common case)
- Parse a `.noodle.toml` WITH `[recovery]` — should still parse, with a clear warning and defaults applied
- **Test the removed-key check:** assert that a `.noodle.toml` containing `[recovery]` or `recovery.max_retries` returns warning(s), not an error
- Dispatch failure path: stage transitions to failed and scheduler receives failure context; no silent retry/reset-to-pending path in loop or dispatch state machine.
- Runtime fallback path still works for supported fallback flows (for example, non-process runtime dispatch failure that falls back to `process`).
