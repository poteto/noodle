# Team Execution

The lead orchestrates — it does NOT implement. Research via sub-agents, delegate all implementation to teammates.

1. **Lead worktree**: Use current worktree if already in one, otherwise `noodle worktree create plan-<N>-lead`
2. **Team**: `TeamCreate` — all tasks go through this team's task list
3. **Per-teammate worktrees**: `noodle worktree create plan-<N>-phase-<M>`
4. **Spawn teammates**: `Task` with `mode: "bypassPermissions"`, `team_name`, worktree path, scope, and domain skill name. Always spawn fresh agents to keep context clean.
5. **Teammates commit** on their own branches
6. **Review before merging**: Spawn a review agent to check each teammate's work against the plan before merging
7. **Merge teammates into lead** (not main):
   ```bash
   git -C .worktrees/plan-<N>-lead merge <teammate-branch>
   noodle worktree cleanup plan-<N>-phase-<M>
   ```
8. **Verify integrated result** in lead worktree (see Verify below)
9. **Merge lead to main**: `noodle worktree merge plan-<N>-lead`

Foundational phases that later phases depend on: execute first, commit in lead worktree, then create teammate worktrees from that point.
