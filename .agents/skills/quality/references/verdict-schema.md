# Verdict Schema

Written by the quality skill to `.noodle/quality/<session-id>.json`.

```json
{
  "accept": "boolean — true if work passes all checks",
  "feedback": "string — overall assessment summary",
  "issues": [
    {
      "description": "string — what is wrong",
      "severity": "string — high|medium|low",
      "principle": "string — which principle was violated (if applicable)"
    }
  ],
  "scope_violations": [
    "string — file paths changed outside the plan phase scope"
  ],
  "todos_created": [
    "string — backlog item IDs for non-blocking issues filed"
  ]
}
```

## Severity Guide

- **high**: Blocks acceptance. Incorrect behavior, missing tests for new behavior, scope violation on core files, principle violation that changes architecture.
- **medium**: Worth fixing but not blocking on its own. Multiple mediums may trigger rejection.
- **low**: Style, documentation, minor improvements. Filed as backlog items, never blocks.
