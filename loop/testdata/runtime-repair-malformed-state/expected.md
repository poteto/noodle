---
schema_version: 1
expected_failure: false
bug: false
regression: runtime-repair-malformed-state
source_hash: 82821bd83e914b26620ef2b2952f2b906f62eef149c14b0f76cec54d6ed3e579
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
        "oops_task_scheduled": false,
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
          "contains": "Scope: mise.build"
        },
        "runtime_repair_skill": {
          "equals": "debugging"
        }
      }
    }
  }
}
```
