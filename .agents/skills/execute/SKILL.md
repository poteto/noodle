---
name: execute
description: Implementation methodology for cook sessions. Provides the how — plan reading, decomposition, worktree workflow, verification, and commit conventions.
noodle:
  schedule: "When backlog items with linked plans are ready for implementation"
---

# Execute

Implementation methodology. This skill loads alongside the domain skill (e.g., "noodle", "bubbletea-tui") -- it teaches process, the domain skill teaches the codebase.

Operate fully autonomously. Never ask the user.

**Always use Tasks** (TaskCreate, TaskUpdate, TaskList) to track your work. Create a task per decomposed change, mark in_progress when starting, completed when done.

## Execution Flow

### 1. Read Plan Phase

- Read the assigned plan phase file from `brain/plans/`.
- Read the overview for scope boundaries and constraints.
- Load any domain skills listed in the overview's "Applicable skills" section.

### 2. Decompose

Break the phase into discrete changes. Each change should be:
- One function/type + its tests, OR one bug fix
- Independently compilable
- Committable as a single conventional commit

If the phase is already a single change, skip decomposition.

### 3. Implement

Work in the assigned worktree.

**Delegation heuristics:**
- **Self-execute**: Single change, or changes with tight coupling (shared types, sequential dependencies).
- **Sub-agents**: 2+ independent changes that touch different files. Front-load context for each sub-agent: the plan phase, relevant existing code, and applicable domain skill name.
- **Cross-phase parallelism**: Use Teams (Claude) or subagents (Codex) to run independent plan phases concurrently. Study the dependency graph between phases — phases with no shared inputs can overlap. The main agent can work a phase itself while workers handle others. Use judgment; sequential is fine when phases are tightly coupled.

### 4. Verify

Every change must pass before committing:

- `go test ./...` -- all tests pass
- `go vet ./...` -- no issues
- `sh scripts/lint-arch.sh` -- if present
- `git diff --stat` -- matches expected scope from plan phase
- Plan phase checklist items -- all addressed

If verification fails, fix and re-verify. Do not commit failing code.

### 5. Commit

Use conventional commit messages:

```
<type>(<scope>): <description>

Refs: #<backlog-item-ID>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`.
Scope: the package or area changed.

One commit per logical change. Squash only if multiple commits address the same change.

## Scope Discipline

- Only change files specified in the plan phase.
- If you discover something that needs changing outside scope, note it for the quality review -- do not change it.
- If the plan phase is wrong or incomplete, flag it in your output. Do not silently deviate.

## Principles

- [[prove-it-works]]
- [[trust-the-output-not-the-report]]
- [[cost-aware-delegation]]
- [[guard-the-context-window]]
- [[boundary-discipline]]
- [[outcome-oriented-execution]]
