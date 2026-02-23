Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 2: Native Planning — Remove Plan Adapter

## Goal

Replace the plan adapter with minimal Go code that reads `brain/plans/` and surfaces plan metadata in the mise brief. Add native CLI commands for plan CRUD. The plan format becomes Noodle-owned, not user-configurable.

## Current State

- `[adapters.plans]` in `noodle.toml` configures shell scripts for sync/create/done/phase-add
- `.noodle/adapters/main.go` implements plan operations (invoked as external process)
- `adapter.Runner.SyncPlans()` runs sync script, parses NDJSON — only caller: `mise/builder.go`
- Plan mutation commands (create, done, phase-add) are only called from skills

## Changes

### Native plan reader

Create a Go package that reads `brain/plans/` directly:
- Parse `brain/plans/index.md` for plan discovery (wikilink format)
- Parse `overview.md` YAML frontmatter (id, status, created)
- List phase files in each plan directory
- Return minimal metadata — the prioritize skill reads actual plan files for deeper analysis

### Native CLI commands

Move plan mutation logic from `.noodle/adapters/main.go` into the Go binary:
- `noodle plan create <todo-id> <slug>` — create plan directory with overview + phase-01
- `noodle plan done <id>` — set status to done in frontmatter
- `noodle plan phase-add <todo-id> "Phase Name"` — add numbered phase file

### Remove plan adapter

- Remove `[adapters.plans]` from config schema and `config.defaultAdapters()`
- Remove plan-related code from `.noodle/adapters/main.go`
- Remove `adapter.Runner.SyncPlans()` — replaced by native reader
- Update `mise/builder.go` to call native reader instead of adapter sync

### Quality verdict ingestion

Add `.noodle/quality/` reading to the mise builder so the prioritize skill can see rejection history. Include verdict summaries (accept/reject, session ID, feedback) in the brief.

## Data Structures

- Plan metadata in brief: `{ id, status, created, directory, phase_files }`
- Quality verdict in brief: `{ session_id, accept, feedback, timestamp }`
- Plan format unchanged: `brain/plans/NN-slug/overview.md` + phase files

## Verification

- `make ci` passes
- Plan parsing tests cover: index parsing, frontmatter extraction
- `noodle plan create` creates plan directory with overview and phase-01
- `noodle plan done <ID>` sets status to done
- `noodle mise` produces brief with plan metadata (no adapter sync)
- Mise brief includes quality verdict history from `.noodle/quality/`
- No `[adapters.plans]` in config schema
