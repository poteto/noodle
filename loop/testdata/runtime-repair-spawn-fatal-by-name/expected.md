---
schema_version: 1
expected_failure: true
bug: false
source_hash: 92e72e9771edd6587c95959de1785a9d8be87b41305804cd6e70ebfd92e4fa33
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
      "oops_task_scheduled": true,
      "normal_task_scheduled": false,
      "spawn_calls": 1,
      "runtime_repair_spawn_calls": 1,
      "normal_spawn_calls": 0,
      "created_worktrees": 1,
      "runtime_repair_spawn": {
        "name": "repair-runtime-*",
        "skill": "oops",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
