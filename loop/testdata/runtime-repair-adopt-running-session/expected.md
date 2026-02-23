---
schema_version: 1
expected_failure: false
bug: false
source_hash: 61bedb7cdf62a5a4c14077c6635d0d84ee9a984effb66ad58be52e4685e8266a
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
    }
  }
}
```
