---
name: execute
description: Implements a backlog item. Reads the task prompt, makes changes, commits.
schedule: "When backlog items with linked plans are ready for implementation"
---

# Execute

Implement the task described in the prompt. Make the code changes, verify they compile, and commit with a conventional commit message.

## Steps

1. Read the task description from the prompt.
2. Make the required changes.
3. Verify the changes compile and pass basic checks.
4. Commit with a message in the format: `<type>(<scope>): <description>`.
