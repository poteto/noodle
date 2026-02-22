---
schema_version: 1
expected_failure: false
bug: false
regression: runtime-repair-inflight-suppresses-queued-work
source_hash: 303f519920f7ae3ee7fa38357eefdbc098a4594fa1b82b5cd53dbfda4f618707
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
    "state-03": {
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
    }
  }
}
```
