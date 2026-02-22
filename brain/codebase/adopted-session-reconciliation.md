# Adopted Session Reconciliation

- Startup adoption must carry full lifecycle semantics, not just concurrency accounting.
- If an adopted session finishes after restart, the loop still needs to run merge, backlog `done`, and queue removal.
- Completed queue items must be removed from `.noodle/queue.json` after successful merge to prevent accidental respawn from stale queue entries.
- Stale adopted entries with no matching queue item should be dropped, not processed through review.
- When tests hang around scheduling completion, check whether review/taster is waiting on a `Done()` channel that is never closed.
