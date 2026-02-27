---
name: execute
description: Implementation methodology for executing tasks. Provides the how — scoping, decomposition, worktree workflow, verification, and commit conventions.
noodle:
  domain_skill: backlog
  schedule: "When backlog items with linked plans are ready for implementation"
---

# Execute

Implementation methodology. This skill loads alongside the domain skill (e.g., "noodle") — it teaches process, the domain skill teaches the codebase.

Operate fully autonomously. Never ask the user.

**Always use Tasks** (TaskCreate, TaskUpdate, TaskList) to track your work. Create a task per decomposed change, mark in_progress when starting, completed when done.

## Execution Flow

### 1. Scope

Establish what needs doing. Sources vary:

- **Plan phase**: Read the assigned phase file from `brain/plans/`. Read the overview for scope boundaries. Load domain skills listed in "Applicable skills."
- **Backlog item**: Read the todo from `brain/todos.md`. If a linked plan exists, read it. Otherwise, scope directly from the item description.
- **Ad-hoc request**: The user prompt is the scope. Identify affected files and packages before starting.

Output of this step: a clear, bounded description of what changes and what doesn't.

### 2. Decompose

Break the scope into discrete changes. Each change should be:
- One function/type + its tests, OR one bug fix
- Independently compilable
- Committable as a single conventional commit

If the scope is already a single change, skip decomposition.

### 3. Implement

#### Worktree First — Non-Negotiable

**Work inside a linked worktree — never edit files on main.** Multiple sessions run concurrently; editing main causes merge conflicts and lost work.

Check first: if the CWD is already inside `.worktrees/`, you're in one — use it. Otherwise, create one:

```bash
noodle worktree create <descriptive-name>
```

Then use absolute paths or `noodle worktree exec <name> <cmd>` for all operations. **Never `cd` into a worktree** — if it gets removed while the shell is inside, the session dies permanently.

Commit inside the worktree. When done, merge back:

```bash
noodle worktree merge <name>
```

Skip the worktree only when the user is interactively driving a single-agent session and explicitly chooses to work on main.

#### Delegation

**Delegation heuristics:**
- **Self-execute**: Single change, or changes with tight coupling (shared types, sequential dependencies).
- **Sub-agents**: 2+ independent changes that touch different files. Front-load context for each sub-agent: the scope, relevant existing code, and applicable domain skill name.
- **Team execution**: 2+ parallelizable phases in a plan. See workflow below.

Sequential is fine when phases are tightly coupled. Study the dependency graph — phases with no shared inputs can overlap.

#### Team Execution

For plans with parallelizable phases, the lead orchestrates from its own worktree:

1. **Lead worktree**: Use current worktree if already in one, otherwise `noodle worktree create plan-<N>-lead`
2. **Team**: `TeamCreate` — all tasks go through this team's task list
3. **Per-teammate worktrees**: `noodle worktree create plan-<N>-phase-<M>` — one per teammate
4. **Spawn teammates**: `Task` with `mode: "bypassPermissions"`, `team_name`, worktree path, scope, and domain skill name. Front-load context to avoid rediscovery.
5. **Teammates commit** on their own branches inside their worktrees
6. **Merge teammates into lead** (not main):
   ```bash
   git -C .worktrees/plan-<N>-lead merge <teammate-branch>
   noodle worktree cleanup plan-<N>-phase-<M>
   ```
7. **Verify integrated result** in lead worktree (see Verify section below)
8. **Merge lead to main**: `noodle worktree merge plan-<N>-lead`

Foundational phases that later phases depend on: execute first, commit in lead worktree, then create teammate worktrees from that point.

### 4. Verify

Every change must pass before committing. If verification fails, fix and re-verify. Do not commit failing code.

**Unit & static analysis** — after each change:
- `go test ./...` (or scoped to changed packages)
- `go vet ./...`
- `sh scripts/lint-arch.sh` — if present

**E2E smoke tests** — after integrating changes, especially before merging to main:
- `pnpm test:smoke` — end-to-end suite, catches integration regressions
- In a worktree: `noodle worktree exec <name> pnpm test:smoke`

**Fixture tests** — when changes affect loop behavior or runtime state:
- `pnpm fixtures:loop` — verify runtime dumps match expectations
- `pnpm fixtures:hash` — verify source hashes are current
- If fixtures need updating: `pnpm fixtures:loop:record` then `pnpm fixtures:hash:sync`

**Scope check:**
- `git diff --stat` — matches expected scope
- Scope checklist items — all addressed (plan phase checklist, todo acceptance criteria, or ad-hoc requirements)

### 5. Commit

Use conventional commit messages:

```
<type>(<scope>): <description>

Refs: #<issue-ID>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`.
Scope: the package or area changed.
Refs line: include when there's a linked issue or backlog item. Omit for ad-hoc work with no tracked item.

One commit per logical change. Squash only if multiple commits address the same change.

## Scope Discipline

- Only change what's in scope. For plan-based work, that means files specified in the phase. For other work, that means what's necessary to satisfy the request.
- If you discover something that needs changing outside scope, note it for the quality review — do not change it.
- If the plan or requirements are wrong or incomplete, flag it in your output. Do not silently deviate.

## Principles

- [[prove-it-works]]
- [[subtract-before-you-add]]
- [[cost-aware-delegation]]
- [[guard-the-context-window]]
- [[boundary-discipline]]
- [[outcome-oriented-execution]]
