---
schema_version: 1
expected_failure: false
bug: false
source_hash: 65a28fdadfc6672e5b6630bcb18b4b0af59310668163ddedfb59af05b8141342
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
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
