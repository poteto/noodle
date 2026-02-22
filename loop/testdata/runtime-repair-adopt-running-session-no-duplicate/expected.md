---
schema_version: 1
expected_failure: false
bug: false
regression: runtime-repair-adopt-running-session-no-duplicate
source_hash: 6ebff87da790ca8a399aaf011127726b8a0ba6fe001aa1d9a29adb567f0c7af4
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
      "transition": "paused",
      "actions": {
        "normal_task_scheduled": false,
        "repair_task_scheduled": false
      },
      "state": {
        "paused": true,
        "runtime_repair_in_flight": true
      },
      "counts": {
        "created_worktrees": {
          "eq": 0
        },
        "normal_spawn_calls": {
          "eq": 0
        },
        "runtime_repair_spawn_calls": {
          "eq": 0
        },
        "spawn_calls": {
          "eq": 0
        }
      },
      "absence": {
        "normal_task_scheduled": true,
        "repair_task_scheduled": true
      },
      "idempotence": {
        "no_duplicate_runtime_repairs_on_extra_cycles": true,
        "no_new_spawns_on_extra_cycles": true
      }
    }
  }
}
```
