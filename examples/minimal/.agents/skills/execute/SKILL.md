---
name: execute
description: Implements a backlog item. Reads the task prompt, makes changes, commits.
schedule: "When backlog items with linked plans are ready for implementation"
---

# Execute

Implement the task described in the prompt. Make the code changes, verify they work, and commit with a conventional commit message.

## Steps

1. Read the task description from the prompt.
2. Make the required changes.
3. Verify: run tests or checks relevant to the change.
4. Commit with a message in the format: `<type>(<scope>): <description>`.
