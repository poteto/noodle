---
name: commit
description: Create conventional commit messages following best conventions. Use when committing code changes, writing commit messages, or formatting git history. Follows conventional commits specification.
---

# Conventional Commit Messages

Follow these conventions when creating commits.

## Branching

**Never create bare branches.** If you need branch isolation, use the `worktree` skill. For simple changes, commit directly on the current branch (including main). Do not run `git checkout -b`.

## Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

The header is required. Scope is optional. All lines must stay under 100 characters.

## Commit Types

| Type | Purpose |
|------|---------|
| `build` | Build system or CI changes |
| `chore` | Routine maintenance tasks |
| `ci` | Continuous integration configuration |
| `deps` | Dependency updates |
| `docs` | Documentation changes |
| `feat` | New feature |
| `fix` | Bug fix |
| `merge` | Merge a worktree or feature branch |
| `perf` | Performance improvement |
| `refactor` | Code refactoring (no behavior change) |
| `revert` | Revert a previous commit |
| `style` | Code style and formatting |
| `test` | Tests added, updated or improved |

## Subject Line Rules

- Use imperative, present tense: "Add feature" not "Added feature"
- Capitalize the first letter
- No period at the end
- Maximum 70 characters

## Body Guidelines

- Do not write title-only commits; include a meaningful body.
- Explain **what** and **why**, not how
- Use imperative mood and present tense
- Include motivation for the change
- Contrast with previous behavior when relevant


## Examples

### Simple fix

```
fix(api): Handle null response in user endpoint

The user API could return null for deleted accounts, causing a crash
in the dashboard. Add null check before accessing user properties.
```

### Feature with scope

```
feat(alerts): Add Slack thread replies for alert updates

When an alert is updated or resolved, post a reply to the original
Slack thread instead of creating a new message. This keeps related
notifications grouped together.
```

### Refactor

```
refactor: Extract common validation logic to shared module

Move duplicate validation code from three endpoints into a shared
validator class. No behavior change.
```

### Breaking change

```
feat(api)!: Remove deprecated v1 endpoints

Remove all v1 API endpoints that were deprecated in version 23.1.
Clients should migrate to v2 endpoints.

BREAKING CHANGE: v1 endpoints no longer available
```

## Revert Format

```
revert: feat(api): Add new endpoint

This reverts commit abc123def456.

Reason: Caused performance regression in production.
```

## Principles

- **One commit per logical change** — never combine unrelated fixes or features into a single commit. If you made three independent changes, create three commits.
- Commits should be independently reviewable
- The repository should be in a working state after each commit
- Sequence commits to maximise option value — foundational infrastructure first, then features that build on it
- Only commit files you actually changed — never pull in unrelated files from concurrent sessions

## References

- [Conventional Commits Specification](https://www.conventionalcommits.org/en/v1.0.0/#specification)
