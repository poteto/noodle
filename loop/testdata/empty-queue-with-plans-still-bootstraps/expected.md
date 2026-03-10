---
schema_version: 1
expected_failure: false
bug: false
source_hash: 3aef715322fced1c8f9ecf0295d64ad4a6660edf7e3b1e56f71ab319c1571eb2
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "running",
      "normal_task_scheduled": true,
      "spawn_calls": 1,
      "normal_spawn_calls": 1,
      "created_worktrees": 0,
      "active_summary_total": 1,
      "first_spawn": {
        "name": "schedule",
        "skill": "schedule",
        "provider": "codex",
        "model": "gpt-5.4"
      }
    }
  }
}
```
