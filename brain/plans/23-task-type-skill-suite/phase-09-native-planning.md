Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 9: Noodle-Native Planning — Remove Plan Adapter

## Goal

Make planning a first-class Noodle concept. Replace the plan adapter abstraction with native Go code that reads and writes `brain/plans/` directly. The plan format (`overview.md` with YAML frontmatter, phase files, `index.md`) becomes Noodle-owned — not a user-configurable adapter.

The backlog adapter stays (backlogs can be external: GitHub Issues, Linear, etc.). Only plans become native.

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
  - Parse phase checkbox lists from overview for phase status
  - Return `[]adapter.PlanItem` (reuse existing type)

### Native plan commands
- Move plan mutation logic from `.noodle/adapters/main.go` into the Go binary
- Add CLI commands: `noodle plan create`, `noodle plan done`, `noodle plan phase-add`
- These replace the adapter scripts — same operations, native execution

### Remove plan adapter
- Remove `[adapters.plans]` from config schema and defaults
- Remove plan-related scripts from `.noodle/adapters/main.go`
- Remove `adapter.Runner.SyncPlans()` — replaced by native reader
- Update `mise/builder.go` to call native reader instead of adapter sync

### Mise many-to-many association
- Compute todo↔plan associations in the mise brief
- A single todo can map to multiple plans (e.g. one backlog item spawns sequential plan phases)
- Multiple todos can map to a single plan (e.g. one plan addresses several related items)
- The prioritize skill uses this to determine: needs planning, ready for execution, or in-progress
- Association source: plan overview frontmatter `id` field + todo wikilinks to `[[plans/...]]`

### Config update
- Remove `[adapters.plans]` section from `noodle.toml` and `config/config.go`
- Plans are always `brain/plans/` — no config needed

## Data Structures

- `PlanItem` type stays as-is (or moves from `adapter/` to a more appropriate package)
- Plan format unchanged: `brain/plans/NN-slug/overview.md` + phase files

## Verification

- `go build ./...` compiles with native plan reader
- `go test ./...` passes — plan parsing tests cover: index parsing, frontmatter parsing, phase status extraction
- `noodle plan create` creates a plan directory with overview and phase-01 scaffold
- `noodle plan done <ID>` sets status to done in frontmatter
- `noodle mise` produces a mise brief with plans (no adapter sync)
- Mise brief includes todo↔plan associations (many-to-many)
- No `[adapters.plans]` in config schema
