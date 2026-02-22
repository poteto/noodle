---
schema_version: 1
expected_failure: false
bug: false
source_hash: dace5849d5aaf595f23e15d27fcbe463526263a439cd3fdc18ceaf3168e7e26e
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "paused",
      "runtime_repair_in_flight": true,
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
    },
    "state-02": {
      "transition": "running",
      "runtime_repair_in_flight": false,
      "repair_task_scheduled": true,
      "oops_task_scheduled": false,
      "normal_task_scheduled": true,
      "spawn_calls": 2,
      "runtime_repair_spawn_calls": 1,
      "normal_spawn_calls": 1,
      "created_worktrees": 2,
      "first_spawn": {
        "name": "42",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      },
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
