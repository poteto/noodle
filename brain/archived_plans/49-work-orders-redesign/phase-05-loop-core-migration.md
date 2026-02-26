Back to [[archived_plans/49-work-orders-redesign/overview]]

# Phase 5: Loop core migration

Covers: #49 (core), #60, #63, #64 (dispatch wiring), #65

## Goal

Swap the loop from reading `queue.json` (flat items) to reading `orders.json` (orders with stages). Wire domain skill dispatch through the registry. Rewrite merge conflict handling to park for review. This is the big-bang phase — dispatch, completion, retry, pending review, queue audit, reconciliation, and compilation stubs for control/schedule all switch together. Everything that touches `QueueItem` in the hot path must move at once because they share the `activeCook` struct.

## Changes

**`loop/types.go`** — Update `activeCook`, `pendingReviewCook`, `pendingRetryCook`:
- Replace `queueItem QueueItem` with `orderID string` + `stageIndex int` + `stage Stage`
- Add order-level fields needed at completion time (`plan []string` for adapter callbacks)

**`loop/loop.go`** — `prepareQueueForCycle()` → `prepareOrdersForCycle()`:
- Call `consumeOrdersNext()` instead of `consumeQueueNext()`
- Call `readOrders()` instead of `readQueue()`
- Normalize and validate orders instead of queue items
- **Simplify filtering (#60):** Don't port `filterStaleScheduleItems`/`hasNonScheduleItems` nested conditionals. Simplify to: if no non-schedule orders exist and work is available (`len(brief.Plans) > 0`), bootstrap a schedule order. If no work exists, go idle. Keep validation (reject malformed orders) but stop making scheduling judgments — the schedule skill owns queue composition. (Note: `brief.NeedsScheduling` is removed in phase 7 — use `len(brief.Plans)` from the start.)

**`loop/loop.go`** — `planCycleSpawns()`:
- Call `dispatchableStages()` (from phase 4) instead of iterating queue items
- Build spawn plan from `dispatchCandidate` list

**`loop/cook.go`** — `spawnCook()`:
- Takes `dispatchCandidate` (or order + stage + index) instead of `QueueItem`
- `buildCookPrompt()` reads from stage fields + order-level plan/rationale
- `cookBaseName()` derives name from `orderID:stageIndex:taskKey` (e.g. `29:0:execute`, `29:1:quality`). Uses `:` separator — unambiguous because orderID (numeric), stageIndex (numeric), and taskKey (alphanumeric) cannot contain `:`. Hyphen was rejected because taskKey could theoretically contain hyphens.
- `activeCook` stores orderID, stageIndex, stage
- **Persist active status:** Before spawning the session, set `Stage.Status = "active"` on the dispatched stage and write `orders.json` via `writeOrdersAtomic()`. This is critical for restart safety — without persisted status, a restart would re-dispatch already-running stages (the in-memory `activeByTarget` map doesn't survive restarts). The write must happen BEFORE the session spawn to prevent a window where a crash after spawn but before persist leaves the stage as `"pending"`.
- **Domain skill dispatch (#64):** Replace `if taskType.Key == "execute" { ... adapter.Skill }` with `if taskType.DomainSkill != "" { req.DomainSkill = taskType.DomainSkill }`. The `DomainSkill` field was added to the registry in phase 1. This is the dispatch wiring — the registry entry already has the value, spawnCook reads it.

**`dispatcher/tmux_dispatcher.go`** (~line 130) and **`dispatcher/sprites_dispatcher.go`** (~line 102):
- Both check `req.TaskKey == "execute"` to decide whether to load domain skill. Change to `req.DomainSkill != ""` so domain skill injection works for any task type that declares one.

**`loop/cook.go`** — `handleCompletion()`:
- On success: check quality verdict before merging (see below), then call `advanceOrder()` (from phase 4), persist with `writeOrdersAtomic()`. `advanceOrder` returns `removed bool` — if true and order was `"active"`, fire adapter "done"; if true and order was `"failing"`, call `markFailed` instead.
- If `removed` is false, more stages remain — they'll be dispatched next cycle
- On failure: call `failStage()` if retries exhausted (which triggers OnFailure stages if present), or retry the same stage
- Merge path: check `canMerge` from the task type registry (same as current code). Mergeable stages merge their worktree, then advance. Non-mergeable stages (debate, schedule) skip merge and just advance. Only call `Adapter.Run("backlog", "done", orderID)` when the final stage of a **non-failing** order completes — not per-stage. For `"failing"` orders, when the last OnFailure stage completes, `advanceOrder` removes the order and the caller calls `markFailed` (the OnFailure pipeline is remediation, not recovery — the original failure stands). Do NOT fire adapter "done" for failing orders.
- Schedule special case: schedule stages have no worktree (run in project dir). handleCompletion must detect schedule and skip merge/worktree cleanup, same as current `isScheduleItem` check.
- **Quality verdict check (#65):** After a stage completes successfully but before merging, read `.noodle/quality/<session-id>.json`. If a verdict file exists and `accept == false`, treat the stage as failed (call `failStage()` instead of advancing). This makes quality verdicts enforceable. Add a `QualityVerdict` struct to `loop/types.go`: `{Accept bool, Feedback string}` (only read the fields the loop needs). Verdict reading is at the boundary — validate at read time, trust internally.
- **Merge conflict handling (#63):** Rewrite `handleMergeConflict()`: instead of calling `markFailed()` + `skipQueueItem()`, call `parkPendingReview()` with `Reason: "merge conflict: <details>"`. Keep the schedule item exemption (schedule merge conflicts still propagate as errors). The human resolves via the web UI, then controlMerge or controlRequestChanges handles the outcome (phase 6). If the order has OnFailure stages and the user rejects, failStage routes to OnFailure naturally.
- **Pending approval interaction:** If `config.PendingApproval()` is true, park for review as before (human sees the verdict in the review UI). Verdict check only applies in `auto` autonomy mode where the loop would otherwise merge without human review.

**`loop/cook.go`** — `retryCook()`:
- Retry dispatches the same stage (same orderID, same stageIndex) with incremented attempt. If `IsOnFailure` is true, retry within the OnFailure pipeline.

**`loop/cook.go`** — `collectCompleted()`:
- Maps session ID → activeCook unchanged (activeCook struct just has different fields)

**`loop/pending_review.go`** — `PendingReviewItem`:
- Replace QueueItem-mirror fields with orderID + stageIndex + stage fields
- Add `Reason string` field (`json:"reason,omitempty"`) — surfaces why the item is parked (e.g., "merge conflict", "quality rejected", "approval required"). Empty means normal completion review.
- `parkPendingReview()` copies from activeCook's new shape, accepts optional reason
- `loadPendingReview()` deserializes the new shape. If parsing fails (old-format file from pre-upgrade): attempt to extract the worktree path from the raw JSON (it's a top-level field in both old and new formats). If found, log a warning with the worktree path so the human can resolve manually ("pending review file has old format — worktree at <path> needs manual merge or cleanup"). If worktree path cannot be extracted, log an error. Do not silently discard — the worktree would leak with no way for the user to discover it.

**`loop/control.go`** (~line 309-313) — Replace hardcoded `taskType.Key == "execute"` in `controlRequestChanges()` with `taskType.DomainSkill != ""` (same domain_skill pattern as spawnCook above). **Note:** This change is transitional — `controlRequestChanges()` is fully rewritten in phase 6. The phase 5 patch ensures compilation; the phase 6 rewrite supersedes it. Do not skip the phase 5 patch, as the code must compile between phases.

**`internal/queuex/queue.go`** — Replace `taskType.Key == "execute"` in the `ApplyRoutingDefaults` function (the function that fills domain skill on queue items at validation time) with `taskType.DomainSkill != ""`. Locate by function name, not line number — line numbers shift across phases.

**`loop/util.go`**:
- `buildCookPrompt()` takes stage + order-level context instead of QueueItem
- `cookBaseName()` takes orderID + stageIndex + stage.TaskKey
- Delete `findQueueItemByTarget()` — replaced by order lookup

**`loop/queue_audit.go`** → rename to **`loop/order_audit.go`**, function `auditOrders()`:
- `auditQueue()` is called from `rebuildRegistry()` during every loop cycle — cannot be deferred. Migrate now.
- Iterate orders, for each order iterate stages, validate stage task types against registry
- Drop orders where no stages resolve. Log `order_drop` events to `queue-events.ndjson`.

**`loop/reconcile.go`** — Update adopted-session recovery:
- Current code parses cook prompts to re-associate sessions after restart (e.g. `work backlog item <id>`)
- `buildCookPrompt()` format changes in this phase — the parser must match the new format
- If prompt parsing fails, sessions become orphaned and may cause duplicate respawns
- **Schedule session regex** (`schedulePromptRegexp` at ~line 100): The current regex matches the old schedule prompt format. Update it to match the new `buildCookPrompt()` output for schedule stages. Document the new prompt format so the regex can be verified against it. Test the regex against both old and new format strings (old format may appear in adopted sessions from a pre-migration restart).
- The new prompt format should include the order ID and stage index in a parseable location (e.g., a structured header line) so reconcile can map session → order ID reliably.

**`internal/taskreg/registry.go`** — Rename registry APIs:
- `QueueItemInput` → `StageInput`
- `ResolveQueueItem()` → `ResolveStage()`
- Update all callers (queuex, loop, audit)

**`dispatcher/preamble.go`** — Update file references:
- Replace `.noodle/queue.json — Scheduled work queue` with `.noodle/orders.json — Work orders` (and `queue-next.json` → `orders-next.json`). This must happen in the same phase as the loop migration — agents dispatched after phase 5 must see the correct file paths.

**`internal/snapshot/snapshot.go`** — Update `InferTaskType()`:
- `InferTaskType()` (~line 686) infers task type from session ID prefix using known types (`"execute"`, `"schedule"`, etc.). After this phase, session IDs follow the pattern `orderID:stageIndex:taskKey` (e.g., `29:0:execute`). Update `InferTaskType` to split on `:` and extract taskKey from the third segment. Fallback to prefix matching for sessions created before the migration (no `:` in ID).

**`loop/builtin_bootstrap.go`** — Update file path references from `queue-next.json` to `orders-next.json`. This was deferred from phase 1 because the orders file format doesn't exist until phase 3 and the loop doesn't consume it until this phase.

**`loop/control.go`** — Minimal compilation stubs:
- Control commands must compile against the new types. Add stub implementations that read/write `orders.json` instead of `queue.json`. Full logic refinement happens in phase 6. **Critically: stub `controlReject()` must call `cancelOrder()` (not the old `skipQueueItem()`).** The old code calls `markFailed()` + `skipQueueItem()` which writes to `queue.json` — a dead file after this phase.
- **Stub `controlMerge()` must include quality verdict check** — even as a stub, the merge path must read `.noodle/quality/<session-id>.json` and reject if `accept == false`. This ensures the quality gate is never bypassable, even between phase 5 and phase 6.
- **Phase 5+6 atomicity:** Phase 5 introduces `parkPendingReview()` for merge conflicts, but the full resolution commands (`controlMerge`, `controlRequestChanges`) are stubs until phase 6. If phase 5 lands before phase 6, parked orders have no real resolution path. The stubs must be functional enough to resolve parked orders (merge worktree + advance, or reject + failStage) — they just don't need the full UX polish that phase 6 adds.

**`internal/snapshot/snapshot.go`** — **Mandatory `LoadSnapshot` patch:** Add a minimal read of `orders.json` (in addition to the existing `queue.json` read) so the web UI doesn't show an empty queue column between phases 5-7. Convert orders to the existing `snapshot.QueueItem` shape temporarily — phase 8 replaces this with proper order types. This prevents a broken UI window without requiring phases to land atomically.

**`loop/schedule.go`** — Minimal compilation stubs:
- Schedule functions must compile against the new types. Stub `bootstrapScheduleOrder()`, `isScheduleOrder()`, etc. Full logic refinement happens in phase 7.

**`loop/queue.go`** — Keep for now (deleted in phase 10). Stop calling old functions.

## Data structures

- `activeCook{orderID, stageIndex, stage, isOnFailure, orderStatus, plan, session, worktreeName, worktreePath, attempt, displayName}`
- `activeByTarget` keyed by `orderID` (one cook per order at a time)
- `QualityVerdict{Accept bool, Feedback string}` — minimal struct for reading verdict files at the merge boundary
- `PendingReviewItem` gains `Reason string` field

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

Complex migration with judgment calls about edge cases and state transitions.

## Verification

### Static
- `go build ./...` and `go vet ./...` pass — the entire `loop/` package must compile, including control.go and schedule.go (via stubs)
- No remaining references to `QueueItemInput` or `ResolveQueueItem` in taskreg
- Grep for `taskType.Key == "execute"` and `req.TaskKey == "execute"` — zero hits in domain-skill contexts

### Runtime
- Unit test: spawnCook persists `Stage.Status = "active"` to orders.json before spawning session
- Unit test: spawnCook with a dispatchCandidate creates activeCook with correct orderID/stageIndex
- Unit test: spawnCook with DomainSkill on registry entry → `req.DomainSkill` set correctly
- Unit test: spawnCook without DomainSkill on registry entry → `req.DomainSkill` empty
- Unit test: handleCompletion on success advances order stage, order persisted
- Unit test: handleCompletion on success of final stage removes order from OrdersFile and fires adapter "done"
- Unit test: handleCompletion on failure retries same stage with incremented attempt
- Unit test: handleCompletion on failure with exhausted retries calls failStage — when terminal=false, order stays (OnFailure will dispatch); when terminal=true, calls markFailed
- Unit test: handleCompletion on final OnFailure stage completion calls markFailed (not adapter "done")
- Unit test: handleCompletion with quality verdict `accept=false` treats stage as failed (calls failStage)
- Unit test: handleCompletion with quality verdict `accept=true` proceeds normally
- Unit test: handleCompletion with no verdict file proceeds normally (verdict is optional)
- Unit test: handleCompletion in `approve` autonomy mode parks for review regardless of verdict
- Unit test: handleCompletion for schedule stage skips merge/worktree cleanup
- Unit test: merge conflict on cook → parks for pending review with reason containing "merge conflict" (not markFailed)
- Unit test: merge conflict on cook → item NOT in failed targets
- Unit test: merge conflict on schedule item → error propagated (schedule exemption preserved)
- Unit test: PendingReviewItem with Reason round-trips through JSON (including empty/omitted case)
- Unit test: loadPendingReview discards old-format files gracefully (logs warning, no crash)
- Unit test: auditOrders drops orders with unresolvable stages, emits `order_drop` event
- Unit test: reconcile re-associates adopted sessions using new prompt format
- Unit test: reconcile schedule regex matches the new schedule prompt format
- Unit test: reconcile schedule regex handles old-format prompt from pre-migration sessions (graceful — either match or skip, no crash)
- Unit test: InferTaskType parses task key from `orderID:stageIndex:taskKey` session ID format (splits on `:`)
- Unit test: InferTaskType falls back to prefix matching for old-format session IDs (no `:` in ID)
- Unit test: prepareOrdersForCycle idle/bootstrap decision matrix — schedule-only orders, no non-schedule orders + plans available (bootstrap), no non-schedule orders + no plans (idle), mix of orders
- Unit test: domain-skill wiring through tmux_dispatcher — `req.DomainSkill` set → dispatcher receives and uses it
- Unit test: domain-skill wiring through sprites_dispatcher — same as above
- Unit test: LoadSnapshot minimal patch reads orders.json and converts to QueueItem shape (temporary bridge for phases 5-7)
- Integration test note: merge conflict detected → order parked as pending review → controlMerge stub resolves → order advances (cover full flow in phase 10 integration tests)
- Unit test: loadPendingReview on old-format file logs warning with worktree path (not silent discard)
- Unit test: controlReject stub calls cancelOrder on orders.json (not skipQueueItem on queue.json)
- Run `go test ./loop/...` — existing tests will break and must be updated in this phase to use the new types
