---
id: 1
created: 2026-02-20
updated: 2026-02-21
status: active
---

# Noodle Open-Source Architecture

## Context

Noodle is becoming an open-source framework for AI coding. It can be dropped into an existing project or used to scaffold a new one with Noodle as its central operating system. Today, Noodle is deeply coupled to the parent project: hardcoded path assumptions, project-specific skill ownership split across `noodle/skills/` and `.agents/skills/`, embedded templates that aren't extensible, a complex role hierarchy (Director/Manager/Operator/Apprentice) with dedicated agent definitions, a deterministic Go assessor that tries to encode scheduling judgment in weighted heuristics, and a config model that doesn't expose the right extension points. The codebase is over-built for the simpler model we now want — the Runner alone spans 88 methods across 20 files.

This plan redesigns Noodle from first principles for open-source adoption. The governing insight: **skills are the only extension point**. Users extend Noodle by writing or overriding skills — pure Claude Code skills with no Noodle-specific metadata. All wiring lives in a single config file. The actor model is radically simplified. The Go binary becomes lean infrastructure: gather state, call an LLM for prioritization, spawn cooks, monitor, log.

## Kitchen Brigade

All actor roles adopt kitchen terminology:

| Role | What | Actor |
|------|------|-------|
| **Chef** | Human — sets strategy, intervenes when they choose | Human |
| **Sous Chef** | Scheduler — reads the mise, prioritizes the queue, decides what to cook next | LLM agent (configurable model) |
| **Taster** | Quality reviewer — checks every dish before it leaves the pass | LLM agent |
| **Cook** | Does the work — uses native sub-agents (Claude Task tool, Codex parallelism) as needed | Claude or Codex session |
| **Mise** | Gathers state into a structured brief (mise en place) | Go code (not an actor) |

**Eliminated roles:** Director (cooks are spawned directly), Manager (cooks use native sub-agents), Operator (merged into Cook), Apprentice/Worker (native sub-agents handle this). President/CEO/CTO renamed to Chef/Sous Chef/Taster.

A cook is a cook — it receives a task and does it. Whether it delegates to sub-agents or works directly is the cook's own judgment based on the task, not a structurally distinct role. The sous chef determines which provider (Claude vs Codex) and model to use for each cook based on annotations made at plan time.

## Skills as the Only Extension Point

Skills are pure Claude Code skills (`SKILL.md` + optional `references/`). No per-skill Noodle metadata. A user's existing skills work with Noodle without modification — just wire them in via config.

All Noodle-specific wiring lives in `.noodle/config.toml`:

```toml
[phases]
quality = "my-review-skill"      # override default taster behavior
product-review = "my-review-skill"  # override default product review

[adapters.backlog]
skill = "my-backlog"             # skill teaches agents to interact
sync = ".noodle/scripts/sync-backlog.sh"  # deterministic reads for mise

[adapters.plans]
skill = "my-plans"
sync = ".noodle/scripts/sync-plans.sh"

[sous-chef]
run = "after-each"               # after-each | after-n | manual
model = "claude-sonnet"          # cheaper model for prioritization

[routing.defaults]
provider = "codex"
model = "gpt-5.3-codex"

[routing.tags]
frontend = { provider = "claude", model = "opus" }
design = { provider = "claude", model = "opus" }
tests = { provider = "codex", model = "spark" }

[skills]
paths = [".noodle/skills", "~/.noodle/skills"]  # project > user > bundled
```

**Templates are eliminated.** Today's embedded `noodle/templates/*.md` files become default skills resolved through the skill precedence chain. Role contracts become part of the skill's `SKILL.md` content. Agent definitions are derived from skill + config at resolution time rather than being separate `.md` files with YAML frontmatter.

## Adapter Pattern for Backlog and Plans

Users have diverse task/plan systems (brain/todos.md, Beads, Linear, GitHub Issues, internal tools). Noodle accommodates this via adapter skills with sync scripts:

1. **Skill** — teaches agents how to read/write the user's system (the `SKILL.md`).
2. **Sync script** — deterministic executable that outputs a normalized format for the mise. Separate from the skill, declared in config.
3. **Normalized format** — the contract between adapters and mise (NDJSON with defined schema).

Noodle ships default adapters for `brain/todos.md` and `brain/plans/`. Users who use different systems write their own sync scripts. The `noodle` meta-skill teaches agents how to create adapter skills and sync scripts.

This preserves the principle that the mise (Go code) operates on deterministic, structured data while the source of that data is fully customizable.

## LLM-Powered Prioritization

The current deterministic Go assessor tries to encode scheduling judgment in weighted heuristics (urgency * 40 + priority * 25 + resourceFit * 20 + ...). This doesn't work well — it can't express natural dependencies like "verification follows execution" without brittle special-case code.

**New model:**

1. **Mise** (Go) — gathers backlog state, resource state, active sessions, recent history into a structured brief. Pure data collection, no scoring.
2. **Sous Chef** (LLM agent) — reads the brief, applies scheduling judgment, outputs a prioritized queue with routing annotations. Runs after each task completion by default (configurable frequency and model).
3. **Go loop** — reads the prioritized queue mechanically. Enforces hard constraints (concurrency limits, exclusivity rules, resource availability). Spawns cooks.

Routing is determined at plan time: the planning agent annotates each phase with the recommended provider/model. The sous chef respects these annotations. Execution is fully deterministic — the Go loop just reads the queue and spawns.

**Fallback:** If the sous chef call fails, the Go loop uses a simple FIFO from the last successful prioritization. The system never blocks on a failed LLM call.

## Aggressive Go Code Deletion

The simpler model means significantly less Go code. Major deletions:

- **Scoring heuristics** (`assess_scoring.go`, scoring functions in `assess_core.go`) — replaced by LLM prioritization. Delete the weighted formula, urgency calculations, resource-fit scoring, recency penalties, quality boosts, duplicate-work penalties.
- **Role enforcement** (`role_enforcement.go`) — the Director/Manager/Operator/CEO/CTO role system. Replace with the three kitchen roles (sous-chef, taster, cook).
- **Role contracts** (`role_contracts.go`) — embedded template injection. Replaced by skill resolution.
- **Agent type constants** (Manager, Operator in `model.go`) — collapse to Cook.
- **Embedded templates** (`templates/*.md`, `templates/templates.go`) — replaced by default skills.
- **Complex model routing** (EntityModels, TaskModels, DomainModels, WorkerModel hierarchies in config) — routing lives in plan annotations, not Go code. Config keeps simple defaults.
- **CEO runtime** (`ceo_runtime.go`) — president-initiated priority overrides. Replaced by sous-chef agent.
- **Phase capabilities complexity** (`phase_caps.go`) — simplify. Fewer phases, fewer attributes per phase. Supervision policies can be simpler with fewer actor types.
- **Batch selection heuristics** (`assess_history.go`, `selectParallelBatch()`) — the LLM handles batch recommendation. Go enforces hard constraints only.

Target: the Go code that remains should be the minimal infrastructure loop — mise, spawn, monitor, log, TUI — with the intelligence living in skills.

## What Ships with Noodle

**Tier 1 — Go binary (lean core):**
- Mise (data gathering, brief construction)
- Cook loop (read queue, enforce constraints, spawn, monitor, log)
- Spawner (tmux today, Spawner interface for future cloud)
- TUI
- CLI (init, cook, status, skills, etc.)
- Skill resolver (project > user > bundled precedence)
- Adapter runner (execute sync scripts, read normalized output)

**Tier 2 — Default skills (ship alongside, fully overridable):**
- `sous-chef` — scheduling and prioritization
- `taster` — quality review
- `backlog` — teaches agents to read/write `brain/todos.md`
- `plans` — teaches agents to read/write `brain/plans/`
- `noodle` — meta-skill: how to configure Noodle, create adapter skills, write sync scripts
- `bootstrap` — detects missing infra, scaffolds defaults, adapts to existing projects

**Tier 3 — Default adapters (sync scripts):**
- `sync-backlog.sh` — parses `brain/todos.md` to normalized NDJSON
- `sync-plans.sh` — parses `brain/plans/` to normalized NDJSON

**Bootstrap flow:**
1. `noodle init` (Go CLI) — writes minimal `.noodle/config.toml` with sensible defaults
2. `noodle bootstrap` (spawns agent with bootstrap skill) — detects missing pieces (brain folder? backlog? plans?), creates appropriate defaults or adapts to what exists

## Repository Extraction

Noodle moves from the parent project's `noodle/` subdirectory to its own repository at `~/code/noodle/`. This is the first major step — extract cleanly, then redesign in the new home.

**Extraction strategy: duplicate the original project, then prune.**

1. **Copy the entire original project** to `~/code/noodle/` as a new git repo with fresh history. This gives us all the dev tooling (Claude settings, skills, hooks, brain, principles) as a starting point.
2. **Move `noodle/` to `old_noodle/`** — the original Go code becomes reference material for the rewrite, not the active source.
3. **Delete parent-project-specific files** — frontend (`src/`, `src-tauri/`), package management, project-specific brain plans/todos, product-specific skills. Keep general-purpose dev skills (debugging, commit, worktree, plan, review, etc.) and brain principles.
4. **Adapt project config** — rewrite CLAUDE.md for Noodle, update hooks, clean up brain references.
5. **The parent project is left completely untouched.** Cleanup of its `noodle/` subdirectory is a separate, later task.

## Scope

In scope:
- **Repository extraction** — move Noodle to `~/code/noodle/` as its own repo, parent project references via relative path
- Kitchen brigade renaming across the entire codebase (Go, skills, TUI, CLI, docs)
- Actor model simplification (Director/Manager/Operator/Apprentice → Cook)
- Skills as the only extension point (no per-skill metadata, centralized config)
- Embedded template → skill migration (eliminate `templates/` embed system)
- Agent definition → skill+config derivation (eliminate separate agent `.md` files)
- LLM-powered prioritization (replace deterministic scoring with sous-chef agent)
- Adapter pattern for backlog/plans (skill + sync script + normalized format)
- Aggressive Go code deletion (scoring, role enforcement, complex model routing, CEO runtime)
- Skill resolver (layered precedence: project > user > bundled)
- Config redesign (centralize all Noodle wiring in `.noodle/config.toml`)
- Default skills and adapters for out-of-box experience
- Bootstrap skill and `noodle init` command
- Enhanced `noodle` meta-skill (teaches agents how to extend Noodle)

Out of scope:
- Publishing Noodle as a standalone installable package (relative path for now)
- Remote skill registry or marketplace
- Cloud spawner implementation
- Web UI
- New TUI features beyond renaming
- Changes to the stamp/NDJSON log pipeline

## Constraints

- **Copy first, redesign second:** Noodle is copied (not moved) to `~/code/noodle/` before architectural changes begin. The parent project is left completely untouched — no risk of breakage. Parent project cleanup is a separate later task.
- **Cross-platform:** macOS local dev, Windows/Linux CI. Sync scripts must be POSIX-compatible or have platform alternatives. No bash 4+ features.
- **Boundary discipline:** Config validation happens at command boundaries. Skill resolver internals are pure functions. Sync scripts are called at the boundary and their output is validated before the mise consumes it.
- **No backwards compatibility:** This is a hard cutover. Legacy templates, agent definitions, role constants, and scoring code are deleted, not adapted. Per the prelaunch posture: no external users yet, prefer deletion over compatibility layers.
- **Skills stay pure:** A Noodle skill is just a Claude Code skill. No Noodle-specific file format, frontmatter, or metadata inside the skill directory. If Noodle needs to know something about a skill, it's declared in config.
- **Deterministic execution:** The Go loop never makes judgment calls. Judgment lives in the sous chef (LLM). The Go loop reads the queue and enforces hard constraints mechanically.
- **Relative path dependency:** The parent project references Noodle at `../noodle/` during the transition. No Go module dependency or binary installation required yet.

Design alternatives considered:
1. **Per-skill `noodle.toml` metadata** for declaring what a skill provides to Noodle.
   Chosen? No.
   Why not: adds a Noodle-specific format that couples skills to Noodle. Centralizing wiring in config keeps skills pure and means existing skills work without modification.

2. **Keep deterministic Go scoring, LLM only for overrides.**
   Chosen? No.
   Why not: the weighted scoring formula can't express natural scheduling dependencies. Every new heuristic is a special case bolted onto a numeric model. LLM prioritization handles the full richness of scheduling judgment.

3. **Keep the Director/Manager/Operator hierarchy, just rename.**
   Chosen? No.
   Why not: the hierarchy exists because Noodle didn't trust agents to self-organize. Modern Claude and Codex have native sub-agent capabilities. A cook that gets a big task naturally delegates. The extra roles added complexity without value.

4. **Separate Cook role for Claude vs Codex (cook-claude / cook-codex).**
   Chosen? No.
   Why not: a cook is a cook. The provider is a routing decision, not a role distinction. The plan annotations and config defaults handle provider selection.

5. **Redesign in place within the parent project, extract later.**
   Chosen? No.
   Why not: extracting first means the redesign happens in the final home. Avoids doing the work twice (once in the parent project, then moving). Relative path reference keeps the parent project working during transition.

6. **Refactor existing codebase (delete and modify) vs rewrite from scratch (pull in proven pieces).**
   Chosen? Rewrite.
   Why: Only ~14% of the current Go code (~4,600 LOC out of ~32,000) lives in cleanly separable packages (parse, event, worktree, model, tmux spawner, stamp). The remaining 86% is a tightly coupled monolith where the role/scoring/template system is the structural spine. Refactoring means touching every file in `cook/` and produces scar tissue shaped by deletions rather than by the new design. A rewrite pulls in the proven library packages verbatim and writes the new core fresh — fewer lines changed, cleaner architecture, meaningful git history from day one.

## Applicable Skills
- `noodle`
- `skill-creator`
- `debugging`
- `worktree`

## Phases

### Repo Extraction (duplicate the original project, prune to Noodle, leave the parent project untouched)
- [[plans/01-noodle-extensible-skill-layering/phase-01-init-repo-and-copy-go-source]]
- [[plans/01-noodle-extensible-skill-layering/phase-02-copy-skills-agents-hooks-templates]]
- [[plans/01-noodle-extensible-skill-layering/phase-03-create-noodle-claude-md-and-config]]
- [[plans/01-noodle-extensible-skill-layering/phase-04-verify-standalone-build-and-test]]

### Architecture Redesign (TBD — planned after extraction is complete)

## Verification

- Static:
  - `go -C ~/code/noodle test ./...`
  - `go -C ~/code/noodle run . commands --json`
- Runtime:
  - Repo extraction: Noodle builds and tests pass in `~/code/noodle/`. The parent project is completely untouched and still works with its local `noodle/` copy.
  - Skill precedence: project override beats user override beats bundled default.
  - Adapter sync: custom sync script output is consumed correctly by mise.
  - Sous chef cycle: LLM prioritization runs, produces ordered queue, Go loop spawns from it.
  - Kitchen renaming: TUI, CLI output, log entries all use new terminology.
  - Bootstrap: `noodle init` + `noodle bootstrap` in a clean temp repo produces a working setup.
  - Config wiring: phase skill overrides and routing tags take effect during cook.
  - Code deletion: no dead code referencing eliminated roles, templates, or scoring.
