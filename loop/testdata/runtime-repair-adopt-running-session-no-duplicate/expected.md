---
schema_version: 1
expected_failure: false
bug: false
source_hash: 837c551f31974f19ce809e2bbed34f20d8b3317b403ac6c6a74d4b79c4879589
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
        "model": "claude-sonnet-4-6"
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
        "model": "claude-sonnet-4-6"
      }
    }
  }
}
```
