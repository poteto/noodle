---
name: schedule
description: Orders scheduler. Reads .noodle/mise.json, writes .noodle/orders-next.json. Schedules work orders based on backlog state, plan phases, session history, and task type schedules.
noodle:
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

**Items with plans:** When a backlog item has a `plan` field (a relative path like `brain/plans/29-foo/overview.md`), read the plan overview and phase files to understand the work. Determine the next unfinished phase (first unchecked `- [ ]` item). Schedule an execute stage for that phase. Populate `order.plan` with the plan path(s). Use `extra_prompt` to inject plan context: the plan overview summary, the specific phase brief, and any cross-phase dependencies.

**Items without plans:** Schedule as a simple execute task using the backlog item's title and description as the prompt.

**Nothing to schedule:** When no backlog items are actionable (all blocked, all in-progress, all done, etc.), still write `orders-next.json` with an empty orders array (`{"orders":[]}`). This signals to the loop that scheduling ran but found nothing — preventing hot-loop re-spawns.

### Follow-Up Stages

After an execute stage, consider what naturally follows. Add follow-up stages to the same order:

- **quality** after **execute** — review the cook's work. **Cross-provider review:** prefer the opposite provider from the execute stage. If codex executed, claude reviews; if claude executed, codex reviews. Fresh eyes from a different model catch more issues.
- **reflect** after **quality** — capture learnings from the completed cycle

These are starting points, not a closed list. If the `task_types` registry contains new types you haven't seen before, read their `schedule` hints and infer where they fit.

### Standalone Orders

Some task types run as standalone single-stage orders:

- **meditate** after several reflects have accumulated — audit the brain vault (expensive, don't over-schedule)
- **debate** when a plan item has an unresolved design question — prepend as a stage before execute

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
| Items without plans | Schedule as simple execute tasks |
| All items blocked/done | Write empty orders array, let loop cooldown |
| All items blocked | Schedule reflect or meditate to use the slot productively |

## Scheduling Heuristics

- **Foundation before feature**: Infrastructure and shared types first.
- **Cheapest mode**: Prefer the lowest-cost provider/model that can handle the task.
- **Explicit rationale**: Every order must cite which principle or rule drove its placement.
- **Work around blockers**: If the top-priority item is blocked, schedule the next viable item — never idle.
- **Timebox failures**: If an item has failed 2+ times in `recent_history`, deschedule or split it.

## Stage Lifecycle

Write stages with `"status": "pending"`. The loop manages all subsequent transitions (dispatching, running, merging, review, completed/failed).

## Model Routing

| Task type | Provider | Model |
|-----------|----------|-------|
| Implementation, execution, coding | codex | gpt-5.3-codex |
| Judgment, strategy, planning, review | claude | claude-opus-4-6 |

When uncertain, codex for implementation, opus for judgment.

## Runtime Routing

Read `routing.available_runtimes` from mise before writing orders.

- If only `process` is available, set stage `"runtime": "process"`.
- If `sprites` is available, prefer `"runtime": "sprites"` for long-running `execute` work.
- Keep `review`, `reflect`, and `meditate` on `"runtime": "process"` unless explicitly justified.
- `cursor` runtime requires `polling` and `remote_sync` capabilities — only use when available and appropriate.
- Always include `"runtime"` on scheduled stages so dispatch routing is explicit.

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
        {"task_key": "execute", "skill": "execute", "provider": "codex", "model": "gpt-5.3-codex", "runtime": "sprites", "status": "pending"},
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
        {"task_key": "execute", "provider": "codex", "model": "gpt-5.3-codex", "runtime": "sprites", "status": "pending"},
        {"task_key": "quality", "provider": "claude", "model": "claude-opus-4-6", "runtime": "process", "status": "pending"}
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
