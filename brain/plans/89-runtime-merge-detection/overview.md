---
id: 89
created: 2026-03-01
status: active
---

# Plan 89 — Runtime Merge Detection

## Context

The `permissions: merge: boolean` field in skill frontmatter is a static declaration that tells the loop whether a stage produces mergeable code. Skills like `quality` set `merge: false` because they don't produce commits — there's nothing to merge.

With the mode system (`auto` / `supervised` / `manual`) now handling merge *approval* gating, `permissions.merge` only answers one question: "does this stage have changes?" That's a question the worktree can answer at runtime by checking for unmerged commits — which `worktree.App.countUnmergedCommits` already does internally.

Replacing the static declaration with runtime detection:
- Removes a frontmatter field, a struct, a registry field, and all their plumbing
- Handles the "execute stage that produced no commits" case correctly (today it enters the merge pipeline and no-ops inside `Merge()`)
- Is inherently idempotent — same worktree state always produces the same result

## Scope

**In scope:**
- Add `HasUnmergedCommits(name string) bool` to `WorktreeManager` interface
- Replace `canMergeStage()` (static registry lookup) with `worktreeHasChanges()` (runtime git check)
- Delete `Permissions` struct and `CanMerge` from frontmatter, taskreg, mise types, schemadoc
- Remove `permissions:` from skill frontmatter files
- Update skill documentation template (`generate/skill_noodle.go`)
- Update/rewrite affected tests

**Out of scope:**
- Mode gating (auto/supervised/manual merge approval) — orthogonal, already works
- Brain archive files — historical references stay as-is
- Brain vision/architecture docs — update references but don't restructure

## Constraints

- `worktree.Merge()` already handles 0 commits gracefully (`commands.go:108-113`), but the code *before* `mergeCookWorktree` does work we want to skip: `persistMergeMetadata` sets status to "merging", merge queue gets an entry, etc. The runtime check must go upstream of that.
- The `WorktreeManager` interface is small (4 methods). Adding one is low-risk.
- `fakeWorktree` in tests needs the new method — default to "has changes" to preserve existing test behavior.

## Alternatives considered

1. **Keep `permissions.merge` alongside mode** — rejected. Two systems answering the same question. The static declaration adds no information that runtime detection doesn't provide, and handles the "execute with no commits" case worse.
2. **Check for changes inside `mergeCookWorktree` instead of upstream** — rejected. By the time we're inside that function, we've already set stage status to "merging" and enqueued a merge request. The check needs to happen before the merge pipeline.
3. **Remove the `Permissions` struct but keep a `produces_changes` semantic field** — rejected. Still a static declaration that can be wrong. Runtime detection is strictly more accurate.

## Applicable skills

- `go-best-practices` — Go patterns for interface changes
- `testing` — test rewrites
- `skill-creator` — updating skill frontmatter docs in `generate/skill_noodle.go`

## Phases

1. [[plans/89-runtime-merge-detection/phase-01-worktree-interface]]
2. [[plans/89-runtime-merge-detection/phase-02-replace-canmerge]]
3. [[plans/89-runtime-merge-detection/phase-03-delete-static-plumbing]]
4. [[plans/89-runtime-merge-detection/phase-04-update-docs-and-skills]]
5. [[plans/89-runtime-merge-detection/phase-05-update-brain]]

## Verification

```
pnpm check
go test ./skill/... ./internal/taskreg/... ./loop/... ./worktree/... ./mise/...
```
