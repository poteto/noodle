# Stamp Processor Edge Fixture

## Input

```ndjson
{"type":"result","subtype":"success","is_error":true,"error":{"message":"Provider internal error","type":"api_error"},"timestamp":1730000000}
{"type":"response_item","payload":{"type":"function_call","name":"exec_command","input":{"command":"pnpm typecheck"}}}
```

## Expected Stamped

```ndjson
{"_ts":"2026-02-22T18:20:00Z","error":{"message":"Provider internal error","type":"api_error"},"is_error":true,"subtype":"success","timestamp":1730000000,"type":"result"}
{"_ts":"2026-02-22T18:20:00Z","payload":{"input":{"command":"pnpm typecheck"},"name":"exec_command","type":"function_call"},"type":"response_item"}
```

## Expected Events

```ndjson
{"provider":"claude","type":"error","message":"Provider internal error","timestamp":"2026-02-22T18:20:00Z"}
{"provider":"codex","type":"action","message":"$ pnpm typecheck","timestamp":"2026-02-22T18:20:00Z"}
```

## Expected Error

```json
```
