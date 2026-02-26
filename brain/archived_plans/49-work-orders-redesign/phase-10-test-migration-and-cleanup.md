Back to [[archived_plans/49-work-orders-redesign/overview]]

# Phase 10: Test migration and cleanup

## Goal

Migrate all test fixtures from queue format to orders format, delete old types and dead code, verify the full suite passes.

## Progress

- [x] Test file `queue.json` string references updated to `orders.json` in the listed test/docstring files.
- [x] Optional package rename completed: `internal/queuex/` → `internal/orderx/` with imports/callers migrated.
- [x] Cross-phase integration tests.
- [x] New fixture scenarios.
- [x] Codex review (three independent reviews) → [[archived_plans/49-work-orders-redesign/codex-review-findings]]

## Changes

**`loop/testdata/*/state-*/.noodle/queue.json`** → rename to `orders.json`:
- Convert fixture queue files from `{items: [...]}` to `{orders: [{stages: [...]}]}` format
- Each old queue item becomes either a stage within an order or a single-stage order, depending on the fixture's intent
- Update `expected.md` assertions to match new runtime dump shape (spawn calls reference order IDs, not item IDs)

**`loop/fixture_test.go`** — Update harness:
- Read `orders.json` instead of `queue.json` from fixture state directories
- Update runtime dump capture to report order-based state (active order IDs, stage status)

**`loop/*_test.go`** — Migrate unit tests:
- Replace `QueueItem` construction with `Order`/`Stage` construction in all test helpers
- Update `testLoopRegistry()` if needed
- Update fake assertions (dispatch call args now come from stages)

**`internal/queuex/queue_test.go`** — Add orders-specific tests, delete queue-only tests that are now dead.

**Delete dead code (subtraction first):**
- `tui/` package — already deleted in phase 8 (moved there to prevent compilation failure when `snapshot.QueueItem` is deleted). Verify directory is gone.

**Delete old queue code:**
- `loop/types.go` — delete `QueueItem`, `Queue` types
- `loop/queue.go` — delete `readQueue`, `writeQueueAtomic`, `consumeQueueNext`, conversion functions
- `internal/queuex/queue.go` — delete `Item`, `Queue` types and their read/write/validate functions (keep only orders-related code)
- `internal/snapshot/types.go` — old `QueueItem` already deleted in phase 8
- `loop/util.go` — delete `findQueueItemByTarget()`
- `loop/queue_audit.go` — already renamed to `loop/order_audit.go` in phase 5. Verify old file is gone.

**Delete old bootstrap code (already replaced in phase 1):**
- Verify `bootstrapPromptTemplate` constant is gone (deleted in phase 1)
- Verify `buildBootstrapPrompt()` is gone (deleted in phase 1)
- Verify `shouldRecoverMissingSyncScripts()` is gone or simplified (phase 1)

**Migrate overlooked callers:**
- `cmd_status.go` — uses `queuex.Read()` for queue depth. Switch to `queuex.ReadOrders()` and count active orders. **Note:** This file compiles through phases 5-9 because the old `queuex` queue functions are kept until this phase. The dependency is intentional — do not delete old `queuex` functions before migrating these callers.
- `cmd_debug.go` — references queue files. Update to orders files. Same compilation dependency as above.
- `dispatcher/preamble.go` — verify already updated in phase 5 (moved there because agents need correct file paths immediately).
- `internal/schemadoc/specs.go` — uses `queuex.Queue{}` for schema docs. Update to `queuex.OrdersFile{}` with new field documentation.
- `loop/defaults.go` — references queue constants/paths. Update to orders paths.
- `generate/skill_noodle.go` — generates noodle skill text referencing queue concepts. Update to orders terminology.

**Rename package** (optional): `internal/queuex/` → `internal/orderx/` if the package now exclusively handles orders. This is a clean rename — update all imports.

**File cleanup:**
- Delete any `.noodle/queue.json` and `.noodle/queue-next.json` references in `.gitignore`, docs, scripts
- Verify `brain/plans/59-subtract-go-logic-and-resilience/` is already deleted (folded into this plan during planning)

## Data structures

- No new types — this phase only deletes and migrates

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

Mechanical migration and deletion. Clear spec from prior phases.

**Fixture parity audit:**
- Before converting fixtures, enumerate what each existing fixture tests (document in a comment or checklist). After conversion, verify each new fixture retains the original intent. This prevents silently dropping edge case coverage during the format migration.

**New fixture scenarios** (required — cover behaviors introduced in this plan):
- Multi-stage order: execute → quality → reflect (success path through all stages)
- OnFailure routing: execute succeeds, quality rejects (verdict `accept=false`), debugging stage dispatches
- Merge conflict: cook completes, merge conflicts, parks for review with reason
- Failed-target stickiness: failed order blocks re-dispatch until requeue
- Requeue recovery: failed order is requeued, stages reset to pending, dispatches again
- Domain skill dispatch: order with `domain_skill: backlog` task type → spawnCook sets `req.DomainSkill` → dispatcher receives it (end-to-end from orders.json through dispatch)

**Cross-phase integration tests** (required — unit tests alone miss state continuity bugs):
- Success pipeline end-to-end: `consumeOrdersNext` → `prepareOrdersForCycle` → `dispatchableStages` → `spawnCook` → `handleCompletion` → `advanceOrder` → persist → next cycle dispatches next stage. Crosses disk I/O boundary.
- OnFailure pipeline end-to-end: stage fails → `failStage` → order becomes `"failing"` → OnFailure stage dispatches → completes → `advanceOrder` removes order → `markFailed` called. Verify the intermediate `orders.json` state at each step (not just the end state).
- Merge-conflict resolution end-to-end: cook completes → merge conflict → `parkPendingReview` with reason → `controlMerge` resolves → order advances. Verify pending review state and final order state.
- Snapshot → Board derivation: `LoadSnapshot` reads orders.json → API serves snapshot → Board `deriveKanbanColumns` produces correct column assignments for queued, cooking, review, done. Verify with fixture orders including OnFailure and pending review with Reason.

**Stale event handling:**
- `queue-events.ndjson` may contain old event types (`bootstrap`, `queue_drop`). Keep the reader for `queue_drop` (aliased to `order_drop`). Remove `bootstrap` event type if no longer written post-phase-1. Do not truncate the events file — old events are historical.

## Verification

### Static
- `go build ./...` and `go vet ./...` pass
- `grep -r "QueueItem\|queue\.json\|queue-next\|QueueItemInput\|ResolveQueueItem" --include="*.go"` returns zero matches
- `grep -r "QueueItem\|queue\.json\|queue-next\|item_id" --include="*.ts" --include="*.tsx"` returns zero matches
- `grep -r "bootstrapPromptTemplate\|shouldRecoverMissingSyncScripts\|schedulablePlanIDs\|NeedsScheduling" --include="*.go"` returns zero matches
- `grep -r "queue\.json\|queue-next\|QueueItem\|buildQueueTaskTypes" docs/ scripts/ --include="*.md" --include="*.sh" --include="*.json"` returns zero matches (catches non-code references)
- `grep -r "queue_audit" --include="*.go"` returns zero matches (file renamed to `order_audit.go` in phase 5)
- Verify `tui/` directory does not exist (deleted in phase 8)
- `sh scripts/lint-arch.sh` passes

### Runtime
- `go test ./...` — full suite passes
- `NOODLE_LOOP_FIXTURE_MODE=check go test ./loop/...` — all fixtures pass with orders format
- Cross-phase integration tests pass (success pipeline, OnFailure pipeline)
- Manual: start noodle fresh, verify full cycle works (schedule creates orders, cook dispatches, stages advance, completion cleans up)
