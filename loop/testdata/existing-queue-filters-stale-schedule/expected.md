---
schema_version: 1
expected_failure: false
bug: false
source_hash: 92f406625ad3412a14e2bf2e0fb6aae8a9c3cf6873cf87a528e51f8e1ffca094
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "running",
      "normal_task_scheduled": true,
      "spawn_calls": 2,
      "normal_spawn_calls": 2,
      "created_worktrees": 1,
      "active_summary_total": 2,
      "first_spawn": {
        "name": "42-0-execute",
        "provider": "codex",
        "model": "gpt-5.4"
      }
    }
  }
}
```
