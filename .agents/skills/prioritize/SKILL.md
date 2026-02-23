---
name: prioritize
description: Queue scheduler. Reads .noodle/mise.json, writes .noodle/queue.json. Schedules work based on backlog state, plan phases, session history, and task type schedules.
noodle:
  blocking: true
  schedule: "When the queue is empty, after backlog changes, or when session history suggests re-evaluation"
---

# Prioritize

Read `.noodle/mise.json`, write `.noodle/queue.json`.
Use `noodle schema mise` and `noodle schema queue` as the schema source of truth.

Operate fully autonomously. Never ask the user to choose or pause for confirmation.

## Plans as Precondition

Only schedule `execute` items that have a linked plan (`plan` field non-null in backlog entry). Skip unplanned items entirely -- note their IDs in `queue.json` under `"action_needed"` so the TUI can surface them.

## Schedule Reading

Read `task_types[].schedule` from mise to know when each task type should run. Honor these hints when placing items.

## Workflow Order

Read `task_types` from mise to discover available task types and their `schedule` hints. Hard constraints:

1. **Execute** -> **Quality** (blocking — no other work until quality clears)
2. **Quality rejection** -> re-queue the item for retry with feedback
3. **Reflect** periodically after completed work (check `recent_history`)
4. **Meditate** after ~3 completed Reflects (check `recent_history` + current queue)

When any item has `"review": true`, it must be the only item type in the queue (blocking). Do not mix other task types into the same queue generation.

## Situational Awareness

| Trigger | Action |
|---------|--------|
| Empty queue | Full survey of mise -- schedule from scratch |
| Quality rejection | Rescope the rejected item for retry with feedback |
| New items with plans | Slot into existing queue respecting workflow order |
| Unplanned items | Skip, add to `action_needed` |
| All items blocked | Schedule meditate or reflect to use the slot productively |

## CEO Patterns

- **Foundation before feature**: Infrastructure and shared types first.
- **Cheapest mode**: Prefer the lowest-cost provider/model that can handle the task.
- **Explicit rationale**: Every queue item must cite which principle or rule drove its placement.
- **Work around blockers**: If the top-priority item is blocked, schedule the next viable item -- never idle.
- **Timebox failures**: If an item has failed 2+ times in `recent_history`, deprioritize or split it.

## Model Routing

| Task type | Provider | Model |
|-----------|----------|-------|
| Implementation, execution, coding | codex | gpt-5.3-codex |
| Judgment, strategy, planning, quality | claude | claude-opus-4-6 |

When uncertain, codex for implementation, opus for judgment.

## Output

Write valid JSON to `.noodle/queue.json` matching `noodle schema queue`.

## Principles

- [[cost-aware-delegation]]
- [[foundational-thinking]]
- [[subtract-before-you-add]]
- [[never-block-on-the-human]]
- [[guard-the-context-window]]
