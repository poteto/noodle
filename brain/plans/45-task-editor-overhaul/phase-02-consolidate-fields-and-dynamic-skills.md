Back to [[plans/45-task-editor-overhaul/overview]]

# Phase 2: Consolidate fields and dynamic skills

## Goal

Replace the hardcoded `taskTypes` array and the redundant Type+Skill field pair with a single "Skill" field backed by dynamically discovered task types. Show the skill name and a middle-truncated description. Handle empty and unknown skill lists gracefully.

## Changes

### `tui/model.go` / `tui/model.go:Options`
- Add `TaskTypes []TaskTypeOption` to `Options` struct
- Store on `Model`, pass to `TaskEditor.OpenNew`/`OpenEdit`

### `cmd_start.go` (or wherever `runTUI` is called)
- Resolve task types via `app.SkillResolver().DiscoverTaskTypes()` at startup
- Map `[]skill.SkillMeta` → `[]TaskTypeOption` and pass into `Options`
- If discovery returns empty, include a synthetic "execute" entry so the form is always usable

### `tui/task_editor.go`
- Define `TaskTypeOption{Key string, Description string}`
- Remove hardcoded `taskTypes` var, `fieldSkill` constant, `skill string` field
- Replace `taskType int` and `skill string` with `skill int` (index into options) and `skillOptions []TaskTypeOption`
- Reduce `fieldCount` from 5 to 4: Prompt(0), Skill(1), Model(2), Provider(3)
- `OpenNew(opts []TaskTypeOption)` / `OpenEdit(item, opts)` — store options, set skill index
- `OpenEdit`: if item's TaskKey not found in options, append a synthetic entry for it (preserve unresolved state instead of coercing to index 0)
- `Submit`: read `skillOptions[e.skill].Key` as TaskKey. Guard: if options empty, return nil (no panic)
- `Render`: show `skillName — truncated description…` using `ansi.Truncate()` for the description portion

### Refresh policy
- Skills are resolved once at TUI startup. No hot-reload — consistent with current snapshot-based model. Plan #38 (resilient-skill-resolution) will add fsnotify; the TUI can subscribe to a refresh message then.

## Data structures

- `TaskTypeOption{Key string, Description string}` — lightweight value type

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical refactor with clear spec |

## Verification

### Static
- `go test ./tui/... && go vet ./...`
- Test: `OpenNew` with empty options → Submit returns nil (no panic)
- Test: `OpenEdit` with unknown TaskKey → synthetic entry appended, skill index points to it
- Test: `OpenEdit` with known TaskKey → correct skill index selected
- Test: Submit produces correct `ControlCommand` with TaskKey from options
- Test: field count is 4, Tab cycles through Prompt→Skill→Model→Provider

### Runtime
- Launch TUI, press `n` — Skill field shows dynamically discovered types
- Skill field shows `name — description…` format
- Tab cycles through 4 fields
- Left/right cycles skill options
