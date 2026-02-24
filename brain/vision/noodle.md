# Noodle: Open-Source AI Coding Framework

## The Problem

The human is the scheduler for all project work. Audits, plans, execution, review — every step blocks on human attention.

## The Vision

Noodle replaces the human as the top-level scheduler. It reads your backlog, decides what matters, spawns agents to do the work, reviews the output, and loops. The human observes via TUI and intervenes when they choose to.

Three ideas run through the whole design:

1. **Everything is a file.** The loop reads and writes JSON/NDJSON in `.noodle/`. The TUI reads those same files. There's no internal state that isn't on disk.
2. **LLMs do judgment, Go does mechanics.** The scheduling loop never decides *what* to work on — it reads a queue file. The prioritize skill (an LLM) writes that file. The quality skill (an LLM) writes verdict files. Go code just enforces constraints and moves files around.
3. **Skills are the only extension point.** No plugin API, no hook system, no registries. A skill is a `SKILL.md` file with optional YAML frontmatter. If it has `noodle:` frontmatter, the loop discovers it as a task type automatically.

## Kitchen Brigade

| Role | What | Actor |
|------|------|-------|
| **Chef** | Human — sets direction, intervenes when they want | Human via TUI/CLI |
| **Prioritize** | Reads the mise brief, writes the prioritized queue | LLM session |
| **Quality** | Reviews completed work, writes accept/reject verdict | LLM session |
| **Cook** | Does the work — execute, plan, reflect, debug, whatever the skill says | LLM session |
| **Mise** | Gathers all project state into a structured brief | Go code |

The chef is always human. Everything else is an LLM session loaded with the right skill, or Go code doing mechanical work.

## Everything is a File

All loop state lives in `.noodle/` as JSON or NDJSON. If it's not on disk, it doesn't exist.

**mise.json** — The mise brief. Go code gathers backlog items, plan phases, active sessions, recent history, resource capacity, discovered task types, and quality verdicts into one file. The prioritize skill reads this to make scheduling decisions.

**queue.json** — The prioritized work queue. Written by the prioritize skill, read by the loop. Each item has a task key, skill name, routing info, and rationale. The loop reads items off the top and spawns them.

**quality/\<session-id\>.json** — Verdict files. The quality skill writes one per completed cook: accept or reject, with feedback. The loop reads these to decide whether to merge the worktree.

**control.ndjson** — Commands from the TUI to the loop. Pause, resume, drain, skip, kill, steer, merge, reject, change autonomy. The loop processes and empties the file each cycle, writes acks to `control-ack.ndjson`.

**sessions/\<session-id\>/events.ndjson** — Per-session event stream. Spawned, cost, state changes, ticket claims, completion. The monitor reads these to detect stuck sessions and materialize ticket state.

## Skills as the Only Extension Point

A skill is a directory containing `SKILL.md` and optional `references/`. The `SKILL.md` is a Claude Code system prompt — markdown that gets loaded as the agent's instructions for that task.

Skills become Noodle task types by adding YAML frontmatter:

```yaml
---
name: execute
description: Implementation methodology for cook sessions
noodle:
  blocking: false
  schedule: "When backlog items with linked plans are ready"
---
```

The `noodle:` block is what makes it a task type. On startup, the skill resolver scans configured paths, parses frontmatter, and builds a task type registry. No hardcoded list — if a skill has `noodle:` frontmatter, the loop can schedule it.

**blocking** controls concurrency: a blocking task type (like prioritize) runs alone. Non-blocking types (execute, quality, reflect) run in parallel up to `max_cooks`.

**schedule** is a natural-language hint for the prioritize skill. It reads these hints when deciding what to put in the queue.

Skills resolve by path order in `.noodle.toml`. Project skills shadow user skills shadow bundled skills. Override any default by dropping a `SKILL.md` with the same name in your project's skill path.

## Architecture

```
                 ┌─────────────────────┐
                 │    Chef (TUI/CLI)   │
                 │  observe · command  │
                 └─────────┬───────────┘
                           │ control.ndjson
┌──────────────────────────┼──────────────────────────┐
│                     noodle start                     │
│                          │                           │
│  ┌─────┐  mise.json  ┌──────────┐  queue.json       │
│  │Mise ├─────────────►│Prioritize├──────────┐        │
│  │(Go) │              │ (skill)  │          │        │
│  └─────┘              └──────────┘          ▼        │
│                                        ┌────────┐    │
│                                        │  Loop  │    │
│                                        │  (Go)  │    │
│                                        └───┬────┘    │
│                              ┌─────────────┼─────┐   │
│                              ▼             ▼     ▼   │
│                          ┌──────┐    ┌──────┐  ...   │
│                          │Cook 1│    │Cook 2│        │
│                          │(tmux)│    │(tmux)│        │
│                          └──┬───┘    └──────┘        │
│                             │                        │
│                             ▼                        │
│                        ┌─────────┐                   │
│                        │Quality  │  verdict JSON     │
│                        │(skill)  ├──────────────►    │
│                        └─────────┘     .noodle/      │
│                                       quality/       │
└──────────────────────────────────────────────────────┘
```

Each cycle: mise gathers state, prioritize writes the queue, the loop spawns cooks in tmux sessions. When a cook finishes, quality reviews it. Approved work gets merged from worktree to main.

## Autonomy Dial

Trust is a dial with three positions:

- **full** — quality runs, auto-merge on accept. No human in the loop.
- **review** — quality runs, verdicts show in the TUI. Human merges or rejects. Default.
- **approve** — same as review, but human must also confirm every merge.

Set in `.noodle.toml` as `autonomy = "review"` or change live via TUI. The loop reads the current mode each cycle.

## Adapters

Adapters bridge your existing backlog and plan formats to Noodle. An adapter is:

1. A skill that teaches agents how to read/write your format
2. Shell scripts declared in `.noodle.toml` that the mise calls for sync/add/done/edit

The adapter scripts output normalized NDJSON. The mise reads it. No Noodle-specific format required in your actual backlog files.

## Configuration

One file: `.noodle.toml` at project root. Declares routing defaults, skill paths, agent binaries, adapter scripts, concurrency limits, autonomy mode, and recovery settings. Schema reference: `config/config.go`.

## What Ships

**Go binary:**
- Mise (state gathering)
- Loop (queue reading, constraint enforcement, spawning, monitoring)
- Dispatcher (tmux session management, NDJSON pipeline)
- Monitor (session observation, stuck detection, ticket materialization)
- Skill resolver (frontmatter parsing, path-based precedence)
- Adapter runner (script execution, NDJSON normalization)
- TUI and CLI

**Default skills (all overridable):**
- `prioritize` — scheduling and queue writing
- `quality` — post-cook review
- `execute` — implementation methodology
- `reflect` — post-session learning
- `oops` — infrastructure failure recovery
- `debugging` — root-cause investigation
- `plan` — phased planning
- `review` — code review
- `debate` — structured design decisions

**Default adapters:**
- `backlog-sync` — parse backlog to normalized NDJSON
- `backlog-add/done/edit` — write operations

## Principles Applied

- [[principles/never-block-on-the-human]] — agents stay unblocked; the human observes asynchronously
- [[principles/cost-aware-delegation]] — LLM for judgment, Go for mechanics
- [[principles/encode-lessons-in-structure]] — skills encode process knowledge; the loop enforces constraints
- [[principles/prove-it-works]] — the event log is the source of truth
- [[principles/subtract-before-you-add]] — one extension point (skills), one config file, file-based state
