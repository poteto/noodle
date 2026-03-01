# Runtime Merge Detection

- Merge detection uses runtime worktree state, not static skill frontmatter.
- `WorktreeManager.HasUnmergedCommits(name)` replaces the deleted `permissions.merge` declaration.
- `worktreeHasChanges()` in the loop checks three paths:
  1. Persisted merge metadata (crash recovery) — stage status is "merging" with `Extra["merge_branch"]`
  2. Sync result (remote branches) — `spawn.json` contains `type: "branch"`
  3. Local worktree — `HasUnmergedCommits` via `git cherry`
- Mode gating is orthogonal: mode controls merge approval, runtime detection controls whether there's anything to approve.

See also [[runtime-routing-owned-by-orders]], [[principles/subtract-before-you-add]]
