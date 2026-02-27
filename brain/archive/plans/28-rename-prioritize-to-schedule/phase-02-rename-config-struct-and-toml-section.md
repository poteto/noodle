Back to [[archive/plans/28-rename-prioritize-to-schedule/overview]]

# Phase 2: Rename config struct and TOML section

## Goal

Rename the config type, struct field, TOML section, and all metadata/validation references from "prioritize" to "schedule". After this phase, `.noodle.toml` uses `[schedule]` and Go code uses `ScheduleConfig`.

## Changes

### `config/config.go`

- `PrioritizeConfig` type renamed to `ScheduleConfig`
- `Config.Prioritize` field renamed to `Config.Schedule`, TOML tag changed to `toml:"schedule"`
- `DefaultConfig()`: field name `Prioritize` becomes `Schedule`, default skill value `"prioritize"` becomes `"schedule"`
- `applyDefaultsFromMetadata()`: all `metadata.IsDefined("prioritize", ...)` calls become `metadata.IsDefined("schedule", ...)`, all `config.Prioritize` references become `config.Schedule`, default skill value `"prioritize"` becomes `"schedule"`
- `validateParsedValues()`: `config.Prioritize` references become `config.Schedule`, error messages change from `"prioritize.run"` / `"prioritize.skill"` to `"schedule.run"` / `"schedule.skill"`

### `.noodle.toml`

- `[prioritize]` section header renamed to `[schedule]`
- `skill = "prioritize"` becomes `skill = "schedule"`

## Data structures

- `PrioritizeConfig` renamed to `ScheduleConfig` (same fields: `Skill`, `Run`, `Model`)

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical rename of type, field, TOML tags, and string literals |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- `config.Load(".noodle.toml")` parses the `[schedule]` section into `Config.Schedule`
- `DefaultConfig().Schedule.Skill` equals `"schedule"`
- Validation rejects `schedule.run = "bogus"` with error mentioning `schedule.run`
