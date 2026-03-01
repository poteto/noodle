Back to [[plans/89-simplify-task-type-frontmatter/overview]]

# Phase 4 — Delete `NoodleMeta` and promote `schedule`

## Goal

With `Permissions` unused (phase 2) and `DomainSkill` deleted (phase 3), `NoodleMeta` holds only `Schedule`. Promote `Schedule` to a top-level `Frontmatter` field and delete `NoodleMeta` entirely.

## Changes

**`skill/frontmatter.go`**:
- Add `Schedule string \`yaml:"schedule,omitempty"\`` to `Frontmatter` struct
- Delete `NoodleMeta` struct
- Delete `Permissions` struct and `CanMerge()` method
- Delete `Noodle *NoodleMeta` field from `Frontmatter`
- Change `IsTaskType()` from `return f.Noodle != nil` to `return f.Schedule != ""`
- Update validation: remove `noodle.schedule is required` error (presence of `schedule` IS the discriminator, no wrapper to validate)

**`skill/frontmatter_test.go`**:
- Rewrite all tests to use top-level `schedule:` instead of `noodle: schedule:`
- Remove all `Permissions`/`CanMerge` test cases
- Remove all `DomainSkill` test cases
- Update `IsTaskType` assertions

**`skill/resolver_test.go`**:
- Update test YAML fixtures from `noodle: schedule:` to `schedule:`

**`internal/taskreg/registry.go`**:
- Read `s.Frontmatter.Schedule` instead of `s.Frontmatter.Noodle.Schedule`
- Remove `CanMerge` field from `TaskType`

**`internal/taskreg/registry_test.go`**:
- Update test fixtures — remove `noodle:` nesting, use top-level `schedule:`
- Remove `CanMerge` assertions

**`mise/types.go`**:
- Remove `CanMerge bool` from `TaskTypeSummary`

**`internal/schemadoc/specs.go`**:
- Remove `task_types[].can_merge` entry

**`loop/task_types.go`**:
- Remove `CanMerge: tt.CanMerge` from `registryToTaskTypeSummaries()`

## Data structures

- `NoodleMeta` struct deleted
- `Permissions` struct deleted
- `CanMerge` field removed from `TaskType` and `TaskTypeSummary`
- `Schedule` promoted to `Frontmatter`

## Principles

- **subtract-before-you-add** — deleting two structs and a nesting layer
- **foundational-thinking** — the data model simplifies to a single discriminator field
- **redesign-from-first-principles** — if building from scratch, task type registration would be a single `schedule` field

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

## Verification

```
go test ./skill/... ./internal/taskreg/... ./loop/... ./mise/...
go vet ./...
```

- Confirm `NoodleMeta` and `Permissions` are fully gone (grep for both in Go source)
- Confirm `CanMerge` appears nowhere except tests that explicitly test its absence
- Confirm `IsTaskType()` returns true for skills with `schedule:` and false without
