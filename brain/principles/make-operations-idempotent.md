# Make Operations Idempotent

**Principle:** Design operations so they converge to the correct state regardless of how many times they run or where they start from. Every state-mutating operation should answer: "What happens if this runs twice? What happens if the previous run crashed halfway?"

## Why

CLI commands, lifecycle operations, and scheduling loops run in environments where crashes, restarts, and retries are normal. If an operation leaves partial state that causes a different outcome on re-execution, every restart becomes a debugging session.

## The Pattern

- **Convergent startup:** `noodle start` scans tmux, cleans stale state, adopts live sessions — converging to the correct state regardless of what the previous run left behind.
- **Content-based cleanup:** `noodle worktree prune` uses patch equivalence (`git cherry`), not commit ancestry, to determine safety. Re-running converges to the same state.
- **Self-healing locks:** Merge lockfile with PID-based stale lock detection ensures orphaned locks from crashed processes are automatically recovered.
- **Idempotent scheduling:** Failed cooks respawn as `task-1-recover-1` (up to max retries). Mise regeneration after each cook ensures fresh input, preventing stale state accumulation.

## The Test

Before shipping a state-mutating operation, ask:
1. What happens if this runs twice in a row?
2. What happens if the previous run crashed at every possible point?
3. Does re-execution converge to the same end state?

If any answer is "it depends on what state was left behind," the operation needs a reconciliation step.

## Relationship to Other Principles

- Extends [[principles/fix-root-causes]] by preventing a class of root causes (partial completion) at design time
- Complements [[principles/suspect-state-before-code]] by making state self-correcting rather than requiring debugging
- Distinct from [[principles/encode-lessons-in-structure]] (which is about how to encode rules, not operation design)
- Distinct from [[principles/boundary-discipline]] (which is about where validation lives, not convergence behavior)

See also [[codebase/adopted-session-reconciliation]], [[codebase/worktree-prune-patch-equivalence]], [[codebase/worktree-gotchas]]
