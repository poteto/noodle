---
schema_version: 1
expected_failure: false
bug: false
regression: provider-command-codex
source_hash: ae288ecbd41a386c567c9567b8c2bbf3aac8a52d4fa6186d328179dacfd2c45b
---

## Expected

```json
{
  "contains": [
    "'codex' 'exec'",
    "'--skip-git-repo-check'",
    "'--full-auto'",
    "'--sandbox' 'workspace-write'",
    "'--json'",
    "'--model' 'gpt-5.3-codex'",
    "< '/tmp/prompt.txt'"
  ],
  "omits": [
    "2>&1"
  ]
}
```
