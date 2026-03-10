---
schema_version: 1
expected_failure: false
bug: false
source_hash: a7e2e788dac1d3d2bbd23378a38cd7dc173dae8a9689c24583e119834f70257d
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
