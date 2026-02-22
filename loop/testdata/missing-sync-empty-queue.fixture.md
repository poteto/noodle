# Loop Fixture: Missing Sync + Empty Queue

## Setup
```json
{
  "queue_items": [],
  "mise_results": [
    {
      "warnings": [
        "backlog sync script missing; returning empty backlog"
      ]
    }
  ]
}
```

## Expected
```json
{
  "spawn_calls": 1,
  "first_spawn_name_prefix": "repair-runtime-",
  "created_worktrees": 1,
  "runtime_repair_in_flight": true
}
```
