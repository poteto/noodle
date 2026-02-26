---
schema_version: 1
expected_failure: false
bug: false
source_hash: 675e98fef6a7da96cb520673e18f63c0d0a64833e75b0ca75f0e5cb1813be149
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
      "first_spawn": {
        "name": "42-0-execute",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
