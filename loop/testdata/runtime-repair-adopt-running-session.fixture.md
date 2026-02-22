# Loop Fixture: Adopt Running Runtime Repair Session

## Setup
```json
{
  "queue_items": [],
  "mise_results": [
    {
      "error": "backlog sync failed"
    }
  ],
  "running_runtime_repair_session_id": "repair-runtime-20260222-185500-1-abc123"
}
```

## Expected
```json
{
  "spawn_calls": 0,
  "created_worktrees": 0,
  "runtime_repair_in_flight": true
}
```
