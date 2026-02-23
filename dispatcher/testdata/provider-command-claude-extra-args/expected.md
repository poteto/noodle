---
schema_version: 1
expected_failure: false
bug: false
source_hash: cb27ed00dc71c2c101cb8754f303cfb0ff47f558a5596221a363b5e54f0289e1
---

## Expected

```json
{
  "contains": [
    "'claude'",
    "'--output-format' 'stream-json'",
    "'--permission-mode' 'bypassPermissions'",
    "'--model' 'claude-sonnet-4-6'",
    "'--append-system-prompt' 'skill-system'",
    "'--dangerously-skip-permissions'",
    "'--verbose'",
    "< '/tmp/prompt.txt'",
    "2>&1"
  ],
  "omits": []
}
```
