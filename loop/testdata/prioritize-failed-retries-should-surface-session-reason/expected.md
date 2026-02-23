---
schema_version: 1
expected_failure: true
bug: true
source_hash: 77a0bd32767ead22dc5ac365b1d7f71725b3b588e97a06f9ef3e3089078bd83f
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "running",
      "runtime_repair_in_flight": false,
      "repair_task_scheduled": false,
      "oops_task_scheduled": false,
      "normal_task_scheduled": true,
      "spawn_calls": 1,
      "runtime_repair_spawn_calls": 0,
      "normal_spawn_calls": 1,
      "created_worktrees": 0,
      "first_spawn": {
        "name": "prioritize",
        "skill": "prioritize",
        "provider": "claude",
        "model": "claude-sonnet-4-6"
      }
    },
    "state-02": {
      "transition": "paused",
      "runtime_repair_in_flight": true,
      "repair_task_scheduled": true,
      "oops_task_scheduled": false,
      "normal_task_scheduled": true,
      "spawn_calls": 2,
      "runtime_repair_spawn_calls": 1,
      "normal_spawn_calls": 1,
      "created_worktrees": 1,
      "first_spawn": {
        "name": "prioritize",
        "skill": "prioritize",
        "provider": "claude",
        "model": "claude-sonnet-4-6"
      },
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
      "normal_task_scheduled": true,
      "spawn_calls": 3,
      "runtime_repair_spawn_calls": 2,
      "normal_spawn_calls": 1,
      "created_worktrees": 2,
      "first_spawn": {
        "name": "prioritize",
        "skill": "prioritize",
        "provider": "claude",
        "model": "claude-sonnet-4-6"
      },
      "runtime_repair_spawn": {
        "name": "repair-runtime-*",
        "skill": "debugging",
        "provider": "claude",
        "model": "claude-sonnet-4-6"
      }
    },
    "state-04": {
      "transition": "paused",
      "runtime_repair_in_flight": true,
      "repair_task_scheduled": true,
      "oops_task_scheduled": false,
      "normal_task_scheduled": true,
      "spawn_calls": 4,
      "runtime_repair_spawn_calls": 3,
      "normal_spawn_calls": 1,
      "created_worktrees": 3,
      "first_spawn": {
        "name": "prioritize",
        "skill": "prioritize",
        "provider": "claude",
        "model": "claude-sonnet-4-6"
      },
      "runtime_repair_spawn": {
        "name": "repair-runtime-*",
        "skill": "debugging",
        "provider": "claude",
        "model": "claude-sonnet-4-6"
      }
    },
    "state-05": {
      "cycle_error": "runtime issue unresolved after 3 repair attempt(s) (loop.collect): prioritize failed after retries: cook exited with status failed",
      "transition": "paused",
      "runtime_repair_in_flight": false,
      "repair_task_scheduled": true,
      "oops_task_scheduled": false,
      "normal_task_scheduled": true,
      "spawn_calls": 4,
      "runtime_repair_spawn_calls": 3,
      "normal_spawn_calls": 1,
      "created_worktrees": 3,
      "first_spawn": {
        "name": "prioritize",
        "skill": "prioritize",
        "provider": "claude",
        "model": "claude-sonnet-4-6"
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
