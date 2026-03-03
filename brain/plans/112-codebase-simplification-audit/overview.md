---
id: 112
created: 2026-03-03
status: active
---

# Plan 112: Codebase Simplification Program (Root-Cause First)

Back to [[plans/index]]

## Context

A full-repo, multi-agent read-only audit was completed to identify simplification and refactor opportunities aligned to first-principles architecture, subtraction, boundary discipline, idempotency, and cross-platform correctness. This plan captures the complete findings inventory so execution can proceed deliberately.

## Scope

**In scope:**
- Consolidated findings across Go backend, loop/runtime/dispatcher, internal core packages, server/WebSocket, UI, docs, skills, automation, and brain structure.
- Executable phased program for resolving the findings with explicit sequencing, rollback, and verification gates.

**Out of scope:**
- Reprioritizing unrelated existing plans.
- Shipping speculative architecture not directly tied to root-cause findings in this inventory.

## Constraints

- No backward-compatibility shims by default.
- Cross-platform support (macOS/Linux/Windows) remains required.
- Error messages should describe failure states.
- Single-writer / idempotent behavior must be preserved for shared state files.

### Alternatives Considered

1. Triage only top 10 findings and defer the rest.
2. Capture all findings in one overview, then phase by root-cause cluster and dependency.
3. Split findings into separate plans per subsystem.

Chosen: **(2)** to preserve full audit signal in one canonical overview while phasing by root-cause cluster and dependency order.

## Applicable Skills

- `execute`
- `go-best-practices`
- `testing`
- `review`
- `ts-best-practices`
- `noodle`
- `skill-creator`
- `brain`

## Root-Cause Clusters

- `RC1` Contract fragmentation (`internal/state` vs `internal/orderx`, projection drift, reducer/ingest contract split). Findings: `48-63`.
- `RC2` Loop/control lifecycle ambiguity (non-deterministic stage failure, duplicated merge/failure orchestration, restart-idempotency holes). Findings: `35-47`.
- `RC3` Runtime/session lifecycle safety (termination error swallowing, recovery metadata parsing fragility, bounded shutdown gaps). Findings: `25-34`, `71-75`.
- `RC4` WS lifecycle split across backend and UI (channel close/send panic risk, subscriber leaks, duplicated UI connection managers). Findings: `69-70`, `76-88`.
- `RC5` CLI/config/adapter boundary drift (root pre-run behavior, duplicated defaults, shell-based adapter pathing). Findings: `1-24`.
- `RC6` Event pipeline inefficiency and silent drop behavior. Findings: `64-68`.
- `RC7` Docs/skills/brain/tooling instruction drift (category — multiple independent drift sources, not a single structural root cause). Findings: `89-119`.

## Severity Distribution

- **P0** (crash, data loss, or hang): `41`, `44`, `50`, `64`, `69`, `71`, `73`, `83` — 8 findings.
- **P1** (incorrect behavior, races, contract violations): `1`, `2`, `20`, `25`, `27`, `28`, `31`, `32`, `36`, `37`, `38`, `40`, `42`, `43`, `45`, `48`, `49`, `53`, `54`, `70`, `72`, `74`, `75`, `76`, `77`, `84`, `85`, `89`, `105`, `106` — 30 findings.
- **P2** (duplication, dead code, quality, hygiene): all remaining — 81 findings.

Within each phase, P0 findings must be resolved first. See per-phase priority sections.

## Effort Sizing

| Phase | Size | Rationale |
|---|---|---|
| `00` Scaffold | S | Link fixes, CI alignment, matrix definition |
| `01` CLI/Config/Adapter | M | 24 findings but mostly boundary cleanup, no model changes |
| `02` Contract Cutline | L | Core model unification + migration — highest risk phase |
| `03` Loop/Runtime Safety | L | 28 findings, lifecycle determinism, crash/restart hardening |
| `04` Server/UI WS | M | WS lifecycle convergence across Go + TS |
| `05` Event Pipeline Perf | S | 3 performance findings, isolated packages |
| `06` Docs/Skills/Brain | M | Many items but individually small, can parallel after `00` |

## Execution Policy

- **Serialization rule:** Phases `01-05` are shared-state core phases and must run sequentially (single active phase at a time).
- **Parallel allowance:** Phase `06` depends only on Phase `00` and may run in parallel with Phases `01-05`. Restricted to non-core files (`docs/`, `.agents/`, `brain/`), with no modifications to loop/runtime/internal contracts. Contract-dependent doc findings (`89`, `95`, `105`, `106`) should align with Phase `02` output if available; defer those items if Phase `02` has not landed.
- **Scaffold-first guard:** Phase `00` must complete before any other phase begins.

## Findings-to-Phase Traceability

| Finding IDs | Owner Phase | Rationale |
|---|---|---|
| `97-103`, `113`, `118` | `Phase 00` | Stabilize verification/test harness and planning traceability before refactors |
| `1-24` | `Phase 01` | Boundary cleanup for CLI/config/adapter surfaces (no downstream dependents) |
| `48-63`, `64`, `67` | `Phase 02` | Canonical contract cutline, internal model consolidation, and event routing correctness |
| `25-47`, `71-75` | `Phase 03` | Loop/runtime/control deterministic lifecycle + restart safety |
| `69-70`, `76-88` | `Phase 04` | Unified WS/server/UI transport lifecycle convergence |
| `65`, `66`, `68` | `Phase 05` | Event pipeline hot-path performance |
| `89-96`, `104-112`, `114-117`, `119` | `Phase 06` | Docs/skills/brain/tooling hygiene and drift removal |

## Rollback Strategy

- Every phase must land as independently revertible commits.
- If a phase fails runtime gates, rollback that phase fully before starting the next.
- `Phase 02` and `Phase 03` require an explicit state snapshot backup before rollout and a documented restore command path.
- No phase may change compatibility posture implicitly; any deliberate forward-only migration must be stated in-phase.

## Findings Inventory (Complete)

### Root CLI / command boundary

1. Root pre-run parses config for all commands; malformed config can block recovery commands.
2. Global `os.Chdir` dependency creates ambient state and test/order coupling.
3. `reportConfigDiagnostics` mixes rendering/policy/dispatch concerns.
4. Global mutable function vars used as test seams across command paths.
5. `runSkillsList` bypasses common resolver boundary.
6. Multiple commands write directly to `os.Stdout` rather than command-bound writers.
7. Browser-launch diagnostic envelope is computed but effectively discarded.
8. Repair worktree naming is timestamp-second based, weak for rapid retry idempotency.
9. CLI metadata is dual-sourced (`cmdmeta` vs command wiring), drift-prone.
10. `cmdmeta.Short` fails open on missing metadata.
11. Dead status helpers remain in command layer.
12. Event payload validation logic duplicated.

### Config / defaults / startup / generation

13. `pnpm` scripts include shell-specific operations unsafe on native Windows shells.
14. Removed config path (`routing.tags`) is still present in samples and tolerated with warning-only behavior.
15. Backlog adapter path is duplicated (shell scripts vs Go adapter command) with drift risk.
16. Defaults are duplicated across parser/default struct/startup/generator.
17. Startup integration test assumes Unix path semantics.
18. Generated-skill text maintenance is large and partially duplicated.

### Adapter / mise / examples

19. Adapter↔mise failure contract is stringly typed and duplicated.
20. `done` / `edit` backlog actions can silently succeed when target ID is missing.
21. Adapter execution model is shell-string based (`sh -c`) rather than argv typed.
22. Backlog item schema is permissive and relies on fragile JSON splicing.
23. `mise.Builder` combines IO, normalization, summarization, and persistence in one path.
24. Example schedule outputs remain overspecified vs current defaults.

### Dispatcher lifecycle / protocol

25. Session terminal classification can race canonical stream finalization.
26. Termination/kill error boundary is swallowed in lifecycle path.
27. Custom runtime prompt path is likely wrong (`req.Name` vs generated session path).
28. Custom runtime command path hardcodes `sh -c` and is non-portable.
29. Canonical parse helper drops malformed lines without surfacing diagnostics.
30. Runtime fallback logic appears duplicated/dead between factory and active paths.
31. Claude turn-end handling can wedge steering on error-result paths.
32. Pending steer messages can be dropped on write failure.
33. Provider branching logic is duplicated and defaults implicitly in multiple spots.
34. Sprites dispatcher/session paths have near-zero coverage and likely drift from process runtime behavior.

### Loop pipeline / control / reconciliation

35. `idle -> running -> idle` churn is structurally baked into cycle path.
36. Stage failure handling is non-deterministic by stage identity.
37. `normalizeOrders` retry path overlaps with repairs it cannot actually fix.
38. Orders file may be rewritten multiple times in a single cycle.
39. Schedule completion semantics are split across mutable flags and file checks.
40. Merge completion orchestration is duplicated across completion/control/reconcile paths.
41. `pending-review` entries can be dropped on restart when parked without active cook metadata.
42. Control command dedupe for commands without IDs is not restart-safe.
43. Remote reconcile branch existence checks local refs only.
44. Corrupt `pending-review.json` is treated as empty rather than surfaced.
45. Adopted-session liveness uses brittle string-matching against JSON.
46. Failure handling and summary/projection mapping are duplicated across paths.
47. Unused helper(s) and redundant candidate filtering remain in loop path.

### Internal core packages

48. Canonical model split across `internal/state` and `internal/orderx` with incompatible lifecycle/group semantics.
49. Event contract duplication between ingest types and reducer switch.
50. Reducer payload decode failures can become silent no-op behavior.
51. Reducer transition logic duplicated across handlers.
52. Ingest synchronization is over-layered for intended single-arbiter ownership.
53. `internal/projection` output shape does not align with live `orderx` contract.
54. `statever` startup guard can be inert if runtime marker writes are not part of normal flow.
55. Projection translation is duplicated across orders/status/snapshot paths.
56. Feed-event mapping relies on string literals instead of typed registry/exhaustiveness.
57. Snapshot fixture loading duplicates production mapping behavior.
58. `internal/dispatch` appears orphaned from production routing path.
59. `internal/rtcap` appears largely test-only/not enforced in live runtime selection.
60. `internal/mode` transition logic overlaps reducer logic.
61. `fixturedir` exposes dead/unused assertion helpers.
62. `taskreg.StageInput` contains unused breadth (`Title`) vs real callsite use.
63. Tiny single-use package candidate (`internal/recover`) could be folded.

### Event / parse / stamp

64. Unknown-provider event routing can silently lose events.
65. Parse/stamp hot path repeatedly unmarshals/remarshals full payload lines.
66. Ticket materialization uses full scans/global sorting and linear key resolution.
67. Provider tool/action normalization is duplicated and can drift.
68. Loop event truncation path rewrites/scans frequently.

### Runtime / monitor / worktree / server

69. WS snapshot broadcast has `send on closed channel` panic risk.
70. WS subscriber lifecycle can leak stale subscriptions after disconnect.
71. Shutdown can block indefinitely on merge/session termination.
72. Recovery/adoption paths rely on string-based session status parsing/mutation.
73. Recovery can skip live sessions when `meta.json` is missing/corrupt.
74. Maintenance behavior is inconsistent: malformed session metadata can fail loop.
75. Worktree CWD safety check uses raw string prefix/equality, weak cross-platform.

### UI client / routing / components

76. WS lifecycle duplicated per hook consumer; multi-connection and teardown races likely.
77. Control fallback can stall on WS timeout path before HTTP fallback.
78. Session-event append path sorts full event list on each event.
79. Channel↔route mapping duplicated and stringly typed across layers.
80. Schedule/bootstrap heuristics duplicated across sidebar/feed components.
81. Session status remains broad/stringly typed in multiple UI branches.
82. CSS surface is highly centralized/large, increasing refactor cost.
83. ReviewList stale selected index can crash report/actions path.
84. TreeView stage routing may misroute schedule stages.
85. React keying strategy for trees/feed rows is fragile for collision/reuse.
86. Markdown rendering pipeline duplicated between static and streaming components.
87. Tree hierarchy build does repeated linear session lookups.
88. Feed keyboard handler rebinds global listener on every snapshot dependency change.

### Docs / README / changelog

89. Orders/stages docs are internally contradictory (new contract vs old examples).
90. Generated docs build/cache artifacts are tracked in repo.
91. Onboarding docs conflict on whether `brain/` is first-run scaffolded vs optional.
92. Docs host/version sources are inconsistent/hardcoded in multiple places.
93. User docs include agent-internal instruction style that should be relocated.
94. Self-learning guidance is duplicated across multiple docs pages.
95. Adapter contract clarity is split across pages and link targets.
96. CLI docs include redundant command forms.

### Scripts / e2e / CI / release tooling

97. E2E smoke depends on live network installs during test execution.
98. CI is Linux-only while product targets cross-platform behavior.
99. Multi-layer retry behavior masks flake root causes and adds runtime.
100. Fixed ports increase collision risk in shared environments.
101. CI duplicates checks instead of canonical `pnpm check`.
102. Release script commit behavior can conflict with commit-msg policy.
103. `sandbox.sh` is brittle (identity/temp-root assumptions).
104. Tooling contains likely dead artifacts (watch/prototype surface drift).

### Skills / hooks / instruction set

105. Noodle authoring examples conflict with current scheduler contract (`do`/`with`, runtime `process`).
106. `skill-creator` validator contract conflicts with existing scheduled skills (`schedule` frontmatter).
107. `brain/index.md` ownership conflicts (manual edits vs auto-index hook single-writer intent).
108. Workspace isolation policy conflicts between commit/execute/worktree guidance.
109. Stale references to retired/nonexistent skills/paths remain in active instructions.
110. Path convention drift (`.claude/*` vs `.agents/*`) increases cognitive load.
111. Repeated boilerplate process text across many skills inflates context overhead.
112. Design-skill bodies/reference payloads are oversized and inconsistently routed.

### Brain structure hygiene

113. Broken live plan links in `brain/plans/index.md` and `brain/todos.md`.
114. Reachability/orphan-note issues in active brain notes.
115. Stale queue-era terminology remains in `brain/vision.md`.
116. Archived plans with stale `status: active` metadata and backlink drift.
117. Plan naming convention drift (`96-101` multi-id directory).
118. Todo priority/frontmatter drift vs open items.
119. Principle overlap clusters create ambiguity and repeated guidance.

## Phases

- [[plans/112-codebase-simplification-audit/phase-00-scaffold-and-gates]]
- [[plans/112-codebase-simplification-audit/phase-01-cli-config-adapter-boundaries]]
- [[plans/112-codebase-simplification-audit/phase-02-canonical-contract-cutline]]
- [[plans/112-codebase-simplification-audit/phase-03-loop-runtime-control-safety]]
- [[plans/112-codebase-simplification-audit/phase-04-server-ui-ws-convergence]]
- [[plans/112-codebase-simplification-audit/phase-05-event-pipeline-and-performance]]
- [[plans/112-codebase-simplification-audit/phase-06-docs-skills-brain-hygiene]]

## Verification Gates

### Global static gate (all phases)

- `pnpm check`
- `go test ./... && go vet ./...`
- `pnpm --filter noodle-ui test`
- `pnpm test:smoke`
- `sh scripts/lint-arch.sh`

### Global runtime gate (all phases)

- Phase-specific runtime checks must be automated where feasible.
- Manual checks are allowed only as secondary confirmation after automated checks pass.

### Cross-platform gate (program-level)

- CI matrix must pass on `linux`, `macos`, and `windows` before final program closure.
