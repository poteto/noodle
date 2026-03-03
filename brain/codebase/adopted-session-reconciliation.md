# Adopted Session Reconciliation

- Startup adoption must carry full lifecycle semantics, not just concurrency accounting — merge, backlog `done`, and order removal still apply.
- Stale adopted entries with no matching order should be dropped, not processed through review.
- Oops sessions need the same adoption behavior; otherwise restarts spawn duplicate oops cooks.
- Startup reconcile must reset stale non-schedule stage statuses `active -> pending` when no session was recovered; otherwise orders become permanently busy.
- Session metadata repair: when canonical claims say a non-schedule session is terminal but PID liveness reports alive, treat as terminal and close the process. Keep scheduler sessions persistent.
- Loop maintenance should run monitor each cycle and bridge terminal `meta.json` statuses back into completion handling so stale active cooks are advanced/failed without waiting on `Done()`.

See also [[codebase/worktree-gotchas]], [[codebase/tmux-shutdown-summary-buffer-race]], [[principles/fix-root-causes]]
