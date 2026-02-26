---
schema_version: 1
expected_failure: false
bug: false
source_hash: a26572f944d5257154b559719699dcaef1c8bca3000c846d5263849e6d826a7d
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
      "first_spawn": {
        "name": "42-0-execute",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
