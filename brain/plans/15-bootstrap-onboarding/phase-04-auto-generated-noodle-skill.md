Back to [[plans/15-bootstrap-onboarding/overview]]

# Phase 4: Auto-Generated Noodle Skill

## Goal

The `noodle` skill becomes a comprehensive, auto-generated reference that teaches agents everything about configuring and operating Noodle. A CI test fails if the skill drifts from the source code — like snapshot tests for fixtures.

Use the `skill-creator` skill when implementing this phase.

## Changes

- **`generate/skill_noodle.go`** (new) — generator that produces `SKILL.md` content from:
  1. **Config schema** — reflect over `config.Config` struct, extract field names, types, defaults, TOML keys, and doc comments. Render as a config reference table.
  2. **CLI commands** — extract from command registrations (cobra or equivalent). Render as a command reference table.
  3. **Prose sections** — stored as Go template strings in the generator. These are human-authored but live in Go code so the generator is the single source:
     - Adapter deep dive: skill+scripts pattern, NDJSON contract, guides for markdown, GitHub Issues, Linear
     - Hook installation: brain injection script, auto-index script, how to add to `.claude/settings.json`
     - Troubleshooting: common issues, `noodle debug` usage

- **`generate/skill_noodle_test.go`** (new) — snapshot test:
  1. Run the generator
  2. Read the committed `.agents/skills/noodle/SKILL.md`
  3. Diff — if they don't match, fail with a message like: "noodle skill is out of date. Run `go generate ./generate/...` to update."

- **`.agents/skills/noodle/SKILL.md`** — now generated, not hand-edited. Contains:
  1. What is Noodle — one-paragraph overview
  2. Config reference (`.noodle.toml`) — every field with type, default, description
  3. CLI commands — reference table
  4. Adapter setup — how to wire markdown, GitHub Issues, Linear, custom
  5. Hook installation — brain injection and auto-index scripts with copy-paste commands
  6. Skill management — how to install/override skills, search path precedence
  7. Troubleshooting — diagnostics, `noodle debug`, common fixes

- **`.agents/skills/noodle/references/`** — delete. Everything lives in SKILL.md now.

- **`Makefile` or `go generate`** — add a generate target: `go generate ./generate/...` that writes the skill file.

## Data Structures

- Template data struct: `SkillData` with `ConfigFields []ConfigField`, `Commands []Command`, and prose section strings.
- `ConfigField` — `TOMLKey string`, `Type string`, `Default string`, `Description string`, `Section string`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Designing the generator template and prose sections requires judgment |

## Verification

### Static
- `go test ./generate/...` — snapshot test passes (generated == committed)
- Skill resolves: `noodle skills list` shows `noodle`
- Every field in `config.Config` appears in the generated skill
- Every CLI command appears in the generated skill

### Runtime
- Change a config field default in `config.go`, run tests — snapshot test fails
- Run `go generate`, re-run tests — passes
- Give the generated skill to an agent, ask it to create a `.noodle.toml` — produces valid config
