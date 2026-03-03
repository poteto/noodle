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

## One Plan at a Time

This is the cardinal scheduling rule. Pick the highest-priority plan with remaining phases and schedule all of them. Do not spread work across multiple plans — finishing one plan end-to-end produces shippable results; advancing many plans one phase each produces nothing usable. If the current plan is blocked, idle (empty orders) rather than context-switching to a different plan. Exception: shared infra orders can run alongside a plan's phases.

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

**Shared infrastructure:** When multiple plans depend on common infrastructure (shared types, utilities, base packages), propose a standalone infra order before the plan's phases. Use a descriptive slug ID (e.g., `"infra-shared-types"`). If the infra work is substantial, create a backlog item via the adapter (`noodle adapter run backlog add`), then use that item's ID as the order ID.

**Items with plans:** When a backlog item has a `plan` field (a relative path like `brain/plans/29-foo/overview.md`), read the plan overview and phase files to understand the work. Schedule an order for each remaining unfinished phase (each unchecked `- [ ]` item). Populate `order.plan` with the plan path(s). Use `extra_prompt` to inject plan context: the plan overview summary, the specific phase brief, and any cross-phase dependencies.

**Parallelizing phases:** Read the plan to identify dependencies between phases. Phases that depend on earlier phases' output (shared types, APIs, schemas) must be ordered sequentially. Phases that touch independent areas of the codebase (separate packages, unrelated features, docs vs code) can be scheduled as separate orders that run in parallel. When in doubt, sequential is safer — but don't serialize work that has no real dependency.

**Items without plans:** Assess complexity before scheduling. If the item is straightforward (single concern, clear scope, small change), schedule as a simple execute task using the backlog item's title and description as the prompt. If the item is complex (multi-file, cross-cutting, ambiguous scope, or you'd want to see an architecture sketch before coding), schedule a plan-first order: a `prompt`-only stage (no `task_key`) that invokes `/plan`, followed by an `adversarial-review` stage to challenge the plan. No quality or reflect stages — planning output is a design document, not code. **Do NOT use `task_key: "execute"` for plan-first stages** — the execute skill tells the agent to implement, which conflicts with the plan skill's "stop after planning" instruction. The plan skill will write phased plans to `brain/plans/`; on the next scheduling cycle, the item will have a `plan` field and can be scheduled normally with the standard execute → quality → reflect pipeline.

**Standalone orders:** Orders can have arbitrary IDs — they don't need to correspond to a backlog item. When a standalone order completes, the `backlog done` adapter call is a no-op (no matching item to mark done). Use standalone orders for shared infrastructure, maintenance tasks, or cross-cutting work that serves multiple backlog items.

**Nothing to schedule:** When no backlog items are actionable (all blocked, all in-progress, all done, etc.), still write `orders-next.json` with an empty orders array (`{"orders":[]}`). This signals to the loop that scheduling ran but found nothing — preventing hot-loop re-spawns.

### Follow-Up and Standalone Stages

Each task type's `schedule` field describes when and how to schedule it — as a follow-up stage within an order, as a standalone order, or both. Read these hints from `task_types` in mise and compose orders accordingly.

## Recent Events

The mise brief includes a `recent_events` array — lifecycle events emitted by the loop since the last schedule run. These are context for your scheduling decisions, not commands. See [references/events.md](references/events.md) for the full event type catalog (internal and external).

### Using Events for Scheduling

Events are context, not commands. Consider them alongside backlog state and session history when deciding what to schedule:

- After `stage.failed` or `order.failed` — consider whether the failure needs a debugging order, or if the item should be retried with a different approach.
- After `order.completed` — consider follow-up work (reflect, related items that were blocked).
- After `merge.conflict` — the affected order may need manual attention; avoid re-scheduling it immediately.
- After external events like `ci.failed` — consider scheduling an investigation or fix order if it seems actionable.
- After `registry.rebuilt` — new task types may be available; check `task_types` for scheduling opportunities.

Don't react mechanically to every event. Use judgment: a single stage failure in a long pipeline is normal; three consecutive failures of the same order suggests a deeper problem.

## Scheduling Heuristics

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

See [references/examples.md](references/examples.md) for order JSON examples and `extra_prompt` field usage.

## Principles

- [[cost-aware-delegation]]
- [[foundational-thinking]]
- [[subtract-before-you-add]]
- [[never-block-on-the-human]]
- [[guard-the-context-window]]
