# Phase 00: Scaffold and Gates

Back to [[plans/112-codebase-simplification-audit/overview]]

## Goal

Stabilize verification and traceability infrastructure so later refactor phases run with reliable signal.

## Depends on

- None.

## Findings in scope

- `97-103`, `113`, `118`

## Changes

- Fix broken plan/todo links and ensure plan traceability is mechanically checkable.
- Move CI/test harness setup concerns to pre-test scaffolding (avoid runtime network installs inside smoke flow).
- Define and wire cross-platform matrix expectations for program closure.
- Align CI to canonical project checks to reduce drift.

## Data structures

- Program verification gate checklist (phase, command, pass/fail artifact).
- Planning traceability map format used by later phases.

## Done when

- Broken links in planning entrypoints are resolved.
- CI/check flow uses canonical check commands and no duplicate drift paths.
- Cross-platform matrix expectations are encoded in workflow definitions.
- Program traceability format is available for phase closure reporting.

## Verification

### Static
- `pnpm check`
- `go test ./... && go vet ./...`
- `pnpm test:smoke`

### Runtime
- Run smoke pipeline once in a clean environment and once in a warm-cache environment; both must pass.

## Rollback

- Revert workflow/tooling edits as one isolated commit group.
- Preserve pre-phase workflow file snapshots to restore quickly if CI signal degrades.
