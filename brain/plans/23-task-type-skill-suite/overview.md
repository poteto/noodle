---
id: 23
created: 2026-02-22
status: draft
---

# Task-Type Skill Suite + First-Class Planning

Combines todo #23 (task-type skills) and todo #22 (opinionated planning).

## Context

Noodle is a light framework with strong opinions and few primitives. Like React — where everything is a component — in Noodle, everything is a file. Skills, plans, state, brain notes — all files that agents read directly.

Task types are Noodle's equivalent of built-in components. Today they're hardcoded in a Go registry. This plan makes them dynamic: any skill with `noodle:` frontmatter in its SKILL.md becomes a task type. Built-in task types (prioritize, execute, quality, etc.) follow the same convention as user-defined ones. Users extend Noodle by adding `noodle:` frontmatter to their skill — no Go code changes, no extra files.

Several built-in skills are missing (quality, oops, debate), several exist but lack principle grounding (prioritize, reflect, meditate), and 5 old role-based skills (CEO, CTO, Director, Manager, Operator) contain patterns worth extracting before deletion.

Planning is currently a user-configurable adapter — it should be native. Plans always live in `brain/plans/` with a Noodle-owned format. The user creates plans outside Noodle using their own agent and the plan skill. Noodle only executes work that has a plan.

## Scope

**In scope:**
- Dynamic task type registry via `noodle:` frontmatter convention (replaces hardcoded Go registry)
- Create or rewrite 8 skills in `.agents/skills/` — 7 task-type skills + 1 utility skill (debugging)
- Extract patterns from old role-based skills (CEO, CTO, Director, Manager, Operator)
- Ground each skill in engineering principles from `brain/principles/`
- Native planning: remove plan adapter, minimal Go reader, `noodle plan` CLI commands
- Context injection: Noodle preamble in cook sessions + skill-specific schema `references/`
- Multi-skill loading in dispatcher (methodology skill alongside domain skill)
- Plans as precondition: prioritize skill only schedules items with linked plans
- Quality verdict ingestion in mise builder
- Rename `spawner` → `dispatcher` throughout Go code
- Dispatcher with command-template runtime (no interface, no enum — just a command)
- `runtime` field in frontmatter for extensible dispatch (command template, not enum)
- Fix stale CLI references, delete old role-based skills

**Out of scope:**
- Remote dispatcher backend — the `runtime` template is launch-only (controls what runs inside tmux). Full remote lifecycle (non-tmux monitoring, stopping) is a future plan.
- Review skill — the Chef (human) does review via the TUI
- Verify skill — the execute agent verifies its own work
- Backlog adapter changes — backlogs stay configurable
- Bootstrap skill updates
- Interactive-only skills (commit, codex, skill-creator, etc.)

## Constraints

- **Lean core, smart skills.** Go core is thin orchestration: process lifecycle, concurrency, file I/O, data assembly. All scheduling intelligence, quality judgment, and task semantics live in skills. The core surfaces data; skills make decisions.
- **Everything is a file.** Skills, plans, brain notes, `.noodle/` state — all files the agent reads directly. This makes agents powerful and the tool extensible.
- **Context injection bridges core and skills.** Two layers: (1) a Noodle context preamble injected by the dispatcher into every cook session — a lean map of `.noodle/` state files and their purpose, and (2) skill-specific schemas in each skill's `references/` directory. The preamble says "here's what exists"; the references say "here's how to use it."
- **Dispatcher, not spawner.** The component that launches sessions is the "dispatcher." It resolves a `runtime` command template (from skill frontmatter or `noodle.toml` default) and executes it. Today the default is tmux. Users can override per-skill to SSH, a cloud CLI, or anything — no Go code changes. Same pattern as the backlog adapter.
- **Plans are the user's responsibility.** Noodle never auto-plans. The user creates plans outside Noodle (using their agent + the plan skill). The prioritize skill only schedules execution for items with a linked plan. Unplanned items surface as "action needed" in the TUI.
- **Autonomy via spawn flags.** Noodle passes flags to disable interactive prompts. Skills don't need to manage this themselves.
- **Subtract before adding.** Delete code, tests, and schemas that no longer serve a purpose. Redesign from first principles.
- **No backwards compatibility.** Pre-launch project. Remove schema versioning, compatibility shims, and any complexity that only exists for migration.
- All skills live in `.agents/skills/`. No stubs directory.
- Skills should be lean — guard the context window.
- Use the `skill-creator` skill when writing each skill.

### Execution contract

Each phase must follow this workflow:
1. Work in a worktree
2. Minimum one commit per phase with conventional messages
3. `make ci` must pass before merging
4. Review changes: verify scope boundaries, check for regressions, flag issues by severity (high/medium/low)
5. Rebase on main before merging (check if main has advanced after every commit)
6. Merge to main at the end of the phase

## User-Defined Task Types

Task types are the core primitive. A task type is a skill whose SKILL.md has `noodle:` frontmatter:

```
.agents/skills/deploy/
├── SKILL.md          # Everything: agent instructions + noodle metadata in frontmatter
└── references/       # Schema docs (loaded into context alongside SKILL.md body)
```

The `noodle:` frontmatter block contains what the Go core needs for scheduling and lifecycle. The skill resolver strips all frontmatter before injecting SKILL.md into agent context — the agent never sees it.

```yaml
---
description: Deploy to production after verified changes land on main
noodle:
  blocking: false
  schedule: "After successful execute on main branch"
  runtime: "ssh deploy-host 'cd {{repo}} && claude -p < {{prompt}}'"
---
```

| Field | Purpose | Default |
|-------|---------|---------|
| `blocking` | Prevents other cooks from running in parallel (enforced by Go core) | `false` |
| `schedule` | One-line guidance for the prioritize skill — when to schedule this task | required |
| `runtime` | Command template for what runs inside the tmux session | project default from `noodle.toml` |

The presence of `noodle:` is what makes a skill a task type. No `task: true` needed — if it has `noodle:`, it's schedulable. Skills without `noodle:` are utility skills.

The `schedule` field goes into the mise brief so the prioritize skill knows when to schedule this task without reading every skill's full SKILL.md. Quality review is not a per-task flag — the quality skill has its own `schedule` field, and the prioritize skill decides when to run it based on session history.

The `runtime` field is a command template that controls what command runs inside the tmux session. When empty (the default), the built-in provider command runs (claude/codex CLI). When set, the dispatcher replaces the provider command with the resolved template — the tmux session, monitoring, and lifecycle management stay the same. The dispatcher always appends `| noodle stamp ...` piping to the command (whether built-in or custom) — runtime templates must NOT include stamp piping.

```yaml
# Default (empty) — built-in provider command:
#   claude -p --output-format stream-json ... | noodle stamp ...

# Custom — run a different command inside the tmux session:
runtime: "ssh gpu-box 'cd {{repo}} && claude -p < {{prompt}}'"
```

Template variables: `{{session}}`, `{{repo}}`, `{{prompt}}`, `{{skill}}`, `{{brief}}`. The dispatcher resolves them before execution.

A project-wide default can be set in `noodle.toml`:

```toml
[runtime]
default = ""  # empty = built-in provider command
```

**Discovery:** The skill resolver parses SKILL.md frontmatter during startup. Skills with a `noodle:` block are registered as task types. Skills without it are utility skills (invoked by other skills at runtime, never scheduled).

**Composition:** Tasks compose dynamically at runtime — the agent reads its skill instructions, which can reference other skills by name. No framework-level composition mechanism. The one special case is execute, where the dispatcher loads both the execute methodology skill and the adapter-configured domain skill (session assembly, not composition). Beyond that:
1. **Scheduling** — the prioritize skill sequences tasks based on session history and schedules
2. **Plans** — multi-phase workflows are plans; each phase is an execute task
3. **Output-driven** — tasks produce state changes (files), the prioritize skill reacts next cycle

## Old Skill Extraction Map

| Source | Pattern | Target |
|--------|---------|--------|
| CEO | Foundation-before-feature ordering | prioritize |
| CEO | Cheapest execution mode that finishes safely | prioritize |
| CEO | Explicit rationale for every scheduling decision | prioritize |
| CEO | Fresh context each cycle — no long-lived drift | prioritize |
| CEO | Work around blockers, don't idle | prioritize |
| CTO | Evidence-first quality governance | quality |
| CTO | Lint-arch as first-class audit evidence | quality |
| CTO | Principle-anchored findings with citations | quality |
| CTO | Advocate, don't block — preserve momentum | quality |
| Manager | "Claims are promises, not proof" — verify artifacts | quality, execute |
| Manager | git diff --stat ALL files, not just claimed | quality, execute |
| Operator | Decompose → Implement → Verify → Commit flow | execute, oops |
| Operator | Worktree isolation, lint-before-commit | execute, oops |
| Operator | Brain update on fix — capture novel gotchas | debugging |
| Manager | Parallel by default — sub-agents for independent work | execute |
| Manager | Minimal-context workers — front-load context | execute |

## Principle-to-Skill Mapping

| Principle | Skills |
|-----------|--------|
| verify-runtime | quality, debugging, execute |
| trust-the-output-not-the-report | quality, execute |
| fix-root-causes | debugging, oops |
| observe-directly | debugging, oops |
| suspect-state-before-code | debugging, oops |
| encode-lessons-in-structure | reflect, meditate |
| cost-aware-delegation | prioritize, execute |
| foundational-thinking | prioritize |
| guard-the-context-window | all |
| never-block-on-the-human | all |
| subtract-before-you-add | meditate, prioritize |
| exhaust-the-design-space | debate |
| boundary-discipline | execute |
| outcome-oriented-execution | quality, execute, debate |
| redesign-from-first-principles | meditate |

## Phases

### Infrastructure

1. [[plans/23-task-type-skill-suite/phase-01-dynamic-task-registry]] — Replace hardcoded Go registry with frontmatter-based discovery
2. [[plans/23-task-type-skill-suite/phase-02-native-planning]] — Remove plan adapter, minimal Go reader + CLI commands
3. [[plans/23-task-type-skill-suite/phase-03-context-injection]] — Rename spawner→dispatcher, runtime command dispatch, context preamble

### Task-type skills

4. [[plans/23-task-type-skill-suite/phase-04-prioritize]] — Queue scheduler with plans-as-precondition and schedule reading
5. [[plans/23-task-type-skill-suite/phase-05-quality]] — Post-cook quality gate with verdict files
6. [[plans/23-task-type-skill-suite/phase-06-execute]] — Implementation methodology (worktrees, delegation, verification)
7. [[plans/23-task-type-skill-suite/phase-07-reflect]] — Learning capture from mistakes and corrections
8. [[plans/23-task-type-skill-suite/phase-08-meditate]] — Brain cleanup and principle extraction
9. [[plans/23-task-type-skill-suite/phase-09-oops]] — Infrastructure fix (user-project and Noodle-internal)
10. [[plans/23-task-type-skill-suite/phase-10-debate]] — Structured debate with per-task state

### Skills + cleanup

11. [[phase-11-utility-skills]] — Debugging amendments + plan skill update
12. [[plans/23-task-type-skill-suite/phase-12-cleanup]] — Stale references, delete old skills

## Verification

- `make ci` passes (test, vet, lintarch, fixtures)
- Task types discovered from `noodle:` frontmatter: `go test ./skill/...` covers discovery
- Each skill SKILL.md has: frontmatter, purpose, principles, contract, process, verification
- Old role-based skills (CEO, CTO, Director, Manager, Operator) are deleted
- No remaining references to `sous-chef` in Go code or config
- Verify, plan, review, and cook task types removed from registry and loop code
- `noodle plan create/done/phase-add` commands work
- No `[adapters.plans]` in config
- Mise brief includes plan metadata and quality verdict history
- Plan skill uses native commands and backlog adapter for link-back
- Execute sessions include both methodology and domain skills (session assembly)
- No remaining references to `spawner` or `Spawner` in Go code
- Dispatcher resolves `runtime` command templates and executes them
- Default runtime from `noodle.toml` used when skill doesn't override
- Custom `runtime` in frontmatter is executed as a command
- Noodle context preamble injected into all cook sessions
- Skills with `.noodle/` interaction include schema docs in `references/`
- Prioritize skill skips unplanned backlog items
- TUI shows "action needed" for unplanned items
- No stale CLI command references
