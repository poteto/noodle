---
schema_version: 1
expected_failure: false
bug: false
source_hash: 0affe71c426b6be74a5a5e1f4636661682e6cf8f95328531673b84ab14d3267a
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
      "created_worktrees": 1,
      "active_summary_total": 1,
      "first_spawn": {
        "name": "42-0-execute",
        "provider": "codex",
        "model": "gpt-5.4"
      }
    }
  }
}
```
