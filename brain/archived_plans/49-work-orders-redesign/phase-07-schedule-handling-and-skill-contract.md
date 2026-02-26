Back to [[archived_plans/49-work-orders-redesign/overview]]

# Phase 7: Schedule handling and skill contract

Covers: #49 (schedule migration), #61 (mise.json simplification)

## Goal

Migrate schedule handling to use orders, simplify mise.json by removing pre-filtering, and update the schedule skill contract so it receives unfiltered plans and outputs orders instead of queue items.

## Changes

### Mise.json simplification (#61)

**`mise/builder.go`** — Delete `schedulablePlanIDs()` function (~lines 162-193). Remove the call site that populates `NeedsScheduling`. All plans flow through to mise.json unconditionally.

**`mise/types.go`** — Remove `NeedsScheduling []int` from the `Brief` struct.

**`mise/builder_test.go`** — Delete `TestSchedulablePlanIDsFallsBackToPlanID`. Add test: all plans appear in `brief.Plans` regardless of backlog status.

**`loop/loop.go`** (~lines 337, 343, 354, 394) — Remove all references to `brief.NeedsScheduling`. The idle-vs-work check should use `len(brief.Plans) == 0` or be removed — the schedule skill handles empty plans.

**`loop/orders.go`** (was `loop/queue.go` callers) — Remove `NeedsScheduling` parameter pass-through to `queuex.NormalizeAndValidate` (already migrated to orders validation in phase 5, but verify no remnants).

**`internal/queuex/queue.go`** — Remove `schedulablePlanIDs []int` parameter from `NormalizeAndValidate` if it still exists. Remove any filtering that uses it.

**`internal/schemadoc/specs.go`** (~line 61) — Remove `needs_scheduling` from queue/orders schema documentation.

### Schedule migration to orders

**`loop/schedule.go`** — Migrate schedule functions:
- `bootstrapScheduleOrder()` replaces `bootstrapScheduleQueue()` — creates an OrdersFile with a single order containing one stage `{task_key: "schedule"}`.
- `scheduleOrder(cfg, prompt)` replaces `scheduleQueueItem()` — creates the schedule order.
- `isScheduleOrder(order)` — checks if order is the schedule singleton (ID == "schedule").
- `hasNonScheduleOrders(orders)` — checks for non-schedule orders. This is the simplified check from #60 — replaces the nested `filterStaleScheduleItems`/`hasNonScheduleItems` conditionals.
- `spawnSchedule()` — takes stage from schedule order instead of QueueItem. Prompt construction includes the new orders schema (not queue item schema).
- `rescheduleForChefPrompt()` — creates schedule order with chef guidance in rationale.
- `buildOrderTaskTypesPrompt()` — update to describe the orders output format and the unfiltered plans input.

**`loop/loop.go`** — Update `prepareOrdersForCycle()` to use new schedule functions.

### Schedule skill contract

**`.agents/skills/schedule/SKILL.md`** — Update both input and output contracts. Use the `skill-creator` skill. Key changes:
- **Input:** Remove references to `needs_scheduling` (~lines 22, 26, 38). The skill reads `plans` array directly, cross-references with `backlog` to determine which plans have open work, and decides what to schedule.
- **Output:** File is `.noodle/orders-next.json` (replaces `queue-next.json`). Schema changes from `{items: [...]}` to `{orders: [...]}` where each order has `stages: [...]`.
- Document that execute → quality → reflect should be stages within ONE order, not separate items.
- Document that any task type can be a stage. The scheduler should prepend a debate stage when a plan item has an unresolved design question.
- Document `on_failure` stages: scheduler can specify a secondary pipeline that runs when any main stage fails.
- Provide example output showing: multi-stage order, order with on_failure, order with debate, single-stage order.

## Data structures

- Schedule order: `Order{ID: "schedule", Stages: [{TaskKey: "schedule", Status: "pending"}]}`
- Standard order: `Order{ID: "29", Stages: [{TaskKey: "execute"}, {TaskKey: "quality"}, {TaskKey: "reflect"}]}`
- Order with debate: `Order{ID: "29", Stages: [{TaskKey: "debate"}, {TaskKey: "execute"}, {TaskKey: "quality"}, {TaskKey: "reflect"}]}`
- Order with failure routing: `Order{ID: "29", Stages: [...], OnFailure: [{TaskKey: "debugging"}]}`
- `Brief` loses `NeedsScheduling []int`

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

Skill contract changes require judgment. Use `skill-creator` skill when editing SKILL.md.

## Verification

### Static
- `go build ./...` and `go vet ./...` pass
- Schedule skill SKILL.md describes orders schema, not queue items
- Grep for `schedulablePlanIDs`, `NeedsScheduling`, and `needs_scheduling` — zero hits in Go code and schedule skill

### Runtime
- Unit test: bootstrapScheduleOrder creates valid single-stage order
- Unit test: hasNonScheduleOrders returns false for schedule-only, true when real orders exist
- Unit test: spawnSchedule builds prompt with orders schema including on_failure documentation
- Unit test: all plans appear in `brief.Plans` regardless of backlog status (mise.json simplification)
- Manual: build mise.json with a mix of plans (some with done backlog items, some without) → all appear in `plans` array
- Manual: run schedule skill against real mise.json, verify orders-next.json output matches new schema
- Automated contract test: generate a schedule skill output (orders-next.json), run it through `NormalizeAndValidateOrders` + `ApplyOrderRoutingDefaults` + `dispatchableStages` pipeline → verify valid candidates are produced (catches schema drift between skill output and Go consumer)
