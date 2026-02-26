Back to [[plans/49-work-orders-redesign/overview]]

# Codex Review Findings — Plan 49

Three independent codex reviews completed 2026-02-26. Findings deduplicated and triaged.

## Critical

1. **Stranded active stage on dispatch failure.** `spawnCook` persists `Stage.Status="active"` before `Dispatch`, but does not revert on dispatch error. After restart, the stage stays `active` with no session — permanently stuck.
   - `loop/cook.go:105-138`
   - Fix: revert stage status to `pending` in the dispatch-error path (lines 134-138).

## High

2. **Stage status not validated during normalization.** `NormalizeAndValidateOrders` validates order status but never calls `ValidateStageStatus` on individual stages. Invalid stage statuses can persist and wedge orders.
   - `internal/orderx/orders.go:153` (missing call to `ValidateStageStatus`)
   - Fix: add `ValidateStageStatus` call in the stage validation loop.

3. **Quality gate fail-open on read/parse errors.** `readQualityVerdict` returns `(zero, false)` on any error — file corruption or parse failure treated as "no verdict," silently bypassing rejection.
   - `loop/cook.go:274-284`
   - Tradeoff: fail-open is arguably correct for missing files (no quality skill ran). Parse errors should probably log a warning or fail-closed.

4. **`controlMerge` swallows `readOrders` errors.** If reading orders.json fails, controlMerge proceeds with stale state — could merge code for already-removed orders.
   - `loop/control.go:279-292`
   - Fix: return error on readOrders failure before attempting merge.

5. **Pending-review resolution not atomic.** State advances in orders.json first, then pending-review.json is updated. Crash between the two leaves a stale pending entry that can be resolved again.
   - `loop/control.go:292-296`, `loop/pending_review.go:126,160`
   - Mitigation: reconcile pending reviews against orders on cycle start, or accept as known limitation.

6. **Snapshot integration test doesn't exercise actual snapshot loading.** `TestIntegrationSnapshotIncludesOrdersAndPendingReviews` reads loop files directly — doesn't call `snapshot.LoadSnapshot` or test API serialization.
   - `loop/integration_test.go:597-686`
   - Fix: either rename to clarify scope or add true snapshot round-trip assertion.

## Medium

7. **`completed`/`failed` order statuses accepted by validation but never dispatched or removed.** The loop only dispatches `active`/`failing` orders — a `completed` order from external input would sit forever.
   - `internal/orderx/queue.go:57-65`, `loop/orders.go:212`
   - Fix: reject `completed`/`failed` during normalization (these are transient states never persisted in normal flow).

8. **ID TrimSpace applied for checks but not written back.** Mixed whitespace variants can bypass dedupe in promotion.
   - `internal/orderx/orders.go:139`
   - Fix: assign trimmed ID back to the order.

9. **Invalid orders-next.json deleted on parse error.** Transient malformation (partial write) permanently loses queued work.
   - `loop/orders.go:315-318`
   - Tradeoff: deleting prevents infinite retry loops. Could rename to `.bad` instead.

10. **`controlRequeue` mutates in-memory failedTargets before durable write.** If order persistence fails, memory and disk diverge.
    - `loop/control.go:505-529`
    - Fix: write orders first, then delete from failedTargets.

11. **`ParseOrdersStrict` not actually strict.** Uses `json.Unmarshal` without `DisallowUnknownFields`. Unknown fields silently dropped on rewrite.
    - `internal/orderx/orders.go:31-37`
    - Fix: use `json.Decoder` with `DisallowUnknownFields()`.

## Low

12. **Single-writer assumed but not enforced.** No file lock around read-modify-write. Multiple loop processes = lost updates.
    - `loop/orders.go:323-344`
    - Known constraint — documented in plan. Not fixing now.

13. **Missing OnFailure Stage.Extra round-trip test coverage.**
    - `internal/orderx/orders_io_test.go`, `internal/snapshot/snapshot_test.go`
    - Fix: add assertion for `OnFailure[].Extra` in existing round-trip tests.

14. **Snapshot package depends on `loop.PendingReviewItem` directly.** Leaky abstraction.
    - `internal/snapshot/types.go:51`
    - Deferred — architectural cleanup, not a correctness bug.
