---
schema_version: 1
expected_failure: false
bug: false
source_hash: 769878f4ff03b35dbc16673663798ef95f20c400f69354cd282a45ef1fa530a0
---

## Expected Stamped

```json
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
