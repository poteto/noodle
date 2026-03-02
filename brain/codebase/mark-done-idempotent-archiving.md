# mark-done Idempotent Archiving

- `scripts/mark-done.sh` is intentionally best-effort and idempotent.
- The command should not fail when a todo is already completed, already archived, or missing from `brain/todos.md`.
- Expected behavior is convergence:
- move `brain/plans/<id>-*` to `brain/archive/plans/` when present
- update `brain/plans/index.md` links from `[[plans/...]]` to `[[archive/plans/...]]` for that ID
- if the todo line exists in `brain/todos.md` (`[ ]` or `[x]`), move it to `brain/archive/completed_todos.md`
- normalize stale `[[plans/...]]` links to `[[archive/plans/...]]` for that ID in completed todos
- Re-running `pnpm mark-done <id>` should succeed and leave state unchanged once converged.
