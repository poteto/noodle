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

## Alternatives Considered

1. **Keep two separate fields, wire up `schedule.run`**: Rejected — confusing matrix. Users think in involvement levels.
2. **Four modes (auto/supervised/manual/off)**: Rejected — "off" is just not running noodle.
3. **Mode as an object with per-behavior overrides**: Rejected — over-engineering. Per-skill `permissions.merge` covers fine-grained needs.

## All Consumers (must be updated atomically in phase 2)

| File | Usage |
|------|-------|
| `config/config.go` | Constants, field, PendingApproval(), ScheduleConfig, validation, defaults |
| `loop/cook_completion.go` | `PendingApproval()` merge gate |
| `loop/control.go` | `controlAutonomy()`, `"autonomy"` action case |
| `loop/state_snapshot.go` | `LoopState.Autonomy` field |
| `loop/stamp_status.go` | stamps `Autonomy` into status.json |
| `internal/statusfile/statusfile.go` | `Status.Autonomy` field |
| `internal/schemadoc/specs.go` | `"autonomy"` field doc |
| `internal/snapshot/types.go` | `Snapshot.Autonomy` field |
| `internal/snapshot/snapshot.go` | maps state → snapshot |
| `server/server.go` | `handleConfig()` response, `validActions` |
| `startup/firstrun.go` | scaffolded `.noodle.toml` template |
| `generate/skill_noodle.go` | generated docs table |
| `scripts/sandbox.sh` | example config |
| **Tests**: `config_test.go`, `loop_test.go`, `log_test.go`, `control_test.go`, `snapshot_test.go`, `integration_test.go`, `firstrun_test.go`, `smoke_test.go`, `helpers_test.go`, `skill_noodle_test.go` | |
| **UI**: `generated-types.ts`, `types.ts`, `api.ts`, `api.test.ts`, `types.test.ts`, `test-utils.ts`, `Board.tsx`, `Board.test.tsx` | |

## Applicable Skills

- `go-best-practices` — all Go phases
- `testing` — phases with test changes
- `ts-best-practices` — UI phase
- `react-best-practices` — UI phase
- `skill-creator` — skill file updates

## Phases

1. [[plans/68-unified-involvement-levels/phase-01-define-mode-type]]
2. [[plans/68-unified-involvement-levels/phase-02-delete-old-fields-swap-all-consumers]]
3. [[plans/68-unified-involvement-levels/phase-03-dispatch-and-schedule-gating]]
4. [[plans/68-unified-involvement-levels/phase-04-mise-and-skills]]
5. [[plans/68-unified-involvement-levels/phase-05-web-ui]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
cd ui && npm run build && npm run typecheck
```
