---
schema_version: 1
expected_failure: false
bug: false
source_hash: aeee7b8a033309b92043eb01f4e5d1b733f4053e7f8ccf8384ba53445521bd96
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "paused",
      "runtime_repair_in_flight": true,
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
    },
    "state-02": {
      "transition": "paused",
      "runtime_repair_in_flight": true,
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
    },
    "state-03": {
      "transition": "paused",
      "runtime_repair_in_flight": true,
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
