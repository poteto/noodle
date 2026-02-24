Back to [[archived_plans/01-noodle-extensible-skill-layering/overview]]

# Phase 8 — Adapter Runner + Mise

## Goal

Implement the adapter pattern and mise construction. Adapters are the bridge between the user's task/plan system and Noodle's runtime. Each adapter declares a set of scripts — one per action (sync, add, done, edit) — that the Go binary calls mechanically. Scripts are commands, not files — they can be shell scripts, Python scripts, compiled binaries, or inline commands like `gh issue close`. The user's agent is responsible for creating adapter scripts that transform their system into Noodle's required format.

The mise ("mise en place") gathers all state into a structured brief for the sous chef. Worktree creation remains automatic runtime behavior in the loop/spawner path — operators should not manually prepare worktrees for each item.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Build exactly what's described. No extra abstractions, no premature generalization, no "just in case" code.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **No extreme concurrency patterns.** Use straightforward goroutines and mutexes where needed. No complex channel orchestration or worker pools unless the phase explicitly calls for them.
- **Tests should increase confidence, not coverage.** Test the critical flows that would break silently if wrong. Skip testing implementation details, trivial getters, or obvious wiring. Ask: "does this test help us iterate faster?"
- **Prefer markdown fixture tests for parser/protocol flows.** Keep input and expected output in the same anonymized `*.fixture.md` file under `testdata/`, and use this pattern wherever practical.


- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

- **`adapter/`** — New package. Runs adapter scripts declared in config, validates output against schemas. Each action has a defined input/output contract.
- **`mise/`** — New package. Gathers state from all sources into a structured brief.
- **`cmd_mise.go`** — `noodle mise` command. Outputs the full gathered brief as JSON for debugging and sous chef inspection.
- **`command_catalog.go`** — Register `mise` command.

## Adapter Scripts

Each adapter declares scripts for each action. Scripts are commands executed as subprocesses, exactly like `package.json` scripts. They can be anything executable.

Commands run via `sh -c` on all platforms. Windows users run Noodle under WSL (see Constraints in overview), so the execution model is always Unix. The default adapter scripts that ship with Noodle are POSIX shell. User-written adapter scripts can be anything executable under `sh -c` — Python, Go, compiled binaries, or inline commands like `gh issue close`.

### Backlog adapter actions

| Action | Input | Output | Description |
|--------|-------|--------|-------------|
| `sync` | none | `BacklogItem` NDJSON to stdout | List all backlog items |
| `add` | JSON on stdin: `{title, description, section, tags}` | Created item ID to stdout | Create a new item |
| `done` | `$ID` as first argument | none | Mark item as complete |
| `edit` | `$ID` as first argument, JSON on stdin: `{title?, description?, section?, tags?}` | none | Update item fields |

### Plans adapter actions

| Action | Input | Output | Description |
|--------|-------|--------|-------------|
| `sync` | none | `PlanItem` NDJSON to stdout | List all plans with phase status |
| `create` | JSON on stdin: `{title, slug}` | Created plan ID to stdout | Create a new plan |
| `done` | `$ID` as first argument | none | Mark plan as complete |
| `phase-add` | `$ID` as first argument, JSON on stdin: `{name}` | none | Add a phase to a plan |

### Invocation contract

Scripts are executed as shell commands — the runner composes the full command string and passes it to the shell:

- **Unix:** `sh -c "<command> <args>"`
- **Windows (WSL):** Same as Unix (WSL provides `sh`)

Arguments are appended to the command string by the runner with proper shell quoting. For `done` and `edit` actions, `$ID` is the first argument:

```
# Config: done = "gh issue close"
# Invocation: sh -c 'gh issue close "42"'

# Config: done = ".noodle/adapters/backlog-done"
# Invocation: sh -c '.noodle/adapters/backlog-done "42"'
```

The runner handles quoting — IDs with spaces, special characters, or Unicode are properly escaped. Script authors don't need to worry about quoting in their config declarations; the runner guarantees that `$1` in a shell script receives the exact ID string. Stdin payloads are raw JSON (not shell-encoded).

### Config shape

```toml
[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = ".noodle/adapters/backlog-sync"
add = ".noodle/adapters/backlog-add"
done = "gh issue close"
edit = ".noodle/adapters/backlog-edit"

[adapters.plans]
skill = "plans"

[adapters.plans.scripts]
sync = "python .noodle/adapters/sync-plans.py"
create = ".noodle/adapters/plan-create"
done = ".noodle/adapters/plan-done"
phase-add = ".noodle/adapters/plan-phase-add"
```

## Normalized Schemas

These are the contracts that all adapter sync scripts must produce. Required fields are marked with `*`. These schemas are the interface between the user's system and Noodle's runtime — everything downstream (mise, sous chef, loop) operates on these types.

### BacklogItem

```json
{
  "id": "42",                    // * unique identifier (string)
  "title": "Fix login bug",     // * short description
  "description": "The auth token refresh has a race condition...",
  "section": "bugs",            //   grouping/category
  "status": "open",             // * "open" | "in_progress" | "done"
  "tags": ["frontend", "auth"], //   routing/filtering tags (default [])
  "plan": "03",                 //   linked plan ID
  "plan_phase": "phase-02",     //   current active phase within linked plan
  "estimate": "medium"          //   size hint: "small" | "medium" | "large"
}
```

**How tags drive routing:** Tags are the primary mechanism for routing items to providers. The `[routing.tags]` config maps tag names to provider/model policies. When the sous chef sees `tags: ["frontend"]` and the config has `frontend = { provider = "claude", model = "opus" }`, it knows to recommend Claude Opus for this item. Multiple tags are resolved by the sous chef — it weighs them against the routing config and makes a judgment call.

**Where tags come from:** The adapter sync script extracts tags from whatever the user's source format supports:

- `brain/todos.md` default: tags are parsed from `#tag` markers in the item description (e.g., `1. Fix login bug #frontend #auth`)
- GitHub Issues: labels map to tags
- Linear: labels or project tags
- Custom systems: whatever the user's sync script produces

The sous chef can also infer implicit tags from context — an item in the `## Frontend` section with no explicit tags might still get routed to Claude Opus based on the section name and description. This is judgment, not mechanical mapping.

### PlanItem

```json
{
  "id": "03",                    // * unique identifier (string)
  "title": "Auth Refactor",      // * short description
  "status": "active",            // * "draft" | "active" | "done"
  "phases": [                    // * ordered list of phases
    {"name": "scaffold", "status": "done"},
    {"name": "implement", "status": "active"},
    {"name": "verify", "status": "pending"}
  ],
  "tags": ["backend", "auth"]    //   routing tags (default [])
}
```

Phase status values: `"pending"` | `"active"` | `"done"` | `"skipped"`

When a backlog item links to a plan (`"plan": "03"`), the sous chef reads both the BacklogItem and the PlanItem to understand the full context — what phase is active, what's already done, how much remains. It uses this to make better routing and prioritization decisions.

### Validation rules

- `sync` output: each line must be valid JSON matching the schema. Missing required fields → reject line with error. Missing optional fields → use defaults (empty string, empty array).
- `add`/`create` output: a single line containing the created item's ID.
- `done`/`edit`/`phase-add`: exit code 0 = success, non-zero = failure. Stderr is captured for error reporting.

## Mise

The mise gathers state from all adapters plus internal Noodle state and writes it to `.noodle/mise.json`. This file is the single source of truth for the sous chef — it reads the file, reasons about priorities, and writes `.noodle/queue.json` in response.

### mise.json schema

```json
{
  "generated_at": "2026-02-21T18:30:00Z",
  "backlog": [
    {
      "id": "42", "title": "Fix login bug", "section": "bugs",
      "status": "open", "tags": ["frontend", "auth"],
      "plan": "03", "plan_phase": "phase-02", "estimate": "medium"
    }
  ],
  "plans": [
    {
      "id": "03", "title": "Auth Refactor", "status": "active",
      "phases": [
        {"name": "scaffold", "status": "done"},
        {"name": "implement", "status": "active"}
      ],
      "tags": ["backend"]
    }
  ],
  "active_cooks": [
    {
      "id": "fix-css-layout", "target": "41",
      "provider": "codex", "model": "gpt-5.3-codex",
      "status": "running", "cost": 0.45, "duration_s": 342
    }
  ],
  "tickets": [
    {
      "target": "41", "target_type": "backlog_item",
      "cook_id": "fix-css-layout", "status": "active",
      "files": ["src/styles/layout.css"]
    }
  ],
  "resources": {
    "max_cooks": 4, "active": 1, "available": 3
  },
  "recent_history": [
    {
      "target": "40", "cook_id": "update-readme",
      "outcome": "accepted", "cost": 0.12, "duration_s": 131
    }
  ],
  "routing": {
    "defaults": {"provider": "codex", "model": "gpt-5.3-codex"},
    "tags": {
      "frontend": {"provider": "claude", "model": "opus"},
      "tests": {"provider": "codex", "model": "spark"}
    }
  }
}
```

The `routing` section is copied from config so the sous chef can see the tag→provider mappings when making decisions. The sous chef doesn't need to read `noodle.toml` — everything it needs is in `mise.json`.

### Mise fields

- `backlog` — from backlog adapter `sync`. Only `open` and `in_progress` items (done items filtered out). Empty array if backlog adapter is not configured or sync script is missing.
- `plans` — from plans adapter `sync`. Only `draft` and `active` plans. Empty array if plans adapter is not configured or sync script is missing. Plans are optional — many teams use backlog only.
- `active_cooks` — from session state files (ID, target, provider, model, status, cost, duration)
- `tickets` — from `.noodle/tickets.json` (materialized by monitor)
- `resources` — from config + active cook count
- `recent_history` — last N completed cooks (target, outcome, cost, duration). Helps the sous chef avoid repeating failures and learn from patterns.
- `routing` — from config. Tag→provider mappings so the sous chef can make informed routing recommendations.

The mise is regenerated by the scheduling loop (Phase 9) after each cook completion and on a configurable interval. The `noodle mise` command also triggers a regeneration and outputs the result.

## Queue

The sous chef reads `mise.json` and writes `queue.json`. This is the complete transformation — from raw backlog state to prioritized, routed work items.

### queue.json schema

```json
{
  "generated_at": "2026-02-21T18:31:00Z",
  "items": [
    {
      "id": "42",
      "provider": "claude",
      "model": "claude-opus-4-6",
      "skill": "plans",
      "review": true,
      "rationale": "Auth refactor phase 2 — complex token logic, needs Opus. Linked plan has scaffold done, implement is next."
    },
    {
      "id": "43",
      "provider": "codex",
      "model": "gpt-5.3-codex",
      "review": false,
      "rationale": "Simple README update, no review needed. Codex handles docs well."
    }
  ]
}
```

### QueueItem fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | BacklogItem ID to work on |
| `provider` | string | yes | `"claude"` or `"codex"` |
| `model` | string | yes | Specific model ID (e.g., `"claude-opus-4-6"`, `"gpt-5.3-codex"`) |
| `skill` | string | no | Skill to load for this cook (default: none, cook uses its own judgment) |
| `review` | bool | no | Whether to run the taster after completion (default: from `review.enabled` config) |
| `rationale` | string | no | Why the sous chef chose this priority/routing. Visible in queue view. |

### Routing decision flow

```
BacklogItem tags                    [routing.tags] config
       │                                    │
       └──────────── sous chef ─────────────┘
                         │
              reads tags + config + context
              applies scheduling judgment
                         │
                    QueueItem
              (provider, model, review, skill)
                         │
              scheduling loop enforces
              hard constraints (tickets,
              concurrency, exclusivity)
                         │
                    SpawnRequest
              (concrete spawn parameters)
```

The sous chef is the only place where routing decisions are made. The Go loop never interprets tags or chooses providers — it mechanically reads the queue and spawns what the sous chef says. The sous chef can override tag-based routing if it has reason to (e.g., a "frontend" item that's mostly tests might still go to Codex).

### End-to-end example

**1. User writes a todo:**

```markdown
## Frontend

1. Fix login bug — auth token refresh race condition #auth #urgent [[plans/03-auth-refactor/overview]]
```

**2. `backlog-sync` parses it into BacklogItem NDJSON:**

```json
{"id":"1","title":"Fix login bug — auth token refresh race condition","status":"open","section":"Frontend","tags":["auth","urgent"],"plan":"03","plan_phase":"phase-02","estimate":""}
```

Tags `#auth` and `#urgent` are extracted and stripped from the title. The plan link is resolved — `plan_phase` is set to the first `active` phase from the linked PlanItem.

**3. Mise gathers state into `mise.json`:**

The BacklogItem appears in `backlog[]` alongside plans, active cooks, tickets, resources, and the routing config from `noodle.toml`.

**4. Sous chef reads `mise.json`, writes `queue.json`:**

The sous chef sees `tags: ["auth", "urgent"]` and `routing.tags.auth` is not configured, but the item links to a plan with complex token logic. It uses judgment:

```json
{"id":"1","provider":"claude","model":"claude-opus-4-6","skill":"plans","review":true,"rationale":"Complex auth refactor phase 2 — token race condition needs careful reasoning. Opus for correctness. Review required."}
```

**5. Loop reads queue, spawns cook:**

The loop validates constraints (no active ticket on item 1, concurrency under limit), automatically creates a dedicated worktree for the cook, and spawns a Claude Opus cook with the `plans` skill loaded and the backlog item context.

## Data Structures

- `AdapterRunner` — Executes scripts declared in config. Generic method: `Run(adapter, action string, args ...string)` with optional stdin. The runner looks up the command string from `config.Adapters[adapter].Scripts[action]` and executes it. Works for all adapter types (backlog and plans) and all actions (sync, add, done, edit, create, phase-add, etc.) without needing per-action Go methods.
- `BacklogItem` — See schema above
- `PlanItem` — See schema above
- `Mise` — Full brief struct
- `ResourceSnapshot` — Max concurrency, active count, available slots

## Verification

### Static
- `go test ./adapter/...` — Script runner tests with mock executables
- `go test ./mise/...` — Brief construction tests with fixture data
- Sync output with missing required field → clear error per line
- Sync output with missing optional field → defaults applied
- Script exit code non-zero → error with stderr captured
- Missing sync script for configured adapter → warning logged, empty array returned (not fatal)

### Runtime
- Default backlog adapter `sync` parses `brain/todos.md` → valid `BacklogItem` NDJSON
- Default plans adapter `sync` parses `brain/plans/` → valid `PlanItem` NDJSON
- Default backlog adapter `done 1` marks item 1 complete in `brain/todos.md`
- `noodle mise` outputs the full gathered brief as JSON
- Adapter contract test with GitHub Issues-shaped data: sync script outputs `BacklogItem` NDJSON with string IDs, labels as tags, milestone as section — validates against schema
- Mise with no plans adapter configured: `plans` array is empty, sous chef receives valid brief
- Mise with missing sync script: logs warning, produces empty array for that adapter (not an error)
- Mise-to-queue handoff does not require manual worktree setup; the downstream loop provisions the cook worktree automatically at spawn time
