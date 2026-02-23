---
id: 26
created: 2026-02-23
status: draft
---

# Context Usage Tracking

## Context

Noodle tracks cost (USD) per session but doesn't track context window efficiency — how much of the finite context budget each skill, prompt, and agent consumes, or whether sessions hit compression (a signal of waste). Cost tells you *that* something was expensive; context usage tells you *why* and what to fix.

We manually audited skill descriptions this session and saved ~550 tokens/session. That audit should be automated and data-driven.

## Scope

**In scope:**
- Capture compression events, per-turn context usage, and peak context from both Claude and Codex
- Attribute context usage to the active skill/task type
- Extend session metadata with context-specific fields
- Surface context metrics in mise.json for skill consumption
- Teach quality gate and meditate to flag context inefficiency
- Show context metrics in the TUI alongside existing cost display

**Out of scope:**
- Changing the context window size or model selection based on metrics (future optimization)
- Historical trend analysis across runs (future — requires persistent DB beyond session files)
- Modifying the Claude Code or Codex output format (we parse what they emit)

## Constraints

- **Extend, don't replace.** Build on existing `CanonicalEvent`, `SessionClaims`, `SessionMeta` types. No new data stores — enrich the session metadata file.
- **Both providers.** Claude and Codex adapters must both emit context metrics. Claude emits compression events natively; Codex may not — handle gracefully with zero values.
- **Skill attribution is best-effort.** `spawn.json` currently persists `skill`; phase 3 adds `task_key`. We still cannot detect mid-session skill switches in interactive mode. Per-session attribution is sufficient.

## Alternatives considered

1. **Separate metrics store** (`.noodle/metrics/`) — rejected per subtract-before-you-add. Session metadata already exists and is read by all consumers. Adding a second data source creates sync issues.
2. **Real-time context budget enforcement** — deferred. Useful but separate concern. This plan instruments; enforcement can build on it.
3. **Per-tool-call token tracking** — too granular. Per-turn is the right level: it maps to model API calls and is what providers report.

## Applicable skills

- `noodle` — CLI and config conventions
- `bubbletea-tui` — TUI component patterns for the metrics display phase
- `skill-creator` — for updating quality and meditate skill specs

## Phases

1. [[plans/26-context-usage-tracking/phase-01-extend-canonical-events-with-context-metrics]]
2. [[plans/26-context-usage-tracking/phase-02-enrich-session-metadata-with-context-fields]]
3. [[plans/26-context-usage-tracking/phase-03-add-skill-attribution-to-canonical-events]]
4. [[plans/26-context-usage-tracking/phase-04-surface-context-metrics-in-mise-json]]
5. [[plans/26-context-usage-tracking/phase-05-add-context-efficiency-to-quality-skill]]
6. [[plans/26-context-usage-tracking/phase-06-add-context-audit-to-meditate-skill]]
7. [[plans/26-context-usage-tracking/phase-07-surface-context-metrics-in-tui]]

Phases 1-4 are sequential (data model → aggregation → exposure). Phases 5-6 are independent and can run in parallel after phase 4. Phase 7 is blocked on [[plans/25-tui-revamp/overview]] (currently draft) — implement against the revamped TUI layout, not the current one.

## Verification

- `go test ./...` — all tests pass after every phase
- `go vet ./...` — no issues
- End-to-end: run a cook session, inspect `meta.json` for new context fields
