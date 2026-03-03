---
id: 68
created: 2026-02-27
status: ready
---

# Unified Involvement Levels

## Context

Noodle currently has two independent dials for human oversight:

- **`autonomy`** (auto | approve) — controls merge gating only
- **`schedule.run`** (after-each | after-n | manual) — intended to control scheduling frequency but is **vestigial** (not consumed by any loop code)

Neither field controls dispatch. The user has no way to say "I want to drive everything myself" or "run fully autonomous" with a single setting. The two fields create a confusing 2×3 matrix where most combinations are meaningless.

## Scope

**In scope:**
- New `mode` field (auto | supervised | manual) replacing both `autonomy` and `schedule.run`
- Merge gating derived from mode (auto = per-skill, supervised/manual = always approve)
- Dispatch gating: manual mode suppresses auto-dispatch and retries
- Schedule gating: manual mode suppresses auto-schedule injection
- `dispatch` control command for manual-mode users to trigger cook spawning
- `mode` control command replaces `autonomy` control command
- Snapshot/SSE/status expose `mode` instead of `autonomy`
- Web UI displays and controls mode
- Schedule skill receives mode context via mise
- Noodle skill docs updated

**Out of scope:**
- Per-skill mode overrides (mode is global only)
- New UI pages or layouts — just update existing components
- Changes to `permissions.merge` per-skill — continues working as fine-grained override under auto mode
- Backward compatibility — old `autonomy` and `schedule.run` fields are deleted, not migrated

## Behavior Matrix

| Behavior | `auto` | `supervised` | `manual` |
|----------|--------|-------------|----------|
| Auto-schedule injection | yes | yes | no |
| Auto-dispatch (tick spawns cooks) | yes | yes | no |
| Auto-retry on failure | yes | yes | no |
| Merge gating | per-skill `permissions.merge` | always approve | always approve |
| Quality verdict checked | yes | no (parked before verdict) | no (parked before verdict) |

## Constraints

- **No backward compatibility**: `autonomy` and `schedule.run` are deleted. Users update `.noodle.toml` to use `mode`. No migration, no deprecation warnings, no dual paths.
- **`permissions.merge` unchanged**: Per-skill merge permission continues to work. In `auto` mode, skills with `permissions.merge: false` skip merge entirely. In `supervised`/`manual`, everything parks for review regardless.
- **Dispatch gating includes retries**: Manual mode suppresses `planCycleSpawns()` AND retry paths (`processPendingRetries`, `retryCook`).
- **Old config fields silently ignored**: TOML parser drops unrecognized keys. Users with old `autonomy`/`[schedule]` in `.noodle.toml` won't get errors — the fields are just ignored and `mode` defaults to `auto`. This is acceptable; no migration shim needed.

## Alternatives Considered

1. **Keep two separate fields, wire up `schedule.run`**: Rejected — confusing matrix. Users think in involvement levels.
2. **Four modes (auto/supervised/manual/off)**: Rejected — "off" is just not running noodle.
3. **Mode as an object with per-behavior overrides**: Rejected — over-engineering. Per-skill `permissions.merge` covers fine-grained needs.

## All Consumers (must be updated atomically in phase 1)

| File | Usage |
|------|-------|
| `config/types_defaults.go` | Constants, field, `PendingApproval()`, `ScheduleConfig`, defaults |
| `config/parse.go` | Validation for `autonomy` and `schedule.run` |
| `loop/cook_completion.go` | Merge gating via `canMergeStage()` — mode hook point (see note below) |
| `loop/cook_merge.go` | `canMergeStage()`, `resolveMergeMode()`, `parkPendingReview()` |
| `loop/control.go` | `"autonomy"` action case dispatches to `controlAutonomy()` |
| `loop/control_orders.go` | `controlAutonomy()` definition |
| `loop/control_review.go` | Merge-related code for pending reviews |
| `loop/state_snapshot.go` | `LoopState.Autonomy` field |
| `loop/stamp_status.go` | stamps `Autonomy` into status.json |
| `internal/statusfile/statusfile.go` | `Status.Autonomy` field |
| `internal/schemadoc/specs.go` | `"autonomy"` field doc |
| `internal/snapshot/types.go` | `Snapshot.Autonomy` field |
| `internal/snapshot/snapshot_builder.go` | maps state → snapshot |
| `internal/snapshot/fixture_test.go` | `state.Autonomy` in test fixtures |
| `internal/snapshot/testdata/*/expected.md` | golden files containing `"autonomy"` |
| `server/server.go` | `handleConfig()` response |
| `server/ws_hub.go` | `validActions` map (includes `"autonomy"`) |
| `startup/firstrun.go` | scaffolded `.noodle.toml` template |
| `generate/skill_noodle.go` | generated docs table — `"autonomy"` and `"schedule.run"` rows |
| `scripts/sandbox.sh` | example config |
| **Tests**: `config_test.go`, `loop_test.go`, `log_test.go`, `control_test.go`, `snapshot_test.go`, `integration_test.go`, `firstrun_test.go`, `smoke_test.go`, `helpers_test.go`, `skill_noodle_test.go`, `fixture_test.go` | |
| **UI**: `generated-types.ts` (auto-generated — regenerate, don't hand-edit), `types.ts`, `api.ts`, `api.test.ts`, `types.test.ts`, `test-utils.ts`, `Dashboard.tsx`, `TaskEditor.test.tsx` | |

**Note — merge gating flow has changed:** `PendingApproval()` is defined on `Config` but **never called from loop code**. The merge flow now works through `canMergeStage()` (per-task-type via skill registry) → `resolveMergeMode()` → merge queue or `parkPendingReview()`. For supervised/manual modes to park all merges, the mode check must hook into `cook_completion.go` where `canMerge` is evaluated (around line 139), NOT replace a `PendingApproval()` call.

## Applicable Skills

- `go-best-practices` — all Go phases
- `testing` — phases with test changes
- `ts-best-practices` — UI phase
- `react-best-practices` — UI phase
- `skill-creator` — skill file updates

## Phases

1. [[plans/68-unified-involvement-levels/phase-01-atomic-swap]]
2. [[plans/68-unified-involvement-levels/phase-02-dispatch-and-schedule-gating]]
3. [[plans/68-unified-involvement-levels/phase-03-mise-and-skills]]
4. [[plans/68-unified-involvement-levels/phase-04-web-ui]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
cd ui && npm run build && npm run typecheck
```
