---
schema_version: 1
expected_failure: false
bug: false
source_hash: f729c647c96b32661a76e88dfee5e9f9d1267425c125af3dcf103f14c0c8eef1
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
