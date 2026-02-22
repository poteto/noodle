Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 2 — Config System

## Goal

Parse `noodle.toml` into a validated Go struct that the rest of the system reads. The config is the single source of truth for all Noodle wiring — skill paths, routing, adapter declarations, sous-chef settings, recovery policy, monitoring thresholds.

No `noodle init` command — project setup is handled by the bootstrap skill (Phase 14), which is the entry point for new users. The Go binary is pure runtime infrastructure; it reads config, it doesn't create it.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Build exactly what's described. No extra abstractions, no premature generalization, no "just in case" code.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **No extreme concurrency patterns.** Use straightforward goroutines and mutexes where needed. No complex channel orchestration or worker pools unless the phase explicitly calls for them.
- **Tests should increase confidence, not coverage.** Test the critical flows that would break silently if wrong. Skip testing implementation details, trivial getters, or obvious wiring. Ask: "does this test help us iterate faster?"

- **End-of-phase Claude review (required).** After implementing this phase, write the review prompt to a temp file and run a non-interactive Claude review with tools disabled and bypassed permissions, for example: `prompt_file="$(mktemp)"; printf '%s\n' "Review the changes for this phase. Report risks, regressions, and missing tests." > "$prompt_file"; claude -p --dangerously-skip-permissions --tools "" -- "$(cat "$prompt_file")" | tee .noodle/reviews/<phase-id>-review.log; rm -f "$prompt_file"`.
- **Observe Claude liveness in global logs while it runs.** Check the global `~/.claude` directory (project session `.jsonl` logs under `~/.claude/projects/`) and watch the active session log; as long as new log entries are being written, Claude is still working.
- **Stall criteria + completion gate.** Treat the review as stalled only when the active global `~/.claude/projects/` session log stops changing for more than 180s *and* the Claude process is still alive. Do not mark the phase complete until the Claude command exits and the global log contains the final assistant output, with blocking findings addressed.
- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

- **`config/`** — New package. Config types, TOML parsing, validation, defaults.
- **`command_catalog.go`** — Config loading integrated into command startup (read `noodle.toml` if present, fall back to defaults if not).

## Data Structures

- `Config` — Top-level config struct. Fields:
  - `Phases` — Map of phase name → skill name (quality, oops, product-review overrides)
  - `Adapters` — Map of adapter name → `AdapterConfig` (skill + scripts map)
  - `SousChef` — Run frequency, model
  - `Routing` — Default provider/model + tag-based overrides
  - `Skills` — Ordered list of skill search paths
  - `Review` — Whether taster review runs after each cook by default
  - `Recovery` — Max retries, retry suffix pattern
  - `Monitor` — Stuck threshold, ticket stale timeout, poll interval
  - `Concurrency` — Max concurrent cooks
  - `Agents` — Paths to agent CLI directories (`.claude/`, `.codex/`). Set by bootstrap based on platform.
- `AdapterConfig` — Skill name + scripts map (action name → command string)
- `RoutingConfig` — Defaults + tag overrides, each a `ModelPolicy`
- `ConfigDiagnostic` — A validation issue. Fields: field path, message, severity (`repairable` or `fatal`). Repairable issues can be fixed by a cook (missing adapter scripts, unknown skill names, stale paths). Fatal issues prevent spawning entirely (agent CLI not found, no provider configured, tmux missing).
- `ValidationResult` — List of `ConfigDiagnostic`. Methods: `CanSpawn() bool` (true if no fatal issues), `Repairables() []ConfigDiagnostic`, `Fatals() []ConfigDiagnostic`

## Default Config Contract

Every field has a concrete default. If `noodle.toml` is missing or a field is omitted, these values apply:

| Field | Default | Notes |
|-------|---------|-------|
| `phases.oops` | `"oops"` | Skill for fixing user's project infrastructure |
| `phases.debugging` | `"debugging"` | Debugging methodology, loaded by oops and repair |
| `skills.paths` | `["skills", "~/.noodle/skills"]` | Project (committed) > user |
| `sous-chef.run` | `"after-each"` | Run after each cook completes |
| `sous-chef.model` | `"claude-sonnet"` | Cheaper model for prioritization |
| `routing.defaults.provider` | `"claude"` | |
| `routing.defaults.model` | `"claude-sonnet-4-6"` | |
| `adapters.backlog.scripts.sync` | `".noodle/adapters/backlog-sync"` | |
| `adapters.backlog.scripts.add` | `".noodle/adapters/backlog-add"` | |
| `adapters.backlog.scripts.done` | `".noodle/adapters/backlog-done"` | |
| `adapters.backlog.scripts.edit` | `".noodle/adapters/backlog-edit"` | |
| `adapters.plans.scripts.sync` | `".noodle/adapters/plans-sync"` | |
| `adapters.plans.scripts.create` | `".noodle/adapters/plan-create"` | |
| `adapters.plans.scripts.done` | `".noodle/adapters/plan-done"` | |
| `adapters.plans.scripts.phase-add` | `".noodle/adapters/plan-phase-add"` | |
| `review.enabled` | `true` | Taster review runs after every cook by default |
| `recovery.max_retries` | `3` | Per backlog item |
| `monitor.stuck_threshold` | `"120s"` | No activity → flagged stuck |
| `monitor.ticket_stale` | `"30m"` | No progress → ticket abandoned |
| `monitor.poll_interval` | `"5s"` | Fallback when fsnotify unavailable |
| `concurrency.max_cooks` | `4` | Hard cap on simultaneous cooks |
| `agents.claude_dir` | `""` | Path to `.claude/` directory. Set by bootstrap. |
| `agents.codex_dir` | `""` | Path to `.codex/` directory. Set by bootstrap. |

**Adapters are optional.** If `[adapters.plans]` is omitted entirely from config, no plans adapter is loaded — the mise produces an empty `plans` array and the sous chef schedules from backlog only. Similarly, `[adapters.backlog]` can be omitted (empty mise — nothing to schedule). The defaults above apply when running with no config file at all (the default-brain workflow). Projects that use GitHub Issues or Linear typically replace the backlog adapter section and remove the plans adapter entirely.

**Missing adapter scripts are warnings, not fatal.** If an adapter is configured but its sync script doesn't exist, the mise builder logs a warning and produces an empty array for that adapter — it doesn't block `noodle start`. The repair cook can create the missing script. This means a half-configured adapter degrades gracefully rather than preventing all work.

## Validation + Self-Healing

Config validation produces a `ValidationResult` classifying every issue as repairable or fatal. All commands validate config and report diagnostics, but **only `noodle start` triggers repair cooks**. Read-only commands (`noodle status`, `noodle skills list`, etc.) validate and report but never mutate.

**Read-only command behavior:** Read-only commands only fail on issues that prevent *them* from working — missing or unparseable config, unreadable state files. They do NOT fail on spawn-only dependencies (missing tmux, missing agent CLI directory, no provider configured). A user should always be able to run `noodle status` even if the spawner isn't set up yet. Spawn-only diagnostics are shown as warnings, not errors.

The `noodle start` startup sequence:

1. Parse and validate config → `ValidationResult`
2. If no issues → proceed normally
3. If repairable issues only (and spawner is functional) → spawn a repair cook with the `noodle` meta-skill, passing the diagnostics as context. The cook fixes what it can (creates missing adapter scripts, corrects paths, installs missing skills). Re-validate after repair.
4. If fatal issues → print each diagnostic with: the field path, what's wrong, what the expected state is, and the exact command or edit to fix it. Exit non-zero.

Fatal issues are those that prevent spawning: `agents.claude_dir` pointing to a nonexistent directory, no provider configured, tmux not installed. Everything else is repairable.

Error messages must be actionable. Not `"invalid config"` — instead: `"agents.claude_dir: directory '/home/user/.claude' not found. Run the bootstrap skill or set agents.claude_dir to the path where Claude Code stores its configuration."`.

### Agent CLI path resolution

When `agents.claude_dir` or `agents.codex_dir` is empty (not set by bootstrap), the spawner falls back to searching `PATH` for the `claude` or `codex` binary. The resolution order: config path → PATH lookup → fatal error with install instructions.

### Review precedence

The `review.enabled` config field sets the global default. Per-item `review` annotations from the sous chef override the global default. Precedence: **per-item annotation > global config**. If `review.enabled = false` globally but the sous chef annotates `review: true` on a specific item, the taster runs for that item.

## Verification

### Static
- `go test ./config/...` — Round-trip parse tests (TOML → Config → validate)
- Config with missing optional fields uses defaults from the table above
- Config with invalid values (unknown provider, bad run frequency, negative threshold) returns clear errors at parse boundary
- Missing config file results in valid default Config (not an error)
- Every default value has a test asserting its exact value
- Validation classifies issues correctly: missing adapter script → repairable, missing agent dir → fatal
- Fatal diagnostics include field path, description, and fix instructions

### Runtime
- Binary starts cleanly with no `noodle.toml` (uses defaults)
- Binary starts cleanly with a full config (all fields parsed)
- Config with repairable issue + functional spawner → repair cook spawned, issue fixed, startup continues
- Config with fatal issue → detailed error message printed, exit 1
