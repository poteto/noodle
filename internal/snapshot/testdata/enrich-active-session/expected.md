---
schema_version: 1
expected_failure: false
bug: false
source_hash: c23453896ff406677e4f0eebbb2966d286eb27b4c34ded41ed363741476ccd66
---

## Expected Snapshot

```json
{
  "updated_at": "2026-02-27T12:00:00Z",
  "loop_state": "running",
  "sessions": [
    {
      "id": "cook-active",
      "display_name": "Chef Active",
      "status": "running",
      "runtime": "",
      "provider": "claude",
      "model": "claude-sonnet-4-6",
      "total_cost_usd": 0.12,
      "duration_seconds": 9223372036,
      "last_activity": "2026-02-27T12:00:00Z",
      "current_action": "Bash go test ./...",
      "health": "",
      "context_window_usage_pct": 0.5,
      "retry_count": 0,
      "idle_seconds": 0,
      "stuck_threshold_seconds": 0,
      "loop_state": "",
      "task_key": "execute"
    }
  ],
  "active": [
    {
      "id": "cook-active",
      "display_name": "Chef Active",
      "status": "running",
      "runtime": "",
      "provider": "claude",
      "model": "claude-sonnet-4-6",
      "total_cost_usd": 0.12,
      "duration_seconds": 9223372036,
      "last_activity": "2026-02-27T12:00:00Z",
      "current_action": "Bash go test ./...",
      "health": "",
      "context_window_usage_pct": 0.5,
      "retry_count": 0,
      "idle_seconds": 0,
      "stuck_threshold_seconds": 0,
      "loop_state": "",
      "task_key": "execute"
    }
  ],
  "recent": [],
  "orders": [
    {
      "id": "order-1",
      "title": "Feature X",
      "stages": [
        {
          "task_key": "execute",
          "skill": "execute",
          "provider": "claude",
          "model": "claude-sonnet-4-6",
          "status": "active",
          "session_id": "cook-active"
        }
      ],
      "status": "active"
    }
  ],
  "active_order_ids": [
    "order-1"
  ],
  "action_needed": [],
  "events_by_session": {},
  "feed_events": [],
  "total_cost_usd": 0,
  "pending_reviews": [],
  "pending_review_count": 0,
  "autonomy": "",
  "max_cooks": 0
}
```
