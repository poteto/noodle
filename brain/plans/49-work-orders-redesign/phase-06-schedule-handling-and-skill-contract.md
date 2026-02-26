Back to [[plans/49-work-orders-redesign/overview]]

# Phase 6: Schedule handling and skill contract

## Goal

Migrate schedule handling to use orders and update the schedule skill contract so it outputs orders instead of queue items.

## Changes

**`loop/schedule.go`** — Migrate schedule functions:
- `bootstrapScheduleOrder()` replaces `bootstrapScheduleQueue()` — creates an OrdersFile with a single order containing one stage `{task_key: "schedule"}`.
- `scheduleOrder(cfg, prompt)` replaces `scheduleQueueItem()` — creates the schedule order.
- `isScheduleOrder(order)` — checks if order is the schedule singleton (ID == "schedule").
- `hasNonScheduleOrders(orders)` — checks for non-schedule orders. This is the simplified check from #60 — replaces the nested `filterStaleScheduleItems`/`hasNonScheduleItems` conditionals.
- `spawnSchedule()` — takes stage from schedule order instead of QueueItem. Prompt construction includes the new orders schema (not queue item schema).
- `rescheduleForChefPrompt()` — creates schedule order with chef guidance in rationale.
- `buildQueueTaskTypesPrompt()` — update to describe the orders output format.

**`.agents/skills/schedule/SKILL.md`** — Update output contract. Use the `skill-creator` skill for this. Key changes:
- Output file is `.noodle/orders-next.json` (replaces `queue-next.json`).
- Schema changes from `{items: [...]}` to `{orders: [...]}` where each order has `stages: [...]`.
- Document that execute → quality → reflect should be stages within ONE order, not separate items.
- Document that any task type can be a stage. The scheduler should prepend a debate stage when a plan item has an unresolved design question (e.g. `[debate → execute → quality → reflect]`). If the debate concludes the work shouldn't proceed, the cook can signal failure and the loop cancels remaining stages.
- Document `on_failure` stages: scheduler can specify a secondary pipeline that runs when any main stage fails. Example: `{stages: [execute, quality, reflect], on_failure: [debugging]}`. If quality rejects, debugging runs instead of just cancelling reflect.
- Provide example output showing: multi-stage order, order with on_failure, order with debate, single-stage order.

**`loop/loop.go`** — Update `prepareOrdersForCycle()` to use new schedule functions.

## Data structures

- Schedule order: `Order{ID: "schedule", Stages: [{TaskKey: "schedule", Status: "pending"}]}`
- Standard order: `Order{ID: "29", Stages: [{TaskKey: "execute"}, {TaskKey: "quality"}, {TaskKey: "reflect"}]}`
- Order with debate: `Order{ID: "29", Stages: [{TaskKey: "debate"}, {TaskKey: "execute"}, {TaskKey: "quality"}, {TaskKey: "reflect"}]}`
- Order with failure routing: `Order{ID: "29", Stages: [...], OnFailure: [{TaskKey: "debugging"}]}`

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

Skill contract changes require judgment. Use `skill-creator` skill when editing SKILL.md.

## Verification

### Static
- `go build ./...` and `go vet ./...` pass
- Schedule skill SKILL.md describes orders schema, not queue items

### Runtime
- Unit test: bootstrapScheduleOrder creates valid single-stage order
- Unit test: hasNonScheduleOrders returns false for schedule-only, true when real orders exist
- Unit test: spawnSchedule builds prompt with orders schema including on_failure documentation
- Manual: run schedule skill against real mise.json, verify orders-next.json output matches new schema (including on_failure for appropriate items)
