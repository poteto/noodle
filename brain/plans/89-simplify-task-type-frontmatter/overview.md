---
id: 89
created: 2026-03-01
status: active
---

# Plan 89 — Simplify task type frontmatter

## Context

Task type skills declare scheduling and execution metadata under a `noodle:` frontmatter block with three fields:

```yaml
noodle:
  schedule: "When to schedule this"
  permissions:
    merge: false
  domain_skill: backlog
```

All three can be eliminated or replaced:

- **`permissions.merge`** — static declaration of "does this skill produce mergeable code." The mode system handles merge *approval*; whether there are changes to merge is a question the worktree answers at runtime via `countUnmergedCommits` (already exists internally).
- **`domain_skill`** — pass-through used by one skill (execute → backlog). The dispatcher mechanically bundles two skill prompts. The executing agent can invoke `Skill(backlog)` directly instead.
- **`schedule`** — the only field that matters. Once the other two are gone, the `noodle:` wrapper holds a single string. Promote to top-level.

End state:

```yaml
---
name: execute
description: Implementation methodology
schedule: "When backlog items with linked plans are ready"
---
```

`IsTaskType()` becomes `return f.Schedule != ""`. `NoodleMeta` is deleted.

## Scope

**In scope:**
- Add `HasUnmergedCommits(name string) (bool, error)` to `WorktreeManager` interface
- Replace `canMergeStage()` with runtime `worktreeHasChanges()` (two-path: sync result + local worktree)
- Remove `DomainSkill` from dispatch pipeline (registry, loop, dispatcher)
- Delete `NoodleMeta` struct entirely — promote `Schedule` to `Frontmatter`, delete `Permissions`, `CanMerge`, `DomainSkill`
- Update all skill frontmatter files to top-level `schedule:`
- Update execute skill body to reference backlog directly
- Update skill documentation template and brain vault

**Out of scope:**
- Mode gating (auto/supervised/manual) — orthogonal, already works
- Scheduler skill logic — reads `schedule` from mise.json, unaffected by nesting change
- Brain archive files — historical references stay as-is

## Constraints

- Runtime merge check must go upstream of `persistMergeMetadata` / merge queue — not inside `mergeCookWorktree`.
- `HasUnmergedCommits` must return `(bool, error)` — git failures must not silently become "no changes" (adversarial review finding).
- `worktreeHasChanges` must check sync result first (sprites remote branches), then local worktree (adversarial review finding).
- `fakeWorktree` and `noOpWorktree` both implement `WorktreeManager` — both need the new method.

## Alternatives considered

1. **Keep `permissions.merge` alongside mode** — rejected. Static declaration adds no information that runtime detection doesn't provide.
2. **Keep `domain_skill` but promote `schedule`** — rejected. Half-measure. `DomainSkill` is dispatch-time infrastructure for something an agent can do itself.
3. **Keep `noodle:` wrapper with just `schedule:`** — rejected. A wrapper for a single field is pure overhead.
4. **Check for changes inside `mergeCookWorktree`** — rejected. By then we've already set stage status to "merging" and enqueued a merge request.

## Applicable skills

- `go-best-practices` — Go patterns for interface and struct changes
- `testing` — test rewrites
- `skill-creator` — updating skill files and docs template

## Phases

1. [[plans/89-simplify-task-type-frontmatter/phase-01-worktree-interface]]
2. [[plans/89-simplify-task-type-frontmatter/phase-02-runtime-merge-detection]]
3. [[plans/89-simplify-task-type-frontmatter/phase-03-remove-domain-skill]]
4. [[plans/89-simplify-task-type-frontmatter/phase-04-delete-noodle-meta]]
5. [[plans/89-simplify-task-type-frontmatter/phase-05-update-skills-and-docs]]
6. [[plans/89-simplify-task-type-frontmatter/phase-06-update-brain]]

## Verification

```
pnpm check
go test ./skill/... ./internal/taskreg/... ./loop/... ./worktree/... ./dispatcher/... ./mise/...
```
