---
name: execute
description: Implementation methodology for executing tasks. Provides the how — scoping, decomposition, worktree workflow, verification, and commit conventions.
schedule: "When backlog items with linked plans are ready for implementation"
---

# Execute

Implementation methodology. Loads alongside the domain skill (e.g., "noodle") — this teaches process, the domain skill teaches the codebase.

Operate fully autonomously. Never ask the user. Don't stop until the work is fully complete.

**Track all work with Tasks** (TaskCreate, TaskUpdate, TaskList). One task per decomposed change; mark in_progress when starting, completed when done.

## Execution Flow

### 1. Scope

Establish what needs doing:

- **Plan phase**: Read the assigned phase from `brain/plans/`. Read the overview for scope boundaries. Invoke Skill(backlog) for project-specific context. Load domain skills listed in "Applicable skills."
- **Backlog item**: Read the todo from `brain/todos.md`. If a linked plan exists, read it. Otherwise, scope from the description.
- **Ad-hoc request**: The user prompt is the scope. Identify affected files and packages before starting.

Output: a clear, bounded description of what changes and what doesn't.

### 2. Decompose

Break scope into discrete changes. Each change:
- One function/type + its tests, OR one bug fix
- Independently compilable
- One conventional commit

Single-change scopes skip decomposition.

### 3. Implement

#### Worktree First — Non-Negotiable

**Never edit files on main.** Multiple sessions run concurrently; editing main causes merge conflicts and lost work.

If CWD is already inside `.worktrees/`, use it. Otherwise: `noodle worktree create <descriptive-name>`

Use absolute paths or `noodle worktree exec <name> <cmd>`. **Never `cd` into a worktree** — if it gets removed while the shell is inside, the session dies permanently.

Commit inside the worktree. When done: `noodle worktree merge <name>`

Skip only when the user is interactively driving a single-agent session and explicitly chooses main.

#### Delegation

- **Self-execute**: Single change, or tightly coupled changes (shared types, sequential dependencies).
- **Sub-agents**: 2+ independent changes touching different files. Front-load context: scope, relevant code, domain skill name.
- **Team execution**: 2+ parallelizable phases in a plan. See below.
- **Codex**: Mechanical work not requiring judgment (renames, boilerplate, repetitive edits). Never for architectural decisions.

Sequential is fine when phases are tightly coupled. Before parallelizing, ask: "Does any phase's output become another phase's input?" If one phase defines a type/interface that another consumes, they share an API contract and must be sequential — otherwise each invents its own version. Only phases with no type-level coupling can overlap.

#### Team Execution

The lead orchestrates — it does NOT implement. Research via sub-agents, delegate all implementation to teammates.

1. **Lead worktree**: Use current worktree if already in one, otherwise `noodle worktree create plan-<N>-lead`
2. **Team**: `TeamCreate` — all tasks go through this team's task list
3. **Per-teammate worktrees**: `noodle worktree create plan-<N>-phase-<M>`
4. **Spawn teammates**: `Task` with `mode: "bypassPermissions"`, `team_name`, worktree path, scope, and domain skill name. Always spawn fresh agents to keep context clean.
5. **Teammates commit** on their own branches
6. **Review before merging**: Spawn a review agent to check each teammate's work against the plan before merging
7. **Merge teammates into lead** (not main):
   ```bash
   git -C .worktrees/plan-<N>-lead merge <teammate-branch>
   noodle worktree cleanup plan-<N>-phase-<M>
   ```
8. **Verify integrated result** in lead worktree (see Verify below)
9. **Merge lead to main**: `noodle worktree merge plan-<N>-lead`

Foundational phases that later phases depend on: execute first, commit in lead worktree, then create teammate worktrees from that point.

### 4. Verify

Every change must pass before committing. Fix and re-verify on failure. Never commit failing code.

**Full test suite** — after each change, run the complete suite:
- `pnpm build` — compiles UI and Go binary; catches type errors in both
- `go test ./...` (or scoped to changed packages)
- `go vet ./...`
- `pnpm --filter noodle-ui test` — UI unit tests (vitest)
- `sh scripts/lint-arch.sh` — if present

Or equivalently: `pnpm check` (runs build, Go tests, vet, arch lint, and fixture tests).

**E2E smoke test** — after integrating changes, before merging to main:
- `pnpm test:smoke` — Go e2e tests with `-tags e2e`
- In a worktree: `noodle worktree exec <name> pnpm test:smoke`
- When UI changes: write NEW test cases covering the changed interface

**Fixture tests** — when changes affect loop behavior or runtime state:
- `pnpm fixtures:loop` / `pnpm fixtures:hash`
- Update fixtures: `pnpm fixtures:loop:record` then `pnpm fixtures:hash:sync`

**Visual verification** — when changes affect UI:
- Use the Chrome tool to open the UI in browser, click through affected flows

**Scope check:**
- `git diff --stat` — matches expected scope
- All checklist items addressed (plan phase, todo criteria, or ad-hoc requirements)

### 5. Commit

```
<type>(<scope>): <description>

Refs: #<issue-ID>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`. Scope: package or area changed. Refs: include when linked issue exists; omit for ad-hoc work.

One commit per logical change.

### 6. Yield

After all changes are committed and verified, emit `stage_yield` to signal the deliverable is complete:

```bash
noodle event emit --session $NOODLE_SESSION_ID stage_yield --payload '{"message": "Implemented: <brief summary>"}'
```

This tells the Noodle backend the stage's work is done, even if the agent process hasn't exited yet. Without this, the stage only completes on clean process exit — if the agent is interrupted after committing, the work is lost to the pipeline.

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
