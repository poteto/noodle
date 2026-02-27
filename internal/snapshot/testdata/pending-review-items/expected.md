---
schema_version: 1
expected_failure: false
bug: false
source_hash: 69c3bb28e886b0cc81e576f083086e4f9ebfe0912e160555ae0d00c326719850
---

## Expected Snapshot

```json
{
  "updated_at": "2026-02-27T12:00:00Z",
  "loop_state": "running",
  "sessions": [],
  "active": [],
  "recent": [],
  "orders": [
    {
      "id": "review-1",
      "title": "Feature with conflict",
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
  "active_order_ids": null,
  "action_needed": null,
  "events_by_session": {},
  "feed_events": [],
  "total_cost_usd": 0,
  "pending_reviews": [
    {
      "order_id": "review-1",
      "stage_index": 0,
      "task_key": "execute",
      "worktree_name": "review-1-0-execute",
      "worktree_path": "/tmp/wt/review-1-0-execute",
      "session_id": "cook-review",
      "reason": "merge conflict on branch noodle/cook-review"
    }
  ],
  "pending_review_count": 1,
  "autonomy": "auto",
  "max_cooks": 0
}
```
