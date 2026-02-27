Back to [[plans/68-unified-involvement-levels/overview]]

# Phase 2: Delete old fields, swap all consumers

## Goal

Delete `autonomy`, `schedule.run`, and all their infrastructure. Replace every consumer with `mode`. This is a single atomic phase — no intermediate compile states.

## Changes

### Deletions from config/config.go
- Delete `AutonomyAuto`, `AutonomyApprove` constants
- Delete `Autonomy string` field and `PendingApproval()` method
- Delete `ScheduleConfig` struct and `Schedule ScheduleConfig` field
- Delete autonomy/schedule.run validation from `validateParsedValues()`
- Delete autonomy/schedule.run defaults from `applyDefaults()`
- Add: `Mode` default to `ModeAuto` in `applyDefaults()`, `Mode` validation in `validateParsedValues()`

### Loop consumers
- **loop/cook_completion.go**: `l.config.PendingApproval()` → `!l.config.AutoMerge()`
- **loop/control.go**: delete `controlAutonomy()`. Replace `case "autonomy":` with `case "mode":` calling new `controlMode()` (validates against the three mode values, sets `l.config.Mode`). Update `validActions` comment if any.
- **loop/state_snapshot.go**: `LoopState.Autonomy string` → `LoopState.Mode string`, populated from `l.config.Mode`
- **loop/stamp_status.go**: stamp `Mode` instead of `Autonomy`

### Infrastructure consumers
- **internal/statusfile/statusfile.go**: `Status.Autonomy string` → `Status.Mode string`
- **internal/schemadoc/specs.go**: `"autonomy"` field doc → `"mode"` with description "current mode (auto, supervised, manual)"
- **internal/snapshot/types.go**: `Snapshot.Autonomy string \`json:"autonomy"\`` → `Snapshot.Mode string \`json:"mode"\``
- **internal/snapshot/snapshot.go**: `state.Autonomy` → `state.Mode`
- **server/server.go**: `handleConfig()` — `"autonomy": s.config.Autonomy` → `"mode": s.config.Mode`. Replace `"autonomy"` with `"mode"` in `validActions`.
- **startup/firstrun.go**: `.noodle.toml` template — `autonomy = "auto"` → `mode = "auto"`, remove `[schedule]` section
- **scripts/sandbox.sh**: update config example

### Test migration (mechanical find-replace)
- **config/config_test.go**: all `autonomy`/`schedule.run` references → `mode`
- **loop/loop_test.go**: `cfg.Autonomy = "approve"` → `cfg.Mode = config.ModeSupervised`, `status.Autonomy` → `status.Mode`
- **loop/log_test.go**: same pattern
- **loop/control_test.go**: autonomy control tests → mode control tests
- **loop/integration_test.go**: snapshot assertions
- **internal/snapshot/snapshot_test.go**: `Autonomy: "auto"` → `Mode: "auto"`, `snap.Autonomy` → `snap.Mode`
- **startup/firstrun_test.go**: expected config content
- **e2e/helpers_test.go**, **e2e/smoke_test.go**: config snippets
- **generate/skill_noodle_test.go**: `requiredFields` — replace `"autonomy"` and `"schedule.run"` with `"mode"`

### Project config
- **.noodle.toml**: replace `autonomy = "approve"` and `[schedule]` section with `mode = "supervised"`

## Data Structures

- `LoopState.Mode string` (replaces `.Autonomy`)
- `Status.Mode string` (replaces `.Autonomy`)
- `Snapshot.Mode string \`json:"mode"\`` (replaces `.Autonomy`)

## Routing

Provider: `codex` | Model: `gpt-5.3-codex` — mechanical, high-volume find-and-replace

## Verification

### Static
- `go build ./...` — zero references to `Autonomy`, `PendingApproval`, `AutonomyAuto`, `AutonomyApprove`, `ScheduleConfig`, `Schedule.Run` in non-archive Go files
- `go vet ./...`
- `grep -rn "autonomy\|schedule\.run" --include="*.go" | grep -v archive/ | grep -v brain/` returns nothing

### Runtime
- `go test ./...` — all tests pass
- `sh scripts/lint-arch.sh`
