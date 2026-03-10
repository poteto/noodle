---
id: 1
created: 2026-02-20
updated: 2026-02-22
status: done
---

# Noodle Open-Source Architecture

## Context

Noodle is an open-source framework for AI coding. It can be dropped into an existing project or used to scaffold a new one with Noodle as its central operating system. The Go module is at the repo root. Only the `worktree` package was carried over — everything else is being written fresh.

**Governing insight: skills are the only extension point.** Users extend Noodle by writing or overriding skills — pure Claude Code skills with no Noodle-specific metadata. All wiring lives in `noodle.toml`. The Go binary is lean infrastructure: gather state, call an LLM for prioritization, spawn cooks, monitor, log.

## Kitchen Brigade

All actor roles adopt kitchen terminology:

| Role | What | Actor |
|------|------|-------|
| **Chef** | Human — sets strategy, intervenes when they choose | Human |
| **Sous Chef** | Scheduler — reads the mise, prioritizes the queue, decides what to cook next | LLM agent (configurable model) |
| **Taster** | Quality reviewer — checks every dish before it leaves the pass | LLM agent |
| **Cook** | Does the work — uses native sub-agents (Claude Task tool, Codex parallelism) as needed | Claude or Codex session |
| **Mise** | Gathers state into a structured brief (mise en place) | Go code (not an actor) |

A cook is a cook — it receives a task and does it. Whether it delegates to sub-agents or works directly is the cook's own judgment based on the task, not a structurally distinct role. The sous chef determines which provider (Claude vs Codex) and model to use for each cook based on annotations made at plan time.

## Session Types

Every agent session Noodle spawns is structurally a cook — same spawner, same lifecycle, same monitoring. The session type determines which skill is loaded and what triggers the spawn:

| Session Type | Trigger | Default Skill | Purpose |
|---|---|---|---|
| **Cook** | Scheduling loop, from queue | Per backlog item (sous chef annotates) | Execute a backlog item |
| **Sous Chef** | Scheduling loop, each cycle | `sous-chef` | Read mise, prioritize queue, annotate routing |
| **Taster** | After cook completes (if `review: true`) | `taster` | Review cook's work, accept/reject |
| **Oops** | Runtime, on user codebase/workflow issue | Configurable via `phases.oops` | Fix broken tests, builds, or project infrastructure |
| **Repair** | `noodle start` startup or runtime, on Noodle config issue | `noodle` | Fix broken Noodle config/infrastructure |
| **Reflect** | After significant sessions | `reflect` | Persist learnings to brain |
| **Meditate** | Periodic or on demand | `meditate` | Audit brain, prune stale content, find connections |
| **Debate** | Taster or direct invocation | `debate` | Multi-round structured validation |

The oops session is distinct from repair: repair fixes Noodle's own configuration, while oops addresses problems in the user's project (broken tests, failing builds, workflow issues). The oops skill is configurable — users specify their own via `phases.oops` in config. Noodle ships a default that teaches agents generic debugging methodology, but users with project-specific tooling should override it.

## Skills as the Only Extension Point

Skills are pure Claude Code skills (`SKILL.md` + optional `references/`). No per-skill Noodle metadata. A user's existing skills work with Noodle without modification — just wire them in via config.

All Noodle-specific wiring lives in `noodle.toml`:

```toml
[phases]
quality = "my-review-skill"      # override default taster behavior
product-review = "my-review-skill"  # override default product review

[adapters.backlog]
skill = "my-backlog"             # skill teaches agents the semantics

[adapters.backlog.scripts]       # commands, not files — like package.json scripts
sync = "gh issue list --json number,title,body,labels,state | jq -c '...'"
add = ".noodle/adapters/backlog-add"
done = "gh issue close"
edit = ".noodle/adapters/backlog-edit"

[adapters.plans]
skill = "my-plans"

[adapters.plans.scripts]
sync = ".noodle/adapters/plans-sync"
create = ".noodle/adapters/plan-create"
done = ".noodle/adapters/plan-done"
phase-add = ".noodle/adapters/plan-phase-add"

[sous-chef]
run = "after-each"               # after-each | after-n | manual
model = "claude-sonnet"          # cheaper model for prioritization

[routing.defaults]                  # example: route to Codex by default
provider = "codex"
model = "gpt-5.4"

[routing.tags]
frontend = { provider = "claude", model = "opus" }
design = { provider = "claude", model = "opus" }
tests = { provider = "codex", model = "spark" }

[skills]
paths = ["skills", "~/.noodle/skills"]  # project > user
```

## Adapter Pattern for Backlog and Plans

Users have diverse task/plan systems (brain/todos.md, Linear, GitHub Issues, internal tools). Noodle accommodates this via adapters — a skill plus a set of scripts per action:

1. **Skill** — teaches agents the semantics of the user's system (the `SKILL.md`). What makes a good issue title, how to describe work, what labels mean.
2. **Scripts** — deterministic commands for each CRUD action: `sync` (read all), `add`, `done`, `edit`. Declared in config, executed by the Go binary. Can be any executable — shell scripts, Python, compiled binaries, or inline commands like `gh issue close`.
3. **Normalized schemas** — the contract between adapter sync output and the mise. `BacklogItem` and `PlanItem` NDJSON with required/optional fields defined in Phase 8.

The user's agent is responsible for creating adapter scripts that transform their system into Noodle's required format. The `noodle` meta-skill teaches agents how to write adapters. Noodle ships default adapters for `brain/todos.md` and `brain/plans/`.

This preserves the principle that the mise (Go code) operates on deterministic, structured data while the source of that data is fully customizable. The Go binary can mechanically call `done 42` when a cook finishes — no LLM needed to interact with the user's backlog system.

**Adapters are optional.** If `[adapters.plans]` is omitted from config, Noodle operates with backlog only — plans data is an empty array in the mise. If both adapters are omitted, the mise contains only internal state (active cooks, tickets, resources). This supports teams that use GitHub Issues with no plan artifacts, or projects still setting up their workflow.

## Directory Layout

Three locations, each with a clear purpose:

**`noodle.toml`** — Project root. Committed to git. All Noodle wiring: skill paths, routing, adapters, thresholds.

**`.noodle/`** — Project directory. Gitignore-able. Runtime state only — nothing here is configuration or needs version control:

```
.noodle/
├── mise.json                    # gathered state brief
├── queue.json                   # prioritized queue from sous chef
├── tickets.json                 # materialized active tickets
├── adapters/                    # default adapter scripts
└── sessions/
    ├── fix-auth-bug/
    │   ├── meta.json            # status, provider, model, cost, health
    │   └── events.ndjson        # append-only event log
    └── add-user-tests/
        ├── meta.json
        └── events.ndjson
```

**`~/.noodle/`** — Global user directory. Skills, binary, per-project state:

```
~/.noodle/
├── bin/                         # noodle binary
├── skills/                      # user-level default skills
└── projects/
    └── my-project-name/         # human-readable, like Claude's ~/.claude/projects/
        └── ...                  # future: cross-project state
```

Project directories under `~/.noodle/projects/` use human-readable names derived from the project path (same convention as Claude's `~/.claude/projects/`), not opaque IDs.

## LLM-Powered Prioritization

The queue and mise are plain JSON files in `.noodle/`, readable by humans and agents:

1. **Mise** (Go) — gathers state into `.noodle/mise.json`. Pure data collection, no scoring.
2. **Sous Chef** (LLM agent) — reads `.noodle/mise.json`, applies scheduling judgment, writes `.noodle/queue.json` with prioritized items and routing annotations.
3. **Go loop** — watches `.noodle/queue.json` via fsnotify. Enforces hard constraints (concurrency limits, exclusivity, tickets). Spawns cooks, always in dedicated worktrees.

**Autonomous ship contract (MVP):** Every cook runs in an auto-created worktree. Once work is verified and accepted by the taster (or review is skipped), the system merges that worktree directly into `main`. Preferred path: the cook uses the `commit` skill to commit and merge. Fallback path: if the cook finishes without merging, the loop detects verified completion and merges the worktree to `main` before marking the item done.

The sous chef doesn't need special binary commands for scheduling — it just reads one file and writes another. The human can inspect both files at any time.

**Fallback:** If the sous chef fails, the loop uses the last valid `.noodle/queue.json` on disk. If no queue file exists, FIFO from the mise.

## What Ships with Noodle

**Tier 1 — Go binary (lean core):**
- Mise (data gathering, brief construction)
- Scheduling loop (read queue, enforce constraints, spawn, monitor, log)
- Spawner (tmux today, Spawner interface for future cloud)
- TUI
- CLI (user-facing: start, status, skills, worktree; advanced/internal: mise, stamp, spawn, tui)
- Skill resolver (ordered search paths from config)
- Adapter runner (execute sync scripts, read normalized output)

**Tier 2 — Default skills (in repo, installed by bootstrap):**
- `bootstrap` — the entry point for new users (see below)
- `sous-chef` — scheduling and prioritization
- `taster` — quality review
- `commit` — commit + merge workflow guidance for autonomous shipping (worktree to `main`)
- `backlog` — teaches agents to read/write the user's backlog system
- `plans` — teaches agents to read/write the user's plan system
- `noodle` — meta-skill: how to configure Noodle, create adapter skills, write sync scripts
- `reflect` — session reflection: persists learnings to the brain
- `meditate` — brain maintenance: audits vault, discovers connections, prunes stale content
- `oops` — diagnoses and fixes project infrastructure issues (broken tests, builds, workflow)
- `debugging` — systematic debugging methodology, loaded by oops and repair sessions
- `doctor` — troubleshooting guide for broken Noodle state; escalates unfixable issues to GitHub

**Tier 3 — Default adapter scripts (installed to `.noodle/adapters/` by bootstrap, always available):**
- Backlog (`brain/todos.md`): `backlog-sync`, `backlog-add`, `backlog-done`, `backlog-edit`
- Plans (`brain/plans/`): `plans-sync`, `plan-create`, `plan-done`, `plan-phase-add`

Scripts are always installed so they're available if the user later enables an adapter. Whether an adapter is *configured* in `noodle.toml` depends on bootstrap choices — interactive users pick their setup, non-interactive mode configures backlog only.

**Bootstrap flow:**
The bootstrap skill is the entry point for adoption. A user tells their agent to look at the Noodle GitHub repo, which has instructions to install the bootstrap skill. The bootstrap skill then:
1. Creates `~/.noodle/` and `~/.noodle/skills/`
2. Copies default skills from the repo to `~/.noodle/skills/`
3. Builds the Go binary and places it on PATH (or `~/.noodle/bin/`)
4. Creates `noodle.toml` in the project with sensible defaults
5. Creates `.noodle/adapters/` with default sync scripts
6. Adds `.noodle/` to the project's `.gitignore`
7. Detects what the project already has and adapts accordingly

No `noodle init` Go command — the bootstrap skill handles all setup intelligently.

## Scope

In scope:
- Kitchen brigade model (chef, sous chef, taster, cook, mise)
- Skills as the only extension point (no per-skill metadata, centralized config)
- LLM-powered prioritization (sous chef agent)
- Adapter pattern for backlog/plans (skill + scripts + normalized format)
- Skill resolver (layered precedence: project > user, via config paths)
- Config system (centralize all wiring in `noodle.toml`)
- Default skills and adapters for out-of-box experience
- Bootstrap skill as the entry point for new users
- Self-healing (repair cooks for fixable issues, actionable errors for fatal ones)
- Opinionated worktree lifecycle for cooks (auto-create, verify, merge-to-`main`)
- TUI, CLI, monitoring, debate system

Out of scope:
- Standalone installable package (relative path for now)
- Remote skill registry or marketplace
- Cloud spawner implementation
- Automated pull request creation/management flow
- Web UI

## Reference Codebase

The previous Noodle implementation lives in a private repository. Its path is stored in `.noodle/reference-path` (gitignored). Agents that need to consult the old code for patterns or proven implementations should read that file to discover the path.

This is a greenfield rewrite — default to writing fresh code. Consult the reference codebase only when a phase explicitly calls for it (e.g., proven parsing logic, tmux session management patterns). When adapting old code, rewrite it to fit the new architecture rather than copying verbatim.

**Leak avoidance:** The resolved path from `.noodle/reference-path` must never appear in committed code, comments, logs, error messages, or plan files. Always refer to it indirectly (e.g., "the reference codebase") and instruct agents to read the path at runtime.

**If missing:** `.noodle/reference-path` is advisory, not required. If the file is absent (CI, new contributors, clean clone), phases proceed fully greenfield — the reference codebase is a design aid, not a dependency. No warning or error needed.

## Cross-Provider Skill Contract

Skills are authored as Claude Code skills (`SKILL.md` + optional `references/`), but cooks can run on any provider (Claude, Codex, future providers). The spawner handles the translation:

- **Claude** — Skills are loaded natively as Claude Code skills. No translation.
- **Codex** — The spawner reads the `SKILL.md` content and injects it as system instructions in the Codex session. Codex supports markdown instructions natively. The `references/` directory contents are concatenated and included as context.
- **Future providers** — Same pattern: read skill content, inject as the provider's instruction mechanism.

Skills stay provider-agnostic because they're just markdown documents that teach an agent how to do something. The spawner is the translation boundary — it knows how each provider accepts instructions. Skill authors don't need to think about providers; the skill format is universal.

If a skill uses Claude-specific features (like MCP tool references or Claude Code slash commands), it won't work on other providers. The `noodle` meta-skill should document which features are portable and which are Claude-only.

## Chef Intervention

The chef (human) has explicit control primitives for the scheduling loop:

| Control | TUI | Effect |
|---------|-----|--------|
| **Pause/Resume** | `p` (toggle) | Stop/resume spawning new cooks. Active cooks continue. |
| **Drain** | `d` | Finish all active cooks, then stop. No new spawns. |
| **Steer** | `s` (any view) | Opens command bar. `@cook-name prompt` redirects a cook (kill + respawn with resume context). `@sous-chef prompt` triggers a re-prioritization cycle with the chef's guidance. Autocompletes actor names. |
| **Kill** | `k` on cook | Terminate a specific cook session. Triggers recovery. |
| **Skip** | `x` on queue item | Remove item from queue. Returns to backlog for future cycles. |

The TUI is the sole intervention surface — it shows what's happening and accepts commands inline. The TUI footer shows only essential keys (4-5 per view); `?` opens a help overlay with the full set. For scripting/automation, the control channel (`.noodle/control.ndjson`) is the programmatic interface — any tool can append commands directly.

**Steer** is the chef's primary creative control — one command bar to talk to any actor. `@cook-name` redirects a cook (kill → resume context → respawn with new instructions). `@sous-chef` triggers an immediate re-prioritization cycle with the chef's guidance layered in — reshaping the queue without manual item-by-item edits.

**Pause vs drain:** Pause is immediate (no new cooks start, but current ones keep running) and resumable. Drain is graceful (let everything finish, then stop) and terminal for the current run.

## Restart Reconciliation

When `noodle start` boots, it may find leftover state from a previous run — orphaned tmux sessions, stale state files, abandoned tickets. The startup sequence reconciles:

1. **Scan tmux** — Find all tmux sessions matching the Noodle naming pattern. For each:
   - If a session has a matching `meta.json` with `status: running` → **adopt** it. Resume monitoring.
   - If a session exists but `meta.json` shows `exited` or is missing → **kill** the orphaned tmux session.
2. **Scan state files** — For each session in `.noodle/sessions/`:
   - If `meta.json` shows `running` but no matching tmux session exists → mark as `exited` (crashed). Remove its tickets. Add to recovery candidates.
3. **Clean tickets** — Regenerate `.noodle/tickets.json` from live session events only. Dead sessions' tickets are dropped.
4. **Clean queue** — Validate `.noodle/queue.json` items. Remove items for cooks already in progress (adopted). Keep items not yet started.
5. **Resume or regenerate** — If the queue and mise are valid, resume the loop. If stale, regenerate the mise and trigger a sous chef cycle.

The result: `noodle start` is always safe to run. If the previous run crashed, the new run picks up where it left off — adopting live sessions, recovering failed ones, and cleaning up orphans. No manual cleanup required.

## Security and Trust

Noodle runs arbitrary code (adapter scripts, agent sessions) with the user's permissions. The trust model:

- **Adapter scripts are user-authored code.** They run with the same permissions as the user. Noodle doesn't sandbox them — they're treated like any other script the user writes. The user is responsible for not putting secrets in their adapter scripts' stdout (which flows into `mise.json`).
- **State files may contain project context.** `.noodle/mise.json`, `queue.json`, and `events.ndjson` contain backlog titles, descriptions, file paths, and cost data. These files are gitignored by default. They should not contain secrets, but users should be aware they exist.
- **Event logs are append-only and local.** They are never sent to external services by Noodle. The agent provider (Claude, Codex) has its own data policy — Noodle doesn't add to or reduce that exposure.
- **Skill content is injected into agent sessions.** Skill authors should not embed secrets in `SKILL.md` or `references/` — these are passed to LLM providers as context.
- **Cost data is local.** Session costs are tracked in `meta.json` for the user's benefit. They are not reported externally.

**Safe defaults:** Logs don't capture raw LLM responses (only structured events). State files are gitignored. The bootstrap skill does not write secrets to config. Adapter script stderr is captured for error reporting but not persisted to event logs.

**Event payload hygiene:** Structured events in `events.ndjson` contain tool names, file paths, cost data, and ticket targets — not raw LLM input/output. Backlog titles and descriptions flow through `mise.json` as-is from adapter sync output. Noodle does not actively redact secrets from these payloads — the user is responsible for not putting secrets in backlog titles, descriptions, or plan content. If a user's adapter script accidentally outputs secrets, they appear in `mise.json` (gitignored, local only). The `noodle` meta-skill should warn about this.

## Constraints

- **Cross-platform:** macOS and Linux natively. **Windows requires WSL** — tmux is the spawner and has no native Windows equivalent. A non-tmux spawner (e.g., process-based for local dev, cloud-based for CI) is out of scope for this plan but the `Spawner` interface makes it addable later. Sync scripts must be POSIX-compatible or have platform alternatives. No bash 4+ features. Agent CLI directories (`.claude/`, `.codex/`) have platform-specific paths — the bootstrap skill detects the correct paths for the user's OS and writes them to config. The Go binary reads these from config rather than hardcoding platform logic.
- **Boundary discipline:** Config validation happens at command boundaries. Skill resolver internals are pure functions. Sync scripts are called at the boundary and their output is validated before the mise consumes it.
- **Skills stay pure:** A Noodle skill is just a Claude Code skill. No Noodle-specific file format, frontmatter, or metadata inside the skill directory. If Noodle needs to know something about a skill, it's declared in config.
- **Deterministic execution:** The Go loop never makes judgment calls. Judgment lives in the sous chef (LLM). The Go loop reads the queue and enforces hard constraints mechanically.
- **Self-healing (`noodle start` only):** When the scheduling loop detects a misconfiguration or broken state, it has two tiers of response. **Tier 1 (repair):** if the spawner is functional, spawn a cook with the `noodle` meta-skill to diagnose and fix the issue — missing adapter scripts, invalid config values, missing skills, stale paths. **Tier 2 (guide):** if Noodle cannot spawn, produce a detailed, actionable error — what's wrong, what the expected state is, and exact steps to fix it. Read-only commands (`noodle status`, `noodle skills list`, etc.) never trigger repair cooks — they validate and report, but don't mutate.

Design alternatives considered:
1. **Per-skill `noodle.toml` metadata** — couples skills to Noodle. Centralizing wiring in config keeps skills pure.
2. **Deterministic Go scoring** — can't express scheduling dependencies. LLM prioritization handles the full richness of scheduling judgment.
3. **Separate Cook role per provider** — a cook is a cook. The provider is a routing decision, not a role distinction.

## Applicable Skills
- `noodle`
- `skill-creator`
- `debugging`
- `worktree`

## Phases

1. [[archive/plans/01-noodle-extensible-skill-layering/phase-01-scaffold]] — Project scaffold + core types
2. [[archive/plans/01-noodle-extensible-skill-layering/phase-02-config]] — Config system
3. [[archive/plans/01-noodle-extensible-skill-layering/phase-03-skill-resolver]] — Skill resolver
4. [[archive/plans/01-noodle-extensible-skill-layering/phase-04-ndjson-pipeline]] — NDJSON pipeline (stamp + parse)
5. [[phase-05-event-log]] — Event log + tickets
6. [[archive/plans/01-noodle-extensible-skill-layering/phase-06-spawner]] — Spawner
7. [[archive/plans/01-noodle-extensible-skill-layering/phase-07-monitor]] — Monitor
8. [[archive/plans/01-noodle-extensible-skill-layering/phase-08-adapters-mise]] — Adapter runner + mise
9. [[archive/plans/01-noodle-extensible-skill-layering/phase-09-scheduling-loop]] — Scheduling loop + recovery
10. [[archive/plans/01-noodle-extensible-skill-layering/phase-10-cli]] — CLI commands
11. [[archive/plans/01-noodle-extensible-skill-layering/phase-11-default-skills]] — Default skills + adapters
12. [[archive/plans/01-noodle-extensible-skill-layering/phase-12-debate]] — Debate system
13. [[archive/plans/01-noodle-extensible-skill-layering/phase-13-tui]] — TUI
14. [[archive/plans/01-noodle-extensible-skill-layering/phase-14-bootstrap-readme]] — Bootstrap + README

## Verification

- Static:
  - `go -C ~/code/noodle test ./...`
  - `go -C ~/code/noodle vet ./...`
  - `go -C ~/code/noodle run . commands --json`
- Runtime:
  - Skill precedence: project skill overrides user skill of same name.
  - Adapter sync: custom sync script output is consumed correctly by mise.
  - Sous chef cycle: sous chef reads `.noodle/mise.json`, writes `.noodle/queue.json`, Go loop picks up changes via fsnotify and spawns.
  - Kitchen renaming: TUI, CLI output, log entries all use new terminology.
  - Bootstrap: invoke bootstrap skill in a clean repo → working Noodle setup.
  - Config wiring: phase skill overrides and routing tags take effect during cook.
