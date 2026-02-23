# queue.json Schema

Output of the prioritize skill. Consumed by the dispatcher.

```json
{
  "generated_at": "ISO 8601 timestamp",
  "items": [
    {
      "id": "string — backlog item ID",
      "task_key": "string — prioritize|execute|reflect|meditate|oops|debate|quality",
      "title": "string — brief description of the work",
      "provider": "string — claude|codex",
      "model": "string — model ID",
      "skill": "string — skill name to load for this task",
      "review": "boolean — true if this item requires blocking review",
      "rationale": "string — why this placement, citing principle or scheduling rule"
    }
  ],
  "action_needed": [
    "string — backlog item IDs skipped (no linked plan, needs user attention)"
  ]
}
```

## Constraints

- Items must respect workflow order: execute -> quality (blocking) -> reflect.
- When any item has `"review": true`, it must be the only item type in the queue (blocking).
- Task keys must match a `task_types[].key` from mise. Unknown keys are rejected at validation.
- Each `rationale` must name a specific principle or scheduling rule.
