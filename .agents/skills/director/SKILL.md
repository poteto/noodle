---
name: director
model: opus
description: >
  Orchestrate work at the right scale — from doing trivial tasks directly, to
  spawning a single manager for simple work, to full multi-manager orchestration
  for complex projects. Triages task complexity first, then picks the cheapest
  execution mode that gets the job done. Use when the user says "direct this",
  "run the director", "start a manager", "orchestrate", or wants a higher-level
  agent to supervise manager sessions.
allowed-tools:
  - Read
  - Write
  - Edit
  - Bash
  - Glob
  - Grep
  - Task
  - TaskCreate
  - TaskGet
  - TaskList
  - TaskUpdate
  - TaskOutput
  - Skill
---

# Director

## Cardinal Rules

### Match Effort to Task

Get work done at the right cost. Spawning a manager for a one-file change is waste; doing a multi-file refactor yourself is also waste. Triage first.

### Never Do the Work (in orchestration mode)

Orchestrate only — never explore, research, implement, or test. Delegate via composed prompts.

- **Allowed**: Read relevant `brain/` files and skill docs under `.agents/skills/`, run `./noodle` commands, compose prompts, spawn managers.
- **Forbidden**: Read source code, grep codebases, run builds/tests, write implementations.

**Exception — Self-Execute mode:** When triage selects self-execute, all tools are available.

### Only Bash: `./noodle` (in orchestration mode)

No ad-hoc bash when running managers. In self-execute mode, use bash freely.

### Guard the Context Window

Every status check costs tokens. Monitoring is the #1 source of context window bloat in director sessions.

The `status` and `wait` commands are context-safe by design. Use them exclusively:

```bash
# Block until all managers finish — replaces passive background monitoring
./noodle wait --session <id> --until all_done --timeout 30m

# Compact status check when needed (2-3 lines per agent, context-safe)
./noodle status --session <id>
```

Between `wait` calls, do nothing. The CLI watches autonomously. Your job is to wait, not watch.

### Permissions

All commands are pre-approved: `Bash(./noodle *)`. If you hit a permission prompt, you're running something you shouldn't.

## Before Starting

1. Read [references/soul.md](references/soul.md) — internalize it.
2. Read `brain/index.md` — project conventions.
3. When orchestrating CEO/CTO work, read `../ceo/SKILL.md` and `../cto/SKILL.md`
   for the canonical role contracts.

## Task Tracking

Use `TaskCreate` for each manager + post-completion steps. Mark `in_progress` when spawned, `completed` when merged. Check `TaskList` after each step.

**Interruptions are additive.** When the user interrupts you and asks for something new, add the new request to your existing task list — don't discard or restart the list. Treat interruptions as additional tasks, not replacements.

## Process

### 0. Triage

Assess the task and choose an execution mode before doing anything else.

**Self-execute** — Low complexity: brain files, configs, renames, up to 5 file changes. Do it yourself.

**Single manager (operator)** — One logical unit of work, no parallelism, <=5 files AND <=2 logical units. Spawn with `--agent operator` — the agent has no delegation tools, so direct execution is structural. Saves $2-3 per phase vs worker overhead.

**Single manager (with workers)** — One logical unit of work, no parallelism, but 6+ files or 3+ logical units where worker delegation pays for itself.

**Multi-manager** — Independent subtasks that benefit from parallelism, or disjoint areas.

| Signal | Self-execute | Single (operator) | Single (workers) | Multi-manager |
|--------|:---:|:---:|:---:|:---:|
| Files touched | 1-5 | 2-5 | 6-14 | 15+ or disjoint |
| Logical units | 1 | 1-2 | 3+ | Multiple disjoint |
| Needs code exploration | No | Maybe | Yes | Yes |
| Needs tests | Maybe | Yes | Yes | Yes |
| Parallelism opportunity | No | No | No | Yes |
| Complexity | Low | Low-Moderate | Moderate | High |

Self-execute tasks that modify up to 5 files and don't require deep code exploration. Run tests yourself if the change warrants it. This saves the 10+ turn overhead of spawning a manager.

When in doubt, choose the smaller scale — you can always spawn more managers later.

**Mixed-mode.** In multi-manager sessions, self-execute trivial subtasks (config edits, single-file-no-codegen) first, then spawn managers for the rest.

**Post-work finalization is always self-execute.** Committing, merging, and cleanup are mechanical — never spawn a recovery/finalize manager for git operations.

---

### Self-Execute Path

1. Use the `worktree` skill (unless brain-only)
2. Do the work directly
3. Commit with `commit` skill
4. Merge worktree back to main
5. Report to user

---

### Manager Path (single or multi)

For **single manager**, skip step 2 — the task IS the prompt.

### 1. Prepare

Build noodle CLI if missing: `[ -f ./noodle ] || (cd noodle && go build -o ../noodle . && cd ..)`

Initialize: `./noodle init --project "$(pwd)"` — capture session ID. Creates `~/.noodle/projects/<hash>/sessions/<id>/`.

### 2. Decompose

Break into **self-contained**, **non-overlapping** tasks. If tightly coupled, use one manager.

### 3. Compose Prompts

Read [references/manager-workflow.md](references/manager-workflow.md) for required prompt inclusions,
verification commands, and conditional gotchas.

**When to use `operator`:** Phases with <=5 files and <=2 logical units, verification/testing with light fixes, or when worker delegation overhead isn't justified. Spawn with `--agent operator` — the agent structurally cannot delegate (no Task, Skill, TeamCreate, TeamDelete, or SendMessage tools). No instruction-following required; the tools simply don't exist.

A vague prompt produces vague work. Share everything you know.

### 4. Spawn

Write prompt files to the session directory, then spawn:

```bash
# Standard manager (delegates to workers)
./noodle spawn --session <id> --name <name> --prompt-file <path>

# Direct manager (does all work itself, no delegation tools)
./noodle spawn --session <id> --name <name> --prompt-file <path> --agent operator
```

**What `spawn` handles automatically:**
- Validates prompt path is inside session directory (errors on violation)
- Auto-appends standard preamble/contract template (embedded in the `noodle` binary via `noodle/templates/`)
- Auto-generates context manifest (no separate Write call needed)
- Checks deps in worktree if `--worktree <path>` is provided
- Checks for orphaned prompt files in project root
- Registers agent in the agent tree with generated AgentID
- Launches tmux with the stamp pipeline (agent-id + event-dir for sidecar events)

**Prompt storage — mandatory rules:**

1. **NEVER write prompt files outside the session directory.** Use: `~/.noodle/projects/<hash>/sessions/<id>/mgr-<name>-prompt.md`
2. **Naming convention:** `mgr-<name>-prompt.md` — matches the manager name used in `--name`.
3. **`spawn` enforces this:** it errors (not warns) if `--prompt-file` points outside the session directory.

### 5. Monitor

Use `wait` to block until completion, and `status` for compact state checks:

```bash
# Block until all managers finish (run with run_in_background for long sessions)
./noodle wait --session <id> --until all_done --timeout 30m

# Or wait for failures to intervene early:
./noodle wait --session <id> --until failure_detected --timeout 5m

# Compact status check (context-safe: 2-3 lines per agent + summary)
./noodle status --session <id>
# phase0  manager  active   $1.23  --
# phase1  manager  exited   $0.78  --
# ALL DONE: false  TOTAL: $2.01

# Full agent tree with hierarchy
./noodle tree --session <id>
```

No raw state file reads. No `logs` in a loop. The CLI mechanically limits context consumption.

### 6. Detect and Recover

Use `status` to check state, `recover` for structured recovery:

```bash
# See what's recoverable
./noodle recover --session <id> --list

# Preview recovery context without acting
./noodle recover --session <id> --agent <agent-id> --dry-run

# Execute recovery (enforced: checks retry limit, does kill+resume, auto-composes context)
./noodle recover --session <id> --agent <agent-id>
```

Recovery is the ONLY path for restarting failed agents:
- Enforces retry limit (refuses after 3 — structurally impossible to over-retry)
- Does kill+resume (spawn-new is not a code path)
- Auto-composes resume context from the agent tree + sidecar events

If max retries exceeded, self-execute the remaining work.

### 7. Collect Reviews

Reviews are spawned incrementally as each manager exits. This step collects all review results.

**Lightweight review (Task subagent):** For most changes, the review subagent diffs and reports. Tell it which branch, the objective, and what to check (correct files, no surprises, commit message matches work). Use Sonnet (`model: "sonnet"`) — good cost/quality tradeoff for structural reviews. But even Sonnet can make false claims about cross-file relationships it hasn't fully traced. If a review flags "dead code" or claims something is unused, verify with Grep before acting.

**Full review (review manager):** Only for complex architectural or security changes — use Opus. See the review prompt checklist in [references/manager-workflow.md](references/manager-workflow.md). Cap at 3 review cycles.

**Test deduplication.** If the manager's worktree branch shows a passing test commit in `git log` (artifact), the review subagent does NOT re-run tests. But if the only evidence is a manager's text claim, re-run. See `brain/principles/trust-the-output-not-the-report.md`.

### 8. Proactive Coordination (status-driven)

During monitoring, coordinate by observing status and re-planning work directly:

- Use `status`, `worker-status`, and `tree` to detect manager questions/blockers.
- Re-prioritize by re-prompting or recovering the specific manager with updated context.
- Emergency stop all managers with:
  ```bash
  ./noodle kill --session <id> --all
  ```

### 9. Complete

1. If managers left uncommitted changes, **self-execute**: Task subagent to review diffs, then verify/commit/merge directly.
2. Run **Performance Review** — see the performance review protocol in [references/manager-workflow.md](references/manager-workflow.md).
3. **Reflection batching.** Reflect once after ALL managers complete and reviews are done — not after each manager. Capture cross-cutting patterns that span multiple managers' work. Use the `reflect` skill once, covering all learnings from the session.
4. **Session summary.** Generate a session summary:

   ```bash
   ./noodle summarize \
     --session <id> \
     --objective "<original user request>" \
     --learnings "<key takeaways from reflection>"
   ```

5. Summarize outcomes, costs, and efficiency suggestions to the user.
6. Spawn next wave if sequential work remains.
7. Delete completed plan files from `brain/plans/` and `brain/index.md`.
8. Commit plan deletion and brain updates (`commit` skill).

**Never clean up session directories.** NDJSON logs, state files, and summaries are the audit trail. They enable future debugging, cost analysis, and pattern discovery across sessions.

## Self-Improvement

You own this skill. Learnings go in the brain; skill mechanics go in these skill files directly.

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Every command returns exit 1, empty output | Dead shell — cwd inside removed worktree | Kill + resume with reorientation prompt |
| No NDJSON output | Auth or missing `claude` binary | Check `which claude` and `ANTHROPIC_API_KEY` |
| Permission stall | Manager hook timeout | Resume with `--dangerously-skip-permissions` |
| Workers fail in worktree | Missing deps | Use `spawn --worktree <path>` for auto dep checking |
| Git shows `.agents/` paths | `.claude/skills/` is a symlink | Use `.agents/` paths for git operations |
| Merge fails with dirty main | Concurrent sessions | `git stash` before merge, `git stash pop` after |

## Observability

CLI tool at `./noodle`. Source in `noodle/`. Build with `cd noodle && go build -o ../noodle .`

All state is built from the agent tree + sidecar events — no NDJSON re-parsing. Commands read from pre-structured data for instant results.

### Session-Scoped Directories

```
~/.noodle/projects/<project-hash>/sessions/<session-id>/
├── agent-tree.json                # Persistent agent hierarchy
├── events/                        # Sidecar event files per agent
│   └── <agent-id>.ndjson
├── inbox/                         # Agent messaging
│   └── <agent-id>/
│       └── msg-*.json
├── mgr-<name>.ndjson              # NDJSON log per manager
├── mgr-<name>-prompt.md           # Prompt file per manager
├── mgr-<name>.director-meta.json  # Manager metadata
├── monitor-state.json             # Aggregated state (written by monitor)
├── events.ndjson                  # Monitor event audit trail
├── costs.json                     # Auto-snapshotted on cleanup
├── summary.json                   # Session summary (generated by summarize)
└── .director-meta.json            # Session metadata
```

### Evolving the Server

The director never modifies this code directly. Spawn a manager to make changes.
