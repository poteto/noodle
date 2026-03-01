Back to [[plans/89-simplify-task-type-frontmatter/overview]]

# Phase 6 — Update brain vault

## Goal

Update brain docs to reflect the simplified frontmatter and runtime merge detection. Archive files stay as-is.

## Changes

**`brain/vision.md`**:
- Update frontmatter examples from `noodle: schedule:` / `merge: false` to top-level `schedule:`

**`brain/architecture.md`**:
- Update frontmatter examples
- Replace merge permission explanation with runtime merge detection explanation

**`brain/codebase/runtime-routing-owned-by-orders.md`**:
- Add note that merge detection followed the same pattern: runtime concern removed from skill metadata

**Create `brain/codebase/runtime-merge-detection.md`**:
- Document: `WorktreeManager.HasUnmergedCommits` replaces static `permissions.merge`
- Two-path check: sync result (remote) then local worktree
- Interaction with mode gating: mode controls approval, runtime detection controls whether there's anything to approve

## Principles

- **encode-lessons-in-structure** — document the pattern so future features follow it

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

## Verification

- Brain files render correctly as Obsidian notes
- No broken wikilinks
- Grep for `noodle:` in brain/ (non-archive) — should reference the old format only in historical context
- Grep for `permissions.*merge` in brain/ — only archive files should match
