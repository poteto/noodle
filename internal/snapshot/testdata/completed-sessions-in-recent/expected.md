---
schema_version: 1
expected_failure: false
bug: false
source_hash: 6eec590733616ed3302ce8ff853f50484317bd2a3d8155e663e69126a0b5d9e5
---

## Expected Snapshot

```json
{
  "updated_at": "2026-02-27T12:00:00Z",
  "loop_state": "running",
  "sessions": [
    {
      "id": "cook-done",
      "display_name": "golden-cumin",
      "status": "completed",
      "runtime": "",
      "provider": "claude",
      "model": "claude-sonnet-4-6",
      "total_cost_usd": 0.05,
      "duration_seconds": 300,
      "last_activity": "2026-02-27T11:45:00Z",
      "current_action": "",
      "health": "yellow",
      "context_window_usage_pct": 0,
      "retry_count": 0,
      "idle_seconds": 0,
      "stuck_threshold_seconds": 0,
      "loop_state": ""
    }
  ],
  "active": [],
  "recent": [
    {
      "id": "cook-done",
      "display_name": "golden-cumin",
      "status": "completed",
      "runtime": "",
      "provider": "claude",
      "model": "claude-sonnet-4-6",
      "total_cost_usd": 0.05,
      "duration_seconds": 300,
      "last_activity": "2026-02-27T11:45:00Z",
      "current_action": "",
      "health": "yellow",
      "context_window_usage_pct": 0,
      "retry_count": 0,
      "idle_seconds": 0,
      "stuck_threshold_seconds": 0,
      "loop_state": ""
    }
  ],
  "orders": [],
  "active_order_ids": null,
  "action_needed": null,
  "events_by_session": {
    "cook-done": []
  },
  "feed_events": [],
  "total_cost_usd": 0.05,
  "pending_reviews": [],
  "pending_review_count": 0,
  "autonomy": "auto",
  "max_cooks": 0
}
```
