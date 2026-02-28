---
schema_version: 1
expected_failure: false
bug: false
source_hash: 5715902fff2c2c6950ce8e2ddff96d0c93c14e6707364fcc1d48b84309060c30
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
  "active_order_ids": [],
  "action_needed": [],
  "events_by_session": {},
  "feed_events": [
    {
      "session_id": "chef",
      "agent_name": "cook-x",
      "task_type": "",
      "at": "2026-02-27T11:00:00Z",
      "label": "Steer",
      "body": "step 1",
      "category": "steer"
    },
    {
      "session_id": "loop",
      "agent_name": "loop",
      "task_type": "",
      "at": "2026-02-27T11:01:00Z",
      "label": "Bootstrap",
      "body": "Bootstrap completed",
      "category": "bootstrap"
    },
    {
      "session_id": "chef",
      "agent_name": "cook-y",
      "task_type": "",
      "at": "2026-02-27T11:02:00Z",
      "label": "Steer",
      "body": "step 3",
      "category": "steer"
    },
    {
      "session_id": "loop",
      "agent_name": "loop",
      "task_type": "",
      "at": "2026-02-27T11:03:00Z",
      "label": "Rebuild",
      "body": "Registry rebuilt — added: [execute], removed: [old]",
      "category": "registry_rebuild"
    },
    {
      "session_id": "chef",
      "agent_name": "cook-z",
      "task_type": "",
      "at": "2026-02-27T11:04:00Z",
      "label": "Steer",
      "body": "step 5",
      "category": "steer"
    }
  ],
  "total_cost_usd": 0,
  "pending_reviews": [],
  "pending_review_count": 0,
  "mode": "",
  "max_cooks": 0
}
```
