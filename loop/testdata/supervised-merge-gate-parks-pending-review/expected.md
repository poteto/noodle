---
schema_version: 1
expected_failure: false
bug: false
source_hash: b15fd7739642943ac2cb6675bbc817c6b4163c6760f5c41e8393c125fb6c8a9d
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "running",
      "normal_task_scheduled": false,
      "spawn_calls": 0,
      "normal_spawn_calls": 0,
      "created_worktrees": 0,
      "active_summary_total": 0,
      "pending_review": {
        "42": "supervised mode requires merge approval"
      },
      "orders": [
        {
          "id": "42",
          "status": "active",
          "stages": [
            {
              "task_key": "execute",
              "status": "active"
            }
          ]
        }
      ]
    }
  }
}
```
