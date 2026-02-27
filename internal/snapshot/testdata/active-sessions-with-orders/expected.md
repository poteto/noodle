---
schema_version: 1
expected_failure: false
bug: false
source_hash: a6aa546adc446bfd651688fcbc1c96af44ffc988a8b8d5e69108663faff6a9b4
---

## Expected Snapshot

```json
{
  "updated_at": "2026-02-27T12:00:00Z",
  "loop_state": "running",
  "sessions": [
    {
      "id": "cook-a",
      "display_name": "Chef Alpha",
      "status": "running",
      "runtime": "",
      "provider": "claude",
      "model": "claude-sonnet-4-6",
      "total_cost_usd": 0,
      "duration_seconds": 0,
      "last_activity": "2026-02-27T11:55:00Z",
      "current_action": "",
      "health": "green",
      "context_window_usage_pct": 0,
      "retry_count": 0,
      "idle_seconds": 10,
      "stuck_threshold_seconds": 120,
      "loop_state": "",
      "worktree_name": "order-1-0-execute",
      "task_key": "execute",
      "title": "Implement feature A"
    },
    {
      "id": "cook-b",
      "display_name": "Chef Beta",
      "status": "running",
      "runtime": "",
      "provider": "claude",
      "model": "claude-sonnet-4-6",
      "total_cost_usd": 0,
      "duration_seconds": 0,
      "last_activity": "2026-02-27T11:50:00Z",
      "current_action": "",
      "health": "green",
      "context_window_usage_pct": 0,
      "retry_count": 0,
      "idle_seconds": 5,
      "stuck_threshold_seconds": 120,
      "loop_state": "",
      "worktree_name": "order-2-1-review",
      "task_key": "review",
      "title": "Review feature B"
    }
  ],
  "active": [
    {
      "id": "cook-a",
      "display_name": "Chef Alpha",
      "status": "running",
      "runtime": "",
      "provider": "claude",
      "model": "claude-sonnet-4-6",
      "total_cost_usd": 0,
      "duration_seconds": 0,
      "last_activity": "2026-02-27T11:55:00Z",
      "current_action": "",
      "health": "green",
      "context_window_usage_pct": 0,
      "retry_count": 0,
      "idle_seconds": 10,
      "stuck_threshold_seconds": 120,
      "loop_state": "",
      "worktree_name": "order-1-0-execute",
      "task_key": "execute",
      "title": "Implement feature A"
    },
    {
      "id": "cook-b",
      "display_name": "Chef Beta",
      "status": "running",
      "runtime": "",
      "provider": "claude",
      "model": "claude-sonnet-4-6",
      "total_cost_usd": 0,
      "duration_seconds": 0,
      "last_activity": "2026-02-27T11:50:00Z",
      "current_action": "",
      "health": "green",
      "context_window_usage_pct": 0,
      "retry_count": 0,
      "idle_seconds": 5,
      "stuck_threshold_seconds": 120,
      "loop_state": "",
      "worktree_name": "order-2-1-review",
      "task_key": "review",
      "title": "Review feature B"
    }
  ],
  "recent": [],
  "orders": [
    {
      "id": "order-1",
      "title": "Implement feature A",
      "stages": [
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "claude",
          "model": "claude-sonnet-4-6",
          "status": "active"
        }
      ],
      "status": "active"
    },
    {
      "id": "order-2",
      "title": "Review feature B",
      "stages": [
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "claude",
          "model": "claude-sonnet-4-6",
          "status": "completed"
        },
        {
          "task_key": "review",
          "skill": "review",
          "provider": "claude",
          "model": "claude-sonnet-4-6",
          "status": "active"
        }
      ],
      "status": "active",
      "on_failure": [
        {
          "task_key": "oops",
          "skill": "oops",
          "provider": "claude",
          "model": "claude-sonnet-4-6",
          "status": "pending"
        }
      ]
    }
  ],
  "active_order_ids": [
    "order-1",
    "order-2"
  ],
  "action_needed": null,
  "events_by_session": {
    "cook-a": [],
    "cook-b": []
  },
  "feed_events": [],
  "total_cost_usd": 0,
  "pending_reviews": [],
  "pending_review_count": 0,
  "autonomy": "auto",
  "max_cooks": 3
}
```
