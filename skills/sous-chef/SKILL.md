# Sous Chef

Read `.noodle/mise.json` and write `.noodle/queue.json`.

Rules:
- Prioritize high-impact, unblocker, and dependency-order tasks first.
- Respect `routing.defaults` and `routing.tags` from the mise.
- Prefer independent tasks in parallel when resources allow.
- Set `review: false` only for trivial low-risk work.

Output contract:
```json
{"generated_at":"...","items":[{"id":"42","provider":"claude","model":"claude-sonnet-4-6","review":true,"rationale":"..."}]}
```
