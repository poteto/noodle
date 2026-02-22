# Tickets Materializer Fixture

## Sessions

```json
{
  "cook-a": [
    {
      "type": "ticket_claim",
      "session_id": "cook-a",
      "timestamp": "2026-02-22T21:00:00Z",
      "payload": {
        "target": "42",
        "target_type": "backlog_item",
        "files": ["src/auth/token.go"]
      }
    },
    {
      "type": "ticket_progress",
      "session_id": "cook-a",
      "timestamp": "2026-02-22T21:10:00Z",
      "payload": {
        "target": "42",
        "target_type": "backlog_item",
        "summary": "updating tests"
      }
    }
  ],
  "cook-b": [
    {
      "type": "ticket_claim",
      "session_id": "cook-b",
      "timestamp": "2026-02-22T20:10:00Z",
      "payload": {
        "target": "phase-03",
        "target_type": "plan_phase"
      }
    },
    {
      "type": "ticket_blocked",
      "session_id": "cook-b",
      "timestamp": "2026-02-22T21:05:00Z",
      "payload": {
        "target": "phase-03",
        "target_type": "plan_phase",
        "blocked_by": "42",
        "reason": "waiting on dependency"
      }
    }
  ],
  "cook-c": [
    {
      "type": "ticket_claim",
      "session_id": "cook-c",
      "timestamp": "2026-02-22T20:00:00Z",
      "payload": {
        "target": "src/legacy/file.go",
        "target_type": "file"
      }
    }
  ]
}
```

## Options

```json
{
  "now": "2026-02-22T21:20:00Z",
  "timeout": "30m",
  "active_sessions": ["cook-a", "cook-b", "cook-c"]
}
```

## Expected Tickets

```json
[
  {
    "target": "src/legacy/file.go",
    "target_type": "file",
    "cook_id": "cook-c",
    "claimed_at": "2026-02-22T20:00:00Z",
    "last_progress": "2026-02-22T20:00:00Z",
    "status": "stale"
  },
  {
    "target": "phase-03",
    "target_type": "plan_phase",
    "cook_id": "cook-b",
    "claimed_at": "2026-02-22T20:10:00Z",
    "last_progress": "2026-02-22T20:10:00Z",
    "status": "stale",
    "blocked_by": "42",
    "reason": "waiting on dependency"
  },
  {
    "target": "42",
    "target_type": "backlog_item",
    "cook_id": "cook-a",
    "files": ["src/auth/token.go"],
    "claimed_at": "2026-02-22T21:00:00Z",
    "last_progress": "2026-02-22T21:10:00Z",
    "status": "active"
  }
]
```
