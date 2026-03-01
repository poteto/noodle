---
name: schedule
description: Reads backlog and produces work orders for the loop.
schedule: "When orders are empty or after backlog changes"
---

# Schedule

Read `brain/todos.md` for backlog items. Write `.noodle/orders-next.json` with orders for each unchecked item.

Each order has a single execute stage:

```json
{
  "orders": [
    {
      "id": "1",
      "title": "the backlog item title",
      "rationale": "unchecked backlog item ready for work",
      "status": "active",
      "stages": [
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "claude",
          "model": "claude-sonnet-4-6",
          "runtime": "process",
          "status": "pending"
        }
      ]
    }
  ]
}
```

When no unchecked items remain, write `{"orders": []}`.
