---
schema_version: 1
expected_failure: false
bug: false
source_hash: f63ad45c1c197c9c5c9eccaaff1a89aa89cfa63dfa6448c0e3b94e5423241acc
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
        "name": "plan-42-0-execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
