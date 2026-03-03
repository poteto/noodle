# Runtime Routing Owned By Orders

- Runtime selection belongs to order stages, not skill frontmatter.
- `loop.spawnCook` sends `DispatchRequest.Runtime` from the stage `runtime` field; when stage runtime is empty it currently falls back to `"process"` (not `config.Runtime.Default`).
- `dispatchSession` only consults `config.Runtime.Default` when `DispatchRequest.Runtime` is empty (for example, schedule-stage dispatch paths).
- Config parsing/validation accepts runtime names `process`, `sprites`, and `cursor`, but default runtime wiring currently registers `process` and optional `sprites`; cursor dispatch is not wired in `defaultDependencies`.
- Skill frontmatter and task registry (`taskreg.TaskType`) should not carry runtime fields.
- Keep non-functional backend tests optional during stub phases; compile-time interface assertions are enough until behavior exists.
- Merge detection followed the same pattern: moved from static skill metadata (`permissions.merge`) to runtime detection via `WorktreeManager.HasUnmergedCommits`.
- `worktreeHasChanges()` checks three paths: (1) persisted merge metadata (crash recovery via `Extra["merge_branch"]`), (2) sync result (remote branches via `spawn.json`), (3) local worktree via `git cherry`.
- Mode gating is orthogonal: mode controls merge approval, runtime detection controls whether there's anything to approve.

See also [[archive/plans/27-remote-dispatchers/overview]], [[principles/boundary-discipline]], [[principles/subtract-before-you-add]]
