---
name: execute
description: >-
  Implementation methodology for executing tasks. Handles scoping, decomposition, worktree workflow,
  verification, and commit conventions. Triggers: "execute", "implement", "build this", "code this".
schedule: "When backlog items with linked plans are ready for implementation"
---

# Execute

Implementation methodology. Loads alongside the domain skill (e.g., "noodle") — this teaches process, the domain skill teaches the codebase. Operate fully autonomously. Never ask the user. Don't stop until the work is fully complete.

**Track all work with Tasks** (TaskCreate, TaskUpdate, TaskList). One task per decomposed change; mark in_progress when starting, completed when done.

## 1. Scope

Establish what needs doing:

- **Plan phase**: Read the assigned phase from `brain/plans/`. Read the overview for scope boundaries. Invoke Skill(backlog) for project-specific context. Load domain skills listed in "Applicable skills."
- **Backlog item**: Read the todo from `brain/todos.md`. If a linked plan exists, read it. Otherwise, scope from the description.
- **Ad-hoc request**: The user prompt is the scope. Identify affected files and packages before starting.

Output: a clear, bounded description of what changes and what doesn't.

## 2. Decompose

Break scope into discrete changes. Each change:
- One function/type + its tests, OR one bug fix
- Independently compilable
- One conventional commit

Single-change scopes skip decomposition.

## 3. Implement

### Worktree First — Non-Negotiable

**Never edit files on main.** Multiple sessions run concurrently; editing main causes merge conflicts and lost work.

If CWD is already inside `.worktrees/`, use it. Otherwise: `noodle worktree create <descriptive-name>`

Use absolute paths or `noodle worktree exec <name> <cmd>`. **Never `cd` into a worktree** — if it gets removed while the shell is inside, the session dies permanently.

Commit inside the worktree. When done: `noodle worktree merge <name>`

Skip only when the user is interactively driving a single-agent session and explicitly chooses main.

### Delegation

- **Self-execute**: Single change, or tightly coupled changes (shared types, sequential dependencies).
- **Sub-agents**: 2+ independent changes touching different files. Front-load context: scope, relevant code, domain skill name.
- **Team execution**: 2+ parallelizable phases in a plan. See `references/team-execution.md`.
- **Codex**: Mechanical work not requiring judgment (renames, boilerplate, repetitive edits). Never for architectural decisions.

Sequential is fine when phases are tightly coupled. Before parallelizing, ask: "Does any phase's output become another phase's input?" Phases with shared type-level contracts must be sequential.

## 4. Verify

Run the full verification suite before committing. See `references/verification.md` for the complete checklist. Never commit failing code — fix and re-verify on failure.

## 5. Commit

`<type>(<scope>): <description>` with optional `Refs: #<issue-ID>` footer. Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`. One commit per logical change.

## 6. Yield

After all changes are committed and verified, emit `stage_yield`:

```bash
noodle event emit --session $NOODLE_SESSION_ID stage_yield --payload '{"message": "Implemented: <brief summary>"}'
```

Without this, the stage only completes on clean process exit — interrupted work is lost to the pipeline.

## Scope Discipline

- Only change what's in scope. No defensive code, backwards-compat shims, or speculative features.
- Out-of-scope discoveries go in the quality review notes — don't change them.
- Wrong or incomplete plan/requirements: flag it in output, don't silently deviate.

## Principles

Read at runtime from `brain/principles/`:
- [[prove-it-works]]
- [[subtract-before-you-add]]
- [[cost-aware-delegation]]
- [[guard-the-context-window]]
- [[boundary-discipline]]
- [[outcome-oriented-execution]]
