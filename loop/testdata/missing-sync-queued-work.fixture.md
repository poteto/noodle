# Loop Fixture: Missing Sync + Existing Queue Work

## Setup
```json
{
  "queue_items": [
    {
      "id": "42",
      "provider": "claude",
      "model": "claude-sonnet-4-6"
    }
  ],
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
  "first_spawn_name": "42",
  "created_worktrees": 1,
  "runtime_repair_in_flight": false
}
```
