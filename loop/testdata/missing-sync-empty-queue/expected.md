---
schema_version: 1
expected_failure: false
bug: false
regression: missing-sync-empty-queue
source_hash: 3c4f83377e569954a38d2e10d02c1dcf3d58f731f4e724701d7d070102c60f34
---

## Expected

```json
{
  "states": {
    "state-01": {
      "error": {
        "absent": true
      },
      "transition": "paused",
      "actions": {
        "normal_task_scheduled": false,
        "repair_task_scheduled": true
      },
      "state": {
        "paused": true,
        "runtime_repair_in_flight": true
      },
      "counts": {
        "created_worktrees": {
          "eq": 1
        },
        "normal_spawn_calls": {
          "eq": 0
        },
        "runtime_repair_spawn_calls": {
          "eq": 1
        },
        "spawn_calls": {
          "eq": 1
        }
      },
      "routing": {
        "runtime_repair_name": {
          "prefix": "repair-runtime-"
        },
        "runtime_repair_prompt": {
          "contains": "Scope: mise.sync"
        },
        "runtime_repair_skill": {
          "equals": "debugging"
        }
      }
    }
  }
}
```
