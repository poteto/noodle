---
schema_version: 1
expected_failure: false
bug: false
regression: bug-routing-stale-queue-config
source_hash: 361d46fb18cbfedf87bbf59fc3ccdd1a371557cd1db3c9c3a0f6a5d3fdddc3ec
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
      "created_worktrees": 1,
      "first_spawn": {
        "name": "10",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
