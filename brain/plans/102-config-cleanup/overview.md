---
id: 102-config-cleanup
created: 2026-03-01
status: active
---

# Config Surface Cleanup (Todos 102-107)

Back to [[plans/index]]

## Context

Noodle's `.noodle.toml` exposes config knobs that are either unused, redundant with scheduler decisions, or internal implementation details. Before launch, shrink the config surface to only what users genuinely need to control.

## Scope

**In scope (6 todos):**

- **102** — Remove `routing.tags.*` from config. Scheduler decides routing per stage.
- **103** — Remove `[recovery]` section (`max_retries`). Scheduler handles retry decisions via events.
- **104** — Remove `stuck_threshold` entirely; hardcode `poll_interval` to 1s.
- **105** — Remove ticket staleness tracking. Scheduler handles stuck stages via events.
- **106** — Rename `max_cooks` to `max_concurrency` in config and all Go code.
- **107** — Remove `max_completion_overflow`, `merge_backpressure_threshold`, `shutdown_timeout` from config. Hardcode sensible defaults. Shutdown: SIGTERM + 2s hard deadline.

**Out of scope:**
- `routing.defaults` stays — users set their default provider/model
- `[adapters]`, `[skills]`, `[agents]`, `[runtime]`, `[server]` sections untouched
- No new config fields added

## Constraints

- **No backward compatibility, but loud failure.** Per CLAUDE.md: no `omitempty` shims, no legacy fallbacks. Phase 1 establishes a removed-key check at the parse boundary — old `.noodle.toml` files with removed fields produce a clear parse error naming the removed field, not silent acceptance.
- **Error messages describe failure state.** Per CLAUDE.md: "session not found", not "session must exist."
- **Boundary discipline.** Config validation stays at the parse boundary in `config/parse.go`. Hardcoded defaults live where the value is consumed.

### Alternatives considered

**Approach A (chosen): Sequential per-todo phases.** Each todo is one phase. Clean Git history, each phase independently shippable.

**Approach B: Batch config struct changes.** One big phase for all struct changes, another for all logic changes. Fewer files touched per phase but harder to review and revert.

**Approach C: Group by file.** One phase per affected file. Maximizes file locality but mixes unrelated semantic changes.

Approach A wins because each todo has different risk profiles (unused field deletion vs behavior change) and sequential phases give clear rollback points.

## Applicable Skills

- `go-best-practices` — Go conventions for the implementation phases
- `testing` — Test updates and new test cases
- `noodle` — Project conventions and config patterns

## Phases

1. [[plans/102-config-cleanup/phase-01-remove-recovery]]
2. [[plans/102-config-cleanup/phase-02-remove-ticket-staleness]]
3. [[plans/102-config-cleanup/phase-03-remove-stuck-detection]]
4. [[plans/102-config-cleanup/phase-04-remove-routing-tags]]
5. [[plans/102-config-cleanup/phase-05-remove-monitor-section]]
6. [[plans/102-config-cleanup/phase-06-rename-max-cooks]]
7. [[plans/102-config-cleanup/phase-07-hardcode-concurrency-internals]]
8. [[plans/102-config-cleanup/phase-08-shutdown-behavior]]

## Verification

**Per-phase gate:** `pnpm check` (runs the full suite including build, vet, lint, and tests). This is the primary verification for every phase — `go build`/`go vet` alone are insufficient as they miss docs, schema, and UI integration.

After all phases:

```sh
pnpm check          # full suite — required gate
```

**E2e smoke test (after all phases):** `noodle start` → enqueue an order → cook runs to completion → `Ctrl-C` exits within ~2s. This proves the full config-parse → loop → cook → shutdown path works end-to-end with the reduced config surface.

Config surface should be:
- `[routing.defaults]` — provider, model
- `[concurrency]` — max_concurrency (renamed from max_cooks)
- `[adapters]`, `[skills]`, `[agents]`, `[runtime]`, `[server]` — unchanged

Removed sections: `[recovery]`, `[monitor]`, `routing.tags.*`
Removed fields from `[concurrency]`: max_completion_overflow, merge_backpressure_threshold, shutdown_timeout
