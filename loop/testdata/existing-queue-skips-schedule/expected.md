---
schema_version: 1
expected_failure: false
bug: false
source_hash: 56be5e4599f4bf9a2fd7b7e8517a1ac88615d163c7c9b0c8f385126c20bf2206
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
      "created_worktrees": 2,
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
