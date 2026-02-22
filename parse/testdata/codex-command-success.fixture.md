# Codex Command Success Fixture

## Input

```ndjson
{"type":"item.completed","timestamp":"2026-02-22T19:20:00Z","item":{"type":"command_execution","command":"npm run lint","exit_code":0}}
```

## Expected Events

```ndjson
{"type":"action","message":"$ npm run lint","timestamp":"2026-02-22T19:20:00Z"}
```

## Expected Error

```json
```
