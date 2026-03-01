Back to [[plans/89-runtime-merge-detection/overview]]

# Phase 5 — Update brain vault references

## Goal

Update brain docs that reference `permissions.merge` so they reflect the new runtime detection approach. Archive files stay as-is (they're historical).

## Changes

**`brain/vision.md`**:
- Update or remove the `merge: false` example if it's used to explain the permissions system

**`brain/architecture.md`**:
- Update the `merge: false` example to explain runtime merge detection instead

**`brain/codebase/runtime-routing-owned-by-orders.md`**:
- Review — this note already established the pattern of moving runtime concerns out of skill metadata. Add a sentence noting that merge detection followed the same pattern.

**Create `brain/codebase/runtime-merge-detection.md`**:
- Document the runtime detection approach: `WorktreeManager.HasUnmergedCommits` replaces static `permissions.merge`
- Note the interaction with mode gating: mode controls approval, runtime detection controls whether there's anything to approve

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

## Verification

- Brain files render correctly as Obsidian notes
- No broken wikilinks
- Grep for `permissions.*merge` in brain/ — only archive files should match
