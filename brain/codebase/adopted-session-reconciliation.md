# Adopted Session Reconciliation

- Startup adoption must carry full lifecycle semantics, not just concurrency accounting.
- If an adopted session finishes after restart, the loop still needs to run merge, backlog `done`, and queue removal.
- Completed queue items must be removed from `.noodle/queue.json` after successful merge to prevent accidental respawn from stale queue entries.
- Stale adopted entries with no matching queue item should be dropped, not processed through review.
- When tests hang around scheduling completion, check whether review/quality is waiting on a `Done()` channel that is never closed.
- Oops sessions need the same startup adoption behavior; otherwise each restart can spawn duplicate oops cooks for the same issue.
- Embedded runtime worktree operations should run in quiet mode so CLI/provisioning logs never bleed into the Bubble Tea frame.

See also [[codebase/worktree-gotchas]], [[principles/suspect-state-before-code]]
