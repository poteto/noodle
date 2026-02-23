Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 9: Noodle-Native Planning — Remove Plan Adapter

## Goal

Make planning a first-class Noodle concept. Replace the plan adapter abstraction with minimal Go code that reads `brain/plans/` and surfaces plan data in the mise brief. The plan format (`overview.md` with YAML frontmatter, phase files, `index.md`) becomes Noodle-owned — not a user-configurable adapter.

The Go core stays lean: it reads plan files and exposes them as data. All scheduling intelligence (associations, phase ordering, status interpretation) stays in the prioritize skill. The backlog adapter stays (backlogs can be external: GitHub Issues, Linear, etc.). Only plans become native.

## Current State

- `[adapters.plans]` in `noodle.toml` configures shell scripts for sync/create/done/phase-add
- `.noodle/adapters/main.go` implements plan operations in Go (invoked as external process)
- `adapter.Runner.SyncPlans()` runs the sync script and parses NDJSON
- Only caller: `mise/builder.go:63` — no other Go code calls the plan adapter
- Plan mutation commands (create, done, phase-add) are only called from skills, not Go code

## Changes

### Native plan reader
- Create a Go package that reads `brain/plans/` directly:
  - Parse `brain/plans/index.md` for plan discovery (wikilink format)
  - Parse `overview.md` YAML frontmatter (id, status, created)
  - List phase files in each plan directory
  - Return minimal plan metadata for the mise brief — the prioritize skill reads the actual plan files for deeper analysis (phase status, associations, ordering)

### Native plan commands
- Move plan mutation logic from `.noodle/adapters/main.go` into the Go binary
- Add CLI commands: `noodle plan create`, `noodle plan done`, `noodle plan phase-add`
- These replace the adapter scripts — same operations, native execution

### Remove plan adapter
- Remove `[adapters.plans]` from config schema and defaults
- Remove plan-related scripts from `.noodle/adapters/main.go`
- Remove `adapter.Runner.SyncPlans()` — replaced by native reader
- Update `mise/builder.go` to call native reader instead of adapter sync

### Simplify plan task type

Remove `TaskKeyPlan` from the registry (`internal/taskreg/registry.go`). Planning is not a Noodle-spawned task — the user creates plans outside of Noodle using their own agent and the plan skill. The loop never spawns plan sessions, so the task type is dead code.

### Config update
- Remove `[adapters.plans]` section from `noodle.toml` and `config/config.go`
- Remove plan adapter entries from `config.defaultAdapters()`
- Plans are always `brain/plans/` — no config needed

## Data Structures

- Plan metadata in mise brief: `{ id, status, created, directory, phase_files: ["phase-01-foo.md", ...] }`
- Plan format unchanged: `brain/plans/NN-slug/overview.md` + phase files
- The prioritize skill reads plan files directly for deeper analysis — Go just surfaces what exists

## Verification

- `go build ./...` compiles with native plan reader
- `go test ./...` passes — plan parsing tests cover: index parsing, frontmatter extraction
- `noodle plan create` creates a plan directory with overview and phase-01 scaffold
- `noodle plan done <ID>` sets status to done in frontmatter
- `noodle mise` produces a mise brief with plan metadata (no adapter sync)
- No `[adapters.plans]` in config schema
- Prioritize skill can read plan files and compute associations from brief data
