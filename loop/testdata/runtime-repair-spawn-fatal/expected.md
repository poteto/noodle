---
schema_version: 1
expected_failure: true
bug: false
source_hash: 088caa230273ca15f121a34493c401ee7ea71bd505b518c11343134e78a1f99d
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "cycle_error": "runtime repair unavailable (mise.build): agent unavailable",
      "transition": "paused",
      "runtime_repair_in_flight": false,
      "repair_task_scheduled": true,
      "oops_task_scheduled": false,
      "normal_task_scheduled": false,
      "spawn_calls": 1,
      "runtime_repair_spawn_calls": 1,
      "normal_spawn_calls": 0,
      "created_worktrees": 1,
      "runtime_repair_spawn": {
        "name": "repair-runtime-*",
        "skill": "debugging",
        "provider": "claude",
        "model": "claude-sonnet-4-6"
      }
    }
  }
}
```
