# Codex Tool Output Error Fixture

## Input

```ndjson
{"type":"response_item","timestamp":"2026-02-22T19:25:00Z","payload":{"type":"function_call_output","output":"Tool error: permission denied"}}
```

## Expected Events

```ndjson
{"type":"error","message":"Tool error: permission denied","timestamp":"2026-02-22T19:25:00Z"}
```
