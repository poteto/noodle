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
      "duration_seconds": 600,
      "last_activity": "2026-02-27T11:58:00Z",
      "current_action": "go test ./...",
      "health": "green",
      "context_window_usage_pct": 0,
      "retry_count": 0,
      "idle_seconds": 5,
      "stuck_threshold_seconds": 120,
      "loop_state": "",
      "worktree_name": "order-1-0-execute",
      "task_key": "execute",
      "title": "Feature X"
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
      "duration_seconds": 600,
      "last_activity": "2026-02-27T11:58:00Z",
      "current_action": "go test ./...",
      "health": "green",
      "context_window_usage_pct": 0,
      "retry_count": 0,
      "idle_seconds": 5,
      "stuck_threshold_seconds": 120,
      "loop_state": "",
      "worktree_name": "order-1-0-execute",
      "task_key": "execute",
      "title": "Feature X"
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
          "status": "active"
        }
      ],
      "status": "active"
    }
  ],
  "active_order_ids": [
    "order-1"
  ],
  "action_needed": null,
  "events_by_session": {
    "cook-active": [
      {
        "at": "2026-02-27T11:50:00Z",
        "label": "Read",
        "body": "Reading main.go",
        "category": "tools"
      },
      {
        "at": "2026-02-27T11:52:00Z",
        "label": "Edit",
        "body": "Editing handler.go",
        "category": "tools"
      },
      {
        "at": "2026-02-27T11:54:00Z",
        "label": "Cost",
        "body": "$0.05 | 1.0k in / 500 out",
        "category": "all"
      },
      {
        "at": "2026-02-27T11:56:00Z",
        "label": "Bash",
        "body": "go test ./...",
        "category": "tools"
      }
    ]
  },
  "feed_events": [
    {
      "session_id": "cook-active",
      "agent_name": "briny-sorrel",
      "task_type": "",
      "at": "2026-02-27T11:50:00Z",
      "label": "Read",
      "body": "Reading main.go",
      "category": "tools"
    },
    {
      "session_id": "cook-active",
      "agent_name": "briny-sorrel",
      "task_type": "",
      "at": "2026-02-27T11:52:00Z",
      "label": "Edit",
      "body": "Editing handler.go",
      "category": "tools"
    },
    {
      "session_id": "cook-active",
      "agent_name": "briny-sorrel",
      "task_type": "",
      "at": "2026-02-27T11:54:00Z",
      "label": "Cost",
      "body": "$0.05 | 1.0k in / 500 out",
      "category": "all"
    },
    {
      "session_id": "cook-active",
      "agent_name": "briny-sorrel",
      "task_type": "",
      "at": "2026-02-27T11:56:00Z",
      "label": "Bash",
      "body": "go test ./...",
      "category": "tools"
    }
  ],
  "total_cost_usd": 0.12,
  "pending_reviews": [],
  "pending_review_count": 0,
  "autonomy": "auto",
  "max_cooks": 0
}
```
