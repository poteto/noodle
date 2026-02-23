---
name: prioritize
description: >
  Noodle queue scheduler. Use when reading `.noodle/mise.json` and writing
  `.noodle/queue.json`, especially when enforcing workflow order constraints:
  Plan must be followed by blocking Chef Review, Execute must be followed by
  Verify, Verify must be followed by Reflect, and Meditate should be scheduled
  periodically after several Reflects.
noodle:
  blocking: true
  schedule: "When the queue is empty, after backlog changes, or when session history suggests re-evaluation"
---

# Prioritize

Read `.noodle/mise.json` and write `.noodle/queue.json`.

Operate fully autonomously. Never ask the user to choose between options, and
never pause for confirmation.

## Required Workflow Order

1. `Plan -> Review from Chef` (blocking)
2. `Execute -> Verify`
3. `Verify -> Reflect`
4. `Meditate` after several `Reflect` tasks

Treat these as hard scheduling constraints.

## Blocking Rule

If any open backlog item is a Chef review item, schedule Chef review before all
other work.

To enforce blocking with parallel workers, output only Chef review items until
Chef review is cleared. Do not include Execute, Verify, Reflect, or Meditate
items in the same queue generation when a Chef review item is pending.

## Stage Classification

Classify each open backlog item by title/tags/description keywords
(case-insensitive):

- `plan`
- `review from chef` (or `chef review`)
- `execute`
- `verify`
- `reflect`
- `meditate`

If an item cannot be classified, treat it as general execution work and place
it only when it does not violate the required workflow order.

## Queue Construction Policy

When no blocking Chef review is pending:

1. Prioritize high-impact/unblocker work while respecting dependencies.
2. Do not schedule `Execute` ahead of unresolved `Review from Chef`.
3. If you schedule an `Execute` item, place a `Verify` item immediately after
   it when one is available.
4. If you schedule a `Verify` item, place a `Reflect` item immediately after it
   when one is available.
5. Schedule one `Meditate` item after roughly every 3 completed/scheduled
   `Reflect` items (use `recent_history` plus planned queue context).

## Model Routing Policy

Set `provider` and `model` per queue item using these defaults:

1. Front-end tasks and judgment-heavy tasks (for example planning and backlog
   grooming): `provider: "claude"`, `model: "claude-opus-4-6"`.
2. Coding execution tasks and tasks that require strong implementation focus:
   `provider: "codex"`, `model: "gpt-5.3-codex"`.

When uncertain, prefer Codex for direct implementation work and Opus for
strategic prioritization or nuanced product judgment work.

## Output Contract

Write valid JSON with this shape:

```json
{"generated_at":"...","items":[{"id":"42","provider":"codex","model":"gpt-5.3-codex","review":true,"rationale":"..."}]}
```

Keep rationale brief and explicit about which ordering rule drove each
placement.
