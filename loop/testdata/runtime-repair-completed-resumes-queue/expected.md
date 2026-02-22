---
schema_version: 1
expected_failure: false
bug: false
regression: runtime-repair-completed-resumes-queue
source_hash: dace5849d5aaf595f23e15d27fcbe463526263a439cd3fdc18ceaf3168e7e26e
---

## Expected

```json
{
  "states": {
    "state-01": {
      "error": {
        "absent": true
      },
      "transition": "paused"
    },
    "state-02": {
      "transition": "running",
      "actions": {
        "normal_task_scheduled": true,
        "repair_task_scheduled": true
      },
      "state": {
        "running": true,
        "runtime_repair_in_flight": false
      },
      "counts": {
        "created_worktrees": {
          "eq": 2
        },
        "normal_spawn_calls": {
          "eq": 1
        },
        "runtime_repair_spawn_calls": {
          "eq": 1
        },
        "spawn_calls": {
          "eq": 2
        }
      },
      "routing": {
        "runtime_repair_name": {
          "prefix": "repair-runtime-"
        },
        "runtime_repair_skill": {
          "equals": "debugging"
        }
      }
    }
  }
}
```
