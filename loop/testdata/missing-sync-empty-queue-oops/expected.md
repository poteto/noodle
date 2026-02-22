---
schema_version: 1
expected_failure: false
bug: false
regression: missing-sync-empty-queue-oops
source_hash: 100e245da5797eb4067442798583737cb971438151a76acc0cf7ef53a1970792
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
        "oops_task_scheduled": true,
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
          "equals": "oops"
        }
      }
    }
  }
}
```
