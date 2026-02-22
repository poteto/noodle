---
schema_version: 1
expected_failure: false
bug: false
regression: runtime-repair-adopt-running-session
source_hash: 84321288798e53f891cca164a5661c49aa124980a96532f2adfe27ef1ecea378
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
    }
  }
}
```
