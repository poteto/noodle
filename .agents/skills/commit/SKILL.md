---
name: commit
description: >
  Create conventional commit messages. Use when the user says "commit this", "save changes",
  "git commit", asks to commit code, write a commit message, or format git history.
  Follows conventional commits specification.
---

# Conventional Commits

Follow these project-specific rules when creating commits.

## Branching

**Never create bare branches.** If you need branch isolation, use the `worktree` skill. For simple changes, commit directly on the current branch (including main). Do not run `git checkout -b`.

## Project Rules

- **One commit per logical change** — never combine unrelated fixes or features. Three independent changes = three commits.
- **Only commit files you changed** — never pull in unrelated files from concurrent sessions.
- **Always include a body** — no title-only commits. Explain what and why.
- **`merge` type** — use `merge` (not a merge commit) when merging a worktree or feature branch.
- **Sequence commits** to maximise option value — foundational infrastructure first, then features that build on it.

## Example

```
feat(alerts): Add Slack thread replies for alert updates

When an alert is updated or resolved, post a reply to the original
Slack thread instead of creating a new message. This keeps related
notifications grouped together.
```

## References

- [Conventional Commits Specification](https://www.conventionalcommits.org/en/v1.0.0/#specification)
