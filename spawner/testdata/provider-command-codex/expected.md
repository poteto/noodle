---
schema_version: 1
expected_failure: false
bug: provider-command-codex
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

