# Canonical Claims Fixture

## Input
```ndjson
{"provider":"claude","type":"init","message":"session started","timestamp":"2026-02-22T15:00:00Z"}
{"provider":"claude","type":"action","message":"Implement monitor package","timestamp":"2026-02-22T15:01:00Z"}
{"provider":"claude","type":"result","message":"turn complete","timestamp":"2026-02-22T15:02:00Z","cost_usd":0.12,"tokens_in":100,"tokens_out":50}
{"provider":"claude","type":"error","message":"provider timeout","timestamp":"2026-02-22T15:03:00Z"}
```

## Expected Claims
```json
{
  "session_id": "cook-a",
  "has_events": true,
  "provider": "claude",
  "total_cost_usd": 0.12,
  "first_event_at": "2026-02-22T15:00:00Z",
  "last_event_at": "2026-02-22T15:03:00Z",
  "last_action": "Implement monitor package",
  "tokens_in": 100,
  "tokens_out": 50,
  "failed": true
}
```
