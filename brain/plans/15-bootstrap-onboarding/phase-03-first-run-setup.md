Back to [[plans/15-bootstrap-onboarding/overview]]

# Phase 3: First-Run Setup in `noodle start`

## Goal

`noodle start` detects a fresh project (no `.noodle.toml`, no `brain/`) and scaffolds the minimum structure needed to run. No separate `init` command — first run is just a special case of self-heal.

## Changes

- **`startup/firstrun.go`** (new) — `EnsureProjectStructure(projectDir string) error`
  1. Create `brain/` directory structure if missing: `brain/index.md`, `brain/todos.md` (starter template with section headers and `<!-- next-id: 1 -->`), `brain/principles.md` (empty index), `brain/plans/index.md`
  2. Create `.noodle/` directory if missing (runtime state directory)
  3. Generate minimal `.noodle.toml` if missing — just enough to start: `[routing.defaults]` with provider/model, `[skills]` with default paths, `[autonomy]` set to `"review"`. Adapter config left empty — the agent sets this up later guided by the noodle skill.
  4. Log what was created so the user/agent can see what happened

- **`cmd_start.go`** (or equivalent) — call `EnsureProjectStructure` before the existing config load and validation. If anything was scaffolded, log it and continue normally.

**Idempotency**: every check is "if not exists, create." Safe to run on already-configured projects — no-op.

**What this does NOT do** (agent handles these):
- Skill installation — agent reads INSTALL.md and copies skills
- Adapter setup — agent reads noodle skill and configures adapters
- Brain injection hook — agent reads noodle skill and installs hooks
- Principle copying — agent offers this when setting up the brain

## Data Structures

- No new types needed. Uses existing `config.Config` and filesystem operations.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical directory/file creation with clear spec |

## Verification

### Static
- `go test ./startup/...` — unit tests for `EnsureProjectStructure`
- Test fresh directory: all expected files created
- Test existing project: no files overwritten
- Generated `.noodle.toml` passes `config.Parse()`

### Runtime
- Run `noodle start --once` in an empty temp directory — should not crash, should create structure and report diagnostics for missing adapters/skills
- Run again — no files recreated, same behavior
