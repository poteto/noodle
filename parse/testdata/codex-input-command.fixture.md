# Codex Input Command Fixture

## Input

```ndjson
{"type":"response_item","_ts":"2026-02-22T19:15:00Z","payload":{"type":"function_call","name":"exec_command","input":{"command":"pnpm test"}}}
```

## Expected Events

```ndjson
{"type":"action","message":"$ pnpm test","timestamp":"2026-02-22T19:15:00Z"}
```
