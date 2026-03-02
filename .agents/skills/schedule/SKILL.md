---
name: schedule
description: Orders scheduler. Reads .noodle/mise.json, writes .noodle/orders-next.json. Schedules work orders based on backlog state, plan phases, session history, and task type schedules.
schedule: "When orders are empty, after backlog changes, or when session history suggests re-evaluation"
---

# Schedule

Read `.noodle/mise.json`, write `.noodle/orders-next.json`.
The loop atomically promotes `orders-next.json` into `orders.json` — never write `orders.json` directly.
Use `noodle schema mise` and `noodle schema orders` as the schema source of truth.

Operate fully autonomously. Never ask the user to choose or pause for confirmation.

## Orders Model

Output is `{orders: [...]}` where each order is a **pipeline of stages** executed sequentially. Group related work into stages within one order rather than separate orders.

### Stages

Each stage has a `task_key` (must match a registered task type) and runs one at a time within the order. The loop advances to the next stage when the current one completes.

A typical order pipeline: execute, then quality, then reflect — all as stages of one order.

## Task Types

Read `task_types` from mise to discover every schedulable task type and its `schedule` hint. Any registered task type can be a stage within an order. Use `task_key` on each stage to bind it to a task type.

### Execute Tasks

Schedule execute tasks from the `backlog` array in mise. Use the backlog item ID (as a string) as the order `id`.

Backlog items always have `id` and `title`. Other fields are adapter-defined and may vary. The default adapter (todos.md) provides: `status`, `section`, `tags`, `estimate`, and `plan`. Custom adapters may include any fields — treat unknown fields as useful context.

**One plan at a time:** Schedule all remaining phases of one plan at once. Pick the highest-priority plan with remaining phases and schedule an order for each unfinished phase. Don't spread work across plans — completing one plan end-to-end produces usable results faster than advancing many plans one phase each. If the current plan is blocked, idle (empty orders) rather than context-switching to a different plan.

**Shared infrastructure:** When multiple plans depend on common infrastructure (shared types, utilities, base packages), propose a standalone infra order that isn't tied to any single plan. Use a descriptive slug ID (e.g., `"infra-shared-types"`, `"infra-event-system"`) — orders don't need to match a backlog item ID. If the infra work is substantial, create a backlog item for it via the adapter (`noodle adapter run backlog add`), then use that item's ID as the order ID. The infra order should run before any plan that depends on it.

**Items with plans:** When a backlog item has a `plan` field (a relative path like `brain/plans/29-foo/overview.md`), read the plan overview and phase files to understand the work. Schedule an order for each remaining unfinished phase (each unchecked `- [ ]` item), in order. Populate `order.plan` with the plan path(s). Use `extra_prompt` to inject plan context: the plan overview summary, the specific phase brief, and any cross-phase dependencies. The loop executes orders sequentially, so later phases naturally wait for earlier ones to complete.

**Items without plans:** Assess complexity before scheduling. If the item is straightforward (single concern, clear scope, small change), schedule as a simple execute task using the backlog item's title and description as the prompt. If the item is complex (multi-file, cross-cutting, ambiguous scope, or you'd want to see an architecture sketch before coding), schedule a general order with `task_key: "execute"` and use `extra_prompt` to instruct the session to invoke `/plan` first — e.g., `"This item needs a plan before implementation. Use /plan to break it down, then execute the first phase."` The plan skill will write phased plans to `brain/plans/`; on the next scheduling cycle, the item will have a `plan` field and can be scheduled normally.

**Standalone orders:** Orders can have arbitrary IDs — they don't need to correspond to a backlog item. When a standalone order completes, the `backlog done` adapter call is a no-op (no matching item to mark done). Use standalone orders for shared infrastructure, maintenance tasks, or cross-cutting work that serves multiple backlog items.

**Nothing to schedule:** When no backlog items are actionable (all blocked, all in-progress, all done, etc.), still write `orders-next.json` with an empty orders array (`{"orders":[]}`). This signals to the loop that scheduling ran but found nothing — preventing hot-loop re-spawns.

### Follow-Up and Standalone Stages

Each task type's `schedule` field describes when and how to schedule it — as a follow-up stage within an order, as a standalone order, or both. Read these hints from `task_types` in mise and compose orders accordingly.

## Recent Events

The mise brief includes a `recent_events` array — lifecycle events emitted by the loop since the last schedule run. These are context for your scheduling decisions, not commands.

### Internal Events

These are emitted automatically by the loop. The V2 backend uses canonical event types:

| Event type | Meaning |
|------------|---------|
| `stage_completed` | A stage finished successfully (includes order ID, stage index) |
| `stage_failed` | A stage failed (includes reason) |
| `order_completed` | All stages in an order finished — the order is done |
| `order_failed` | An order failed terminally |
| `merge_failed` | A merge failed (includes error reason) |
| `order.dropped` | An order was removed because its task type is no longer registered |
| `order.requeued` | A failed order was reset and re-queued for another attempt |
| `registry.rebuilt` | The skill registry was rebuilt (skills added or removed) |

### External Events

Users can inject custom events via `noodle event emit <type> [payload]`. These have arbitrary types like `ci.failed`, `deploy.completed`, `test.flaky`, etc. You won't know every possible type — interpret them from context and the summary string.

### Using Events for Scheduling

Events are context, not commands. Consider them alongside backlog state and session history when deciding what to schedule:

- After `stage.failed` or `order.failed` — consider whether the failure needs a debugging order, or if the item should be retried with a different approach.
- After `order.completed` — consider follow-up work (reflect, related items that were blocked).
- After `merge.conflict` — the affected order may need manual attention; avoid re-scheduling it immediately.
- After external events like `ci.failed` — consider scheduling an investigation or fix order if it seems actionable.
- After `registry.rebuilt` — new task types may be available; check `task_types` for scheduling opportunities.

Don't react mechanically to every event. Use judgment: a single stage failure in a long pipeline is normal; three consecutive failures of the same order suggests a deeper problem.

## Situational Awareness

| Trigger | Action |
|---------|--------|
| Empty orders | Full survey of mise — schedule from scratch |
| Quality rejection | Rescope the rejected item for retry with feedback |
| New backlog items | Create orders respecting workflow stage order |
| Items without plans | Assess complexity — simple items execute directly, complex items get a plan-first order |
| All items blocked/done | Write empty orders array, let loop cooldown |

## Scheduling Heuristics

- **One plan at a time**: Schedule all remaining phases of one plan upfront. Don't spread work across plans — depth-first produces complete, shippable results. The loop executes orders sequentially, so all phases run in order. Exception: shared infra orders can run alongside or before a plan's phases.
- **Foundation before feature**: Infrastructure and shared types first.
- **Cheapest mode**: Prefer the lowest-cost provider/model that can handle the task.
- **Explicit rationale**: Every order must cite which principle or rule drove its placement.
- **Timebox failures**: If an item has failed 2+ times in `recent_history`, deschedule or split it.

## Stage Lifecycle

Write stages with `"status": "pending"`. The loop manages all subsequent transitions (dispatching, running, merging, review, completed/failed).

## Model Routing

| Task type | Provider | Model |
|-----------|----------|-------|
| Tiny/small tasks (no deep thinking needed) | codex | gpt-5.3-codex-spark |
| Tiny/small tasks (no deep thinking needed) | claude | claude-sonnet-4-6 |
| Implementation, execution, coding | codex | gpt-5.3-codex |
| Judgment, strategy, planning, review | claude | claude-opus-4-6 |

Use spark or sonnet for small, mechanical tasks (simple renames, one-liner fixes, straightforward additions). Use full codex for anything requiring multi-step reasoning or cross-file coordination. When uncertain, codex for implementation, opus for judgment.

## Runtime Routing

Always set `"runtime": "process"` on all stages. The `sprites` runtime is still WIP and should not be used yet. Always include `"runtime"` on scheduled stages so dispatch routing is explicit.

## Output

Write valid JSON to `.noodle/orders-next.json` matching `noodle schema orders`.

### extra_prompt

Each stage supports an optional `extra_prompt` string — supplemental instructions about *how* to approach the task. Distinct from `prompt` (what to do) and `rationale` (why it's scheduled).

Use cases:
- Relay failure context from `recent_history` (e.g., "previous attempt failed because tests weren't run — run tests this time")
- Flag dependencies or preconditions the cook should be aware of
- Suggest approach constraints based on scheduling context

Keep it concise (~1000 chars max; silently truncated if exceeded). Leave empty when there's nothing extra to say — don't fill it for the sake of filling it. The field lives on each stage, not at the order level.

### Example: Multi-stage order

```json
{
  "orders": [
    {
      "id": "49",
      "title": "implement work orders redesign",
      "plan": ["plans/49-work-orders-redesign/overview"],
      "rationale": "foundation-before-feature: core infra needed by all other work",
      "status": "active",
      "stages": [
        {"task_key": "execute", "skill": "execute", "provider": "codex", "model": "gpt-5.3-codex", "runtime": "process", "status": "pending"},
        {"task_key": "quality", "skill": "quality", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"},
        {"task_key": "reflect", "skill": "reflect", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
      ]
    }
  ]
}
```

### Example: Order with debate stage

```json
{
  "orders": [
    {
      "id": "52",
      "title": "design cache invalidation strategy",
      "rationale": "unresolved design question needs structured debate before implementation",
      "status": "active",
      "stages": [
        {"task_key": "debate", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"},
        {"task_key": "execute", "provider": "codex", "model": "gpt-5.3-codex", "runtime": "process", "status": "pending"},
        {"task_key": "quality", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
      ]
    }
  ]
}
```

### Example: Shared infrastructure order

```json
{
  "orders": [
    {
      "id": "infra-event-system",
      "title": "shared event types used by subagent-tracking and diffs-integration",
      "rationale": "foundation-before-feature: both plan 84 and plan 86 depend on shared event types",
      "status": "active",
      "stages": [
        {"task_key": "execute", "skill": "execute", "provider": "codex", "model": "gpt-5.3-codex", "runtime": "process", "status": "pending",
         "extra_prompt": "Create shared event types in internal/event/ that both subagent-tracking and diffs-integration will consume. Keep scope narrow — only the shared interfaces, not plan-specific logic."},
        {"task_key": "quality", "skill": "quality", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
      ]
    }
  ]
}
```

### Example: Single-stage order

```json
{
  "orders": [
    {
      "id": "meditate-1",
      "title": "audit brain vault after recent reflects",
      "rationale": "3 reflects accumulated, time to consolidate",
      "status": "active",
      "stages": [
        {"task_key": "meditate", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
      ]
    }
  ]
}
```

## Principles

- [[cost-aware-delegation]]
- [[foundational-thinking]]
- [[subtract-before-you-add]]
- [[never-block-on-the-human]]
- [[guard-the-context-window]]
