# Loop Fixture: Oops Fallback Uses Custom Routing Defaults

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
  ],
  "phases": {
    "debugging": "",
    "oops": "oops"
  },
  "routing_provider": "codex",
  "routing_model": "gpt-5.3-codex"
}
```

## Expected
```json
{
  "actions": {
    "repair_task_scheduled": true,
    "oops_task_scheduled": true,
    "normal_task_scheduled": false
  },
  "state": {
    "runtime_repair_in_flight": true,
    "paused": true
  },
  "transitions": [
    "paused"
  ],
  "counts": {
    "spawn_calls": { "eq": 1 },
    "runtime_repair_spawn_calls": { "eq": 1 },
    "normal_spawn_calls": { "eq": 0 },
    "created_worktrees": { "eq": 1 }
  },
  "routing": {
    "runtime_repair_skill": { "equals": "oops" },
    "runtime_repair_provider": { "equals": "codex" },
    "runtime_repair_model": { "equals": "gpt-5.3-codex" },
    "runtime_repair_name": { "prefix": "repair-runtime-" }
  }
}
```

## Expected Error

```json
{
  "absent": true
}
```
