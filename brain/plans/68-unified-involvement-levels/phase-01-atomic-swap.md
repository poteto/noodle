Back to [[plans/68-unified-involvement-levels/overview]]

# Phase 1: Atomic swap — define Mode, delete old fields, migrate all consumers

## Goal

In a single phase, define the `Mode` type and constants, delete `autonomy` / `schedule.run` / `ScheduleConfig`, and replace every consumer across Go code, tests, config files, and the generator. The codebase compiles and all existing tests pass at the end of this phase.

## Changes

### config/config.go — subtract then add

**Delete:**
- `AutonomyAuto`, `AutonomyApprove` constants
- `Autonomy string` field
- `PendingApproval()` method
- `ScheduleConfig` struct and `Schedule ScheduleConfig` field
- Autonomy/schedule.run validation from `validateParsedValues()`
- Autonomy/schedule.run defaults from `applyDefaults()`

**Add:**
- Constants: `ModeAuto = "auto"`, `ModeSupervised = "supervised"`, `ModeManual = "manual"`
- Field: `Mode string \`toml:"mode"\``
- Query methods: `AutoMerge() bool` (true when `Mode == ModeAuto`), `AutoDispatch() bool` (true when `Mode != ModeManual`), `AutoSchedule() bool` (true when `Mode != ModeManual`)
- Default: `Mode` defaults to `ModeAuto` in `applyDefaults()`
- Validation: `Mode` must be one of the three constants in `validateParsedValues()`

### Loop consumers
- **loop/cook_completion.go**: `l.config.PendingApproval()` → `!l.config.AutoMerge()`
- **loop/control.go**: Delete `controlAutonomy()`. Replace `case "autonomy":` with `case "mode":` calling new `controlMode()` (validates against three mode values, sets `l.config.Mode`).
- **loop/state_snapshot.go**: `LoopState.Autonomy string` → `LoopState.Mode string`, populated from `l.config.Mode`
- **loop/stamp_status.go**: Stamp `Mode` instead of `Autonomy`

### Infrastructure consumers
- **internal/statusfile/statusfile.go**: `Status.Autonomy string` → `Status.Mode string`
- **internal/schemadoc/specs.go**: `"autonomy"` field doc → `"mode"` with description "current mode (auto, supervised, manual)"
- **internal/snapshot/types.go**: `Snapshot.Autonomy string \`json:"autonomy"\`` → `Snapshot.Mode string \`json:"mode"\``
- **internal/snapshot/snapshot.go**: `state.Autonomy` → `state.Mode`
- **server/server.go**: `handleConfig()` — `"autonomy": s.config.Autonomy` → `"mode": s.config.Mode`. Replace `"autonomy"` with `"mode"` in `validActions`.
- **startup/firstrun.go**: `.noodle.toml` template — `autonomy = "auto"` → `mode = "auto"`, remove `[schedule]` section
- **generate/skill_noodle.go**: Replace `"autonomy"` and `"schedule.run"` rows with `"mode"` row. Update any prose. Run `go generate ./generate/...` to regenerate `.agents/skills/noodle/SKILL.md`.
- **scripts/sandbox.sh**: Update config example

### Test migration
- **config/config_test.go**: All `autonomy`/`schedule.run` references → `mode`. Add tests for `AutoMerge()`, `AutoDispatch()`, `AutoSchedule()` across all three modes (9 cases).
- **loop/loop_test.go**: `cfg.Autonomy = "approve"` → `cfg.Mode = config.ModeSupervised`, `status.Autonomy` → `status.Mode`
- **loop/log_test.go**: Same pattern
- **loop/control_test.go**: Autonomy control tests → mode control tests
- **loop/integration_test.go**: Snapshot assertions
- **internal/snapshot/snapshot_test.go**: `Autonomy: "auto"` → `Mode: "auto"`, `snap.Autonomy` → `snap.Mode`
- **internal/snapshot/fixture_test.go**: `state.Autonomy` → `state.Mode`
- **internal/snapshot/testdata/*/expected.md**: Update golden files — `"autonomy"` → `"mode"` in all expected output
- **startup/firstrun_test.go**: Expected config content
- **e2e/helpers_test.go**, **e2e/smoke_test.go**: Config snippets
- **generate/skill_noodle_test.go**: `requiredFields` — replace `"autonomy"` and `"schedule.run"` with `"mode"`

### Project config
- **.noodle.toml**: Replace `autonomy = "approve"` and `[schedule]` section with `mode = "supervised"`

## Data Structures

- `ModeAuto`, `ModeSupervised`, `ModeManual` string constants
- `Config.Mode string`
- `LoopState.Mode string` (replaces `.Autonomy`)
- `Status.Mode string` (replaces `.Autonomy`)
- `Snapshot.Mode string \`json:"mode"\`` (replaces `.Autonomy`)

## Routing

Provider: `codex` | Model: `gpt-5.3-codex` — mechanical, high-volume find-and-replace with clear spec

## Verification

### Static
- `go build ./...` — zero references to `Autonomy`, `PendingApproval`, `AutonomyAuto`, `AutonomyApprove`, `ScheduleConfig`, `Schedule.Run` in non-archive Go files
- `go vet ./...`
- `grep -rn "autonomy\|schedule\.run" --include="*.go" | grep -v archive/ | grep -v brain/` returns nothing

### Runtime
- `go test ./...` — all tests pass
- `sh scripts/lint-arch.sh`
- `go generate ./generate/...` produces up-to-date SKILL.md
