Back to [[plans/45-task-editor-overhaul/overview]]

# Phase 1: Consolidate fields and dynamic skills

## Goal

Replace the hardcoded `taskTypes` array and the redundant Type+Skill field pair with a single "Skill" field backed by dynamically discovered task types from the skill registry. Show the skill name and a middle-truncated description.

## Changes

### `tui/model.go` / `tui/model.go:Options`
- Add a `TaskTypes []TaskTypeOption` field to `Options` struct
- Store it on `Model` and pass it to `TaskEditor` when opening

### `tui/task_editor.go`
- Define `TaskTypeOption` struct: `Key string`, `Description string`
- Remove hardcoded `taskTypes` var and `fieldSkill` constant
- Remove `taskType int` and `skill string` fields from `TaskEditor`
- Add `skill int` (index into the passed-in options list) and `skillOptions []TaskTypeOption`
- Reduce `fieldCount` from 5 to 4: Prompt(0), Skill(1), Model(2), Provider(3)
- Update `OpenNew`/`OpenEdit` to accept and store the options list
- Update `Submit` to read `skillOptions[e.skill].Key` as the TaskKey
- Update `Render` to show skill name + truncated description (e.g. `execute — Impl…from backlog`)

### `cmd_start.go` (or wherever `NewModel` is called)
- Resolve task types via `skill.Resolver.DiscoverTaskTypes()` at startup
- Map `[]skill.SkillMeta` → `[]TaskTypeOption` and pass into `Options`

### String truncation
- Use `ansi.Truncate()` for the description to keep the line within form width
- Format: `skillName — truncated description…`

## Data structures

- `TaskTypeOption{Key string, Description string}` — lightweight value type for the dropdown

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical refactor with clear spec |

## Verification

### Static
- `go test ./tui/... && go vet ./...`
- Existing task editor tests pass (update assertions for 4 fields instead of 5)
- `Submit()` still produces correct `ControlCommand` with `TaskKey` and empty `Skill` override

### Runtime
- Launch TUI, press `n` — verify skill list shows dynamically discovered task types
- Skill field shows name + truncated description
- Tab cycles through 4 fields (not 5)
- Left/right cycles through skill options
- Submit creates queue item with correct TaskKey
