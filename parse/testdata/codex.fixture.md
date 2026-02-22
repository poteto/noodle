# Codex Parse Fixture

## Input

```ndjson
{"type":"session_meta","timestamp":"2026-02-22T18:10:00Z","payload":{"id":"sess-1"}}
{"type":"response_item","timestamp":"2026-02-22T18:10:01Z","payload":{"type":"function_call","name":"exec_command","arguments":"{\"command\":\"npm test\"}"}}
{"type":"response_item","timestamp":"2026-02-22T18:10:02Z","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"I will now update tests."}]}}
{"type":"event_msg","timestamp":"2026-02-22T18:10:03Z","payload":{"type":"task_complete","message":"done","cost":0.8,"tokens_in":210,"tokens_out":55}}
{"type":"item.completed","timestamp":"2026-02-22T18:10:04Z","item":{"type":"command_execution","command":"npm run lint","exit_code":2}}
```

## Expected Events

```ndjson
{"type":"init","message":"codex session started","timestamp":"2026-02-22T18:10:00Z"}
{"type":"action","message":"$ npm test","timestamp":"2026-02-22T18:10:01Z"}
{"type":"action","message":"text:I will now update tests.","timestamp":"2026-02-22T18:10:02Z"}
{"type":"complete","message":"done","timestamp":"2026-02-22T18:10:03Z","cost_usd":0.8,"tokens_in":210,"tokens_out":55}
{"type":"error","message":"command failed: npm run lint","timestamp":"2026-02-22T18:10:04Z"}
```

## Expected Error

```json
```
