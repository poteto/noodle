# Sous Chef

Read `.noodle/mise.json` and write `.noodle/queue.json`.

Rules:
- Prioritize high-impact, unblocker, and dependency-order tasks first.
- Respect `routing.defaults` and `routing.tags` from the mise.
- Prefer independent tasks in parallel when resources allow.
- Set `review: false` only for trivial low-risk work.
- If a backlog to-do has no corresponding plan, schedule a `Plan` task for that to-do before any `Execute` task.
  A to-do has no corresponding plan when either:
  - `backlog[i].plan` is empty, or
  - `backlog[i].plan` is set but no `plans[j].id` matches it.
- For these gaps, emit a queue item with `task_key: "plan"` for the to-do and do not enqueue `task_key: "execute"` for that same to-do until the plan exists.

Output contract:
```json
{"generated_at":"...","items":[{"id":"42","task_key":"plan","provider":"claude","model":"claude-opus-4-6","review":true,"rationale":"..."}]}
```
