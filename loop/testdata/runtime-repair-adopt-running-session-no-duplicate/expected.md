---
schema_version: 1
expected_failure: false
bug: false
regression: runtime-repair-adopt-running-session-no-duplicate
source_hash: 097e29effa7f3cb7b71d07c95abe8570794ec8fcf7e98e067226943d4d943e0a
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "paused",
      "runtime_repair_in_flight": true,
      "repair_task_scheduled": false,
      "oops_task_scheduled": false,
      "normal_task_scheduled": false,
      "spawn_calls": 0,
      "runtime_repair_spawn_calls": 0,
      "normal_spawn_calls": 0,
      "created_worktrees": 0
    },
    "state-02": {
      "transition": "paused",
      "runtime_repair_in_flight": true,
      "repair_task_scheduled": false,
      "oops_task_scheduled": false,
      "normal_task_scheduled": false,
      "spawn_calls": 0,
      "runtime_repair_spawn_calls": 0,
      "normal_spawn_calls": 0,
      "created_worktrees": 0
    }
  }
}
```
