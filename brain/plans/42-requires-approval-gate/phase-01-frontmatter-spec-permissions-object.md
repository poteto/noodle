Back to [[plans/42-requires-approval-gate/overview]]

# Phase 1: Frontmatter Spec — Permissions Object

## Goal

Replace `blocking` with a `permissions` object in the noodle frontmatter. Delete `blocking` entirely — scheduling exclusivity is removed as a concept. The only permission to start is `merge` (defaults to `true`). The struct is extensible for future permissions.

```yaml
noodle:
  permissions:
    merge: false
  schedule: "After each cook session completes"
```

## Changes

### `skill/frontmatter.go`

Replace `Blocking bool` in `NoodleMeta` with a `Permissions` struct:

```go
type NoodleMeta struct {
    Permissions Permissions `yaml:"permissions"`
    Schedule    string      `yaml:"schedule"`
}

type Permissions struct {
    Merge *bool `yaml:"merge,omitempty"`
}

func (p Permissions) CanMerge() bool {
    if p.Merge == nil {
        return true
    }
    return *p.Merge
}
```

Use `*bool` so omitted = default `true`, explicit `false` = no merge permission.

### `internal/taskreg/registry.go`

Remove `Blocking bool` from `TaskType`. Add `CanMerge bool` — set from `s.Frontmatter.Noodle.Permissions.CanMerge()`.

### `loop/task_types.go`

Delete `isBlockingQueueItem` — no longer needed. Update `registryToTaskTypeSummaries` to drop the `Blocking` field.

### `loop/cycle_spawn_plan.go`

Remove `BlockingActive bool` and `IsBlocking func(QueueItem) bool` from `spawnPlanInput`. Remove the `blockingActive` check and `IsBlocking` gate in `planSpawnItems`. Tasks now always run concurrently up to capacity.

### `loop/loop.go`

Remove all code that computes `blockingActive` and passes `IsBlocking` to `spawnPlanInput`. Simplify spawn planning setup.

### `mise/types.go`

Remove `Blocking` field from `TaskTypeSummary`. This is the JSON contract agents read — `blocking` no longer exists as a concept.

### `brain/vision/noodle.md`

Remove `blocking: true` and `blocking: false` from all frontmatter examples. Add a `permissions` example showing `merge: false` for a task that parks for human review.

### All SKILL.md files with `blocking:` in noodle frontmatter

Remove all `blocking:` lines. No skill currently needs `merge: false`, so no `permissions:` block is needed yet.

Skills to update (all in `.agents/skills/`):
- `prioritize` — remove `blocking: true`
- `execute` — remove `blocking: false`
- `review` — remove `blocking: false`
- `meditate`, `oops`, `debate`, `reflect` — remove `blocking: false`

### Tests

- `skill/frontmatter_test.go` — test `permissions.merge`: explicit true, explicit false, omitted (defaults true), entire permissions block omitted
- `internal/taskreg/registry_test.go` — update registry construction to test `CanMerge`, remove `Blocking` assertions
- `loop/cycle_spawn_plan_test.go` (if exists) — remove blocking-related test cases

## Routing

Provider: `codex` | Model: `gpt-5.3-codex` — struct changes and mechanical deletions.

## Verification

```sh
go test ./skill/... ./internal/taskreg/... ./loop/... ./mise/...
go vet ./...
# Verify blocking is fully removed:
rg -i 'blocking' --type go --type md -g '!brain/archived_plans/*' -g '!brain/todos.md' -g '!brain/plans/42-*'
```
