# Adopted Session Reconciliation

- Startup adoption must carry full lifecycle semantics, not just concurrency accounting.
- If an adopted session finishes after restart, the loop still needs to run merge, backlog `done`, and queue removal.
- Completed queue items must be removed from `.noodle/queue.json` after successful merge to prevent accidental respawn from stale queue entries.
- Stale adopted entries with no matching queue item should be dropped, not processed through review.
- When tests hang around scheduling completion, check whether review/quality is waiting on a `Done()` channel that is never closed.
- Oops sessions need the same startup adoption behavior; otherwise each restart can spawn duplicate oops cooks for the same issue.
- Startup reconcile must reset stale non-schedule stage statuses `active -> pending` when no session was recovered for that order; otherwise orders become permanently busy and the scheduler can create duplicate recovery orders for the same backlog item.
- Session metadata repair: when canonical claims say a non-schedule session is terminal (`completed`/`failed`) but PID liveness still reports alive, treat it as terminal for state and close the lingering process. Keep scheduler sessions (`skill: schedule`) persistent.
- Loop maintenance should run monitor each cycle and bridge terminal `meta.json` statuses back into completion handling so stale active cooks are advanced/failed without waiting on `Done()`.
- Embedded runtime worktree operations should run in quiet mode so CLI/provisioning logs never bleed into the Bubble Tea frame.

See also [[codebase/worktree-gotchas]], [[codebase/tmux-shutdown-summary-buffer-race]], [[principles/fix-root-causes]]
