---
schema_version: 1
expected_failure: false
bug: false
source_hash: 136774ec9c4669f340b3f99e18a334c181b312c6b7606536a27692064ad8b9cc
---

## Expected Snapshot

```json
{
  "updated_at": "2026-02-27T12:00:00Z",
  "loop_state": "running",
  "sessions": [],
  "active": [],
  "recent": [],
  "orders": [],
  "active_order_ids": null,
  "action_needed": null,
  "events_by_session": {},
  "feed_events": [
    {
      "session_id": "loop",
      "agent_name": "loop",
      "task_type": "",
      "at": "2026-02-27T11:01:00Z",
      "label": "Bootstrap",
      "body": "Creating schedule skill from workflow analysis",
      "category": "bootstrap"
    },
    {
      "session_id": "chef",
      "agent_name": "cook-a",
      "task_type": "",
      "at": "2026-02-27T11:02:00Z",
      "label": "Steer",
      "body": "focus on tests",
      "category": "steer"
    },
    {
      "session_id": "loop",
      "agent_name": "loop",
      "task_type": "",
      "at": "2026-02-27T11:03:00Z",
      "label": "Rebuild",
      "body": "Registry rebuilt — added: [execute], removed: []",
      "category": "registry_rebuild"
    },
    {
      "session_id": "chef",
      "agent_name": "cook-b",
      "task_type": "",
      "at": "2026-02-27T11:04:00Z",
      "label": "Steer",
      "body": "fix the bug",
      "category": "steer"
    }
  ],
  "total_cost_usd": 0,
  "pending_reviews": [],
  "pending_review_count": 0,
  "autonomy": "auto",
  "max_cooks": 0
}
```
