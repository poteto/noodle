---
schema_version: 1
expected_failure: false
bug: false
source_hash: 9028558862ca499f4f59f00a7eee7ec2673e965da3f9d12a78597dc468c9229b
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
      "active_summary_total": 2,
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
