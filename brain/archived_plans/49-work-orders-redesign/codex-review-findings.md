Back to [[archived_plans/49-work-orders-redesign/overview]]

# Codex Review Findings — Plan 49

Three independent codex reviews completed 2026-02-26. Findings deduplicated and triaged.

## Critical

1. ~~**Stranded active stage on dispatch failure.**~~ ✓ Fixed — `spawnCook` now reverts stage status to `pending` on dispatch error. `loop/cook.go`.

## High

2. ~~**Stage status not validated during normalization.**~~ ✓ Fixed — `NormalizeAndValidateOrders` now calls `ValidateStageStatus` on all stages and OnFailure stages. `internal/orderx/orders.go`.

3. ~~**Quality gate fail-open on read/parse errors.**~~ ✓ Fixed — `readQualityVerdict` now logs a warning on parse errors. Missing file still returns false (correct — no verdict). `loop/cook.go`.

4. ~~**`controlMerge` swallows `readOrders` errors.**~~ ✓ Fixed — returns error on readOrders failure. `loop/control.go`.

5. ~~**Pending-review resolution not atomic.**~~ ✓ Mitigated — added `reconcilePendingReview` to cycle start that prunes pending reviews for orders no longer in orders.json. `loop/pending_review.go`, `loop/reconcile.go`.

6. ~~**Snapshot integration test doesn't exercise actual snapshot loading.**~~ ✓ Fixed — renamed to `TestIntegrationLoopFilesReadableForSnapshot` to clarify scope. `loop/integration_test.go`.

## Medium

7. ~~**`completed`/`failed` order statuses accepted but never dispatched.**~~ ✓ Fixed — `NormalizeAndValidateOrders` now rejects terminal statuses. `internal/orderx/orders.go`.

8. ~~**ID TrimSpace applied for checks but not written back.**~~ ✓ Fixed — trimmed ID assigned back to order. `internal/orderx/orders.go`.

9. ~~**Invalid orders-next.json deleted on parse error.**~~ ✓ Fixed — renamed to `.bad` instead. `loop/orders.go`.

10. ~~**`controlRequeue` mutates in-memory before durable write.**~~ ✓ Fixed — reordered to write orders first, then delete from failedTargets. `loop/control.go`.

11. ~~**`ParseOrdersStrict` not actually strict.**~~ ✓ Fixed — uses `json.Decoder` with `DisallowUnknownFields()`. `internal/orderx/orders.go`.

## Low

12. **Single-writer assumed but not enforced.** Known constraint — documented in plan. Not fixing.

13. ~~**Missing OnFailure Stage.Extra round-trip test coverage.**~~ ✓ Fixed — added Extra to OnFailure fixture and assertion. `internal/orderx/orders_io_test.go`.

14. **Snapshot package depends on `loop.PendingReviewItem` directly.** Deferred — architectural cleanup, not a correctness bug.
