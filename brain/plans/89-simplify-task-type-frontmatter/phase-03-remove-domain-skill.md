Back to [[plans/89-simplify-task-type-frontmatter/overview]]

# Phase 3 — Remove `DomainSkill` from dispatch pipeline

## Goal

Remove the mechanical domain skill bundling from the dispatch pipeline. The executing agent invokes domain skills directly via `Skill()` instead of having the dispatcher concatenate prompts.

## Changes

**`internal/taskreg/registry.go`**:
- Remove `DomainSkill` field from `TaskType` struct
- Remove `DomainSkill` assignment in `NewFromSkills`

**`internal/taskreg/registry_test.go`**:
- Delete `TestDomainSkillPropagated` test
- Remove `DomainSkill` assertions from other tests

**`loop/cook_spawn.go`**:
- Remove the `DomainSkill` conditional (lines 96-98)

**`dispatcher/types.go`**:
- Remove `DomainSkill` field from `DispatchRequest`

**`dispatcher/dispatch_helpers.go`**:
- Remove the `DomainSkill` branch from `resolveSkillBundle` — simplify to: if `SystemPrompt` is set use it, otherwise `loadSkillBundle`

**`dispatcher/skill_bundle.go`**:
- Delete `loadExecuteBundle` function entirely

**`.agents/skills/execute/SKILL.md`**:
- Remove `domain_skill: backlog` from frontmatter
- Add instruction in skill body to invoke `Skill(backlog)` for project-specific context

## Data structures

- `DomainSkill` removed from `TaskType` and `DispatchRequest`

## Principles

- **subtract-before-you-add** — removing infrastructure for something an agent does naturally
- **migrate-callers-then-delete-legacy-apis** — update the one caller (execute skill), delete the pipeline

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

## Verification

```
go test ./internal/taskreg/... ./loop/... ./dispatcher/...
go vet ./...
```

- Confirm `loadExecuteBundle` is fully deleted (grep)
- Confirm `DomainSkill` appears nowhere in Go source (grep)
- Confirm execute skill frontmatter no longer has `domain_skill:`
