# Projection Owns Snapshot Active Order IDs

Phase 7 of [[plans/115-canonical-state-convergence/overview]] removed the last snapshot-structure fallback in `internal/snapshot`.

`/api/snapshot` and websocket snapshot payloads now take structural order fields from `loop.LoopState.Projection` only:

- `orders`
- `active_order_ids`
- `action_needed`
- `pending_reviews`
- `pending_review_count`
- `mode`

Important consequence:

- `active_order_ids` is no longer inferred from live cooks.
- Review-blocked or otherwise non-running but still active orders remain in `active_order_ids` if projection says they are active.
- Runtime enrichment is limited to session-level fields like active/recent sessions, current action, cost, and context usage.

If a snapshot test or fixture disagrees with the live-cook subset, the projection view is now the source of truth.
