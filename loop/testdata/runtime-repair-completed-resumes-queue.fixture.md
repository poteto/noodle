# Loop Fixture: Runtime Repair Completion Resumes Queue Scheduling

## Setup
```json
{
  "queue_items": [
    {
      "id": "42",
      "provider": "codex",
      "model": "gpt-5.3-codex"
    }
  ],
  "mise_results": [
    {
      "error": "planner state read failed"
    },
    {}
  ],
  "cycle_inputs": [
    {
      "runtime_repair_session_status": "completed"
    }
  ]
}
```

## Expected
```json
{
  "actions": {
    "repair_task_scheduled": true,
    "normal_task_scheduled": true
  },
  "state": {
    "runtime_repair_in_flight": false,
    "running": true
  },
  "transitions": [
    "paused",
    "running"
  ],
  "counts": {
    "spawn_calls": { "eq": 2 },
    "runtime_repair_spawn_calls": { "eq": 1 },
    "normal_spawn_calls": { "eq": 1 },
    "created_worktrees": { "eq": 2 }
  },
  "routing": {
    "runtime_repair_name": { "prefix": "repair-runtime-" },
    "runtime_repair_skill": { "equals": "debugging" }
  }
}
```

## Expected Error

```json
{
  "absent": true
}
```
