---
schema_version: 1
expected_failure: false
bug: false
regression: runtime-repair-oops-fallback-custom-routing
source_hash: f65cccad32f066a8ea71d15970109a9a3f0ff945cfdcdeb1167b2120bd763fdf
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
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
