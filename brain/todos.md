---
priority: [95, 84, 88, 90, 110, 96, 101, 100, 86, 111, 85, 69]
# Priority notes:
#    codebase simplification program — root-cause-first execution of full audit
#    deterministic completion detection — replace fragile completion heuristics
#   97 adapter schema validator — surface broken adapters
#   95 orders.json ownership — correctness, agents shouldn't write orders
#   84 sub-agent tracking — visibility into agent orchestration
#   88 sub-agent tracking v2 — hardening, depends on 84
#   90 interactive sessions — collaborative mode, key differentiator
#   110 chat input enhancements — polish interactive sessions UX
#   96,101 split out Sprites — delete built-in runtime, plugin interface
#   100 runtime mode UI setting
#   86 diffs integration — code-change review UX
#   111 backlog-in-UI visibility — improve operator ergonomics
#   85 .noodle.toml live reload — convenience, restart works fine
#   69 cursor dispatcher — extended runtime, process+sprites cover launch
---

# Todos

<!-- next-id: 115 -->
<!-- completed todos live in archive/completed_todos.md -->
<!-- completed plans live in archive/plans/ -->

## Onboarding & Getting Started


## Remote Dispatchers

69. [ ] Cursor dispatcher — implement CursorBackend (real HTTP client replacing stub), PollingDispatcher + pollingSession (deferred from plan 27 phase 4), webhook receiver endpoint on Noodle's HTTP server for status change notifications, factory wiring in loop/defaults.go. Flow: push worktree branch → launch Cursor Cloud Agent via API → agent pushes to target branch (no PR) → detect completion via webhook or polling → pull target branch back to worktree. [[plans/69-cursor-dispatcher/overview]]

## UI

100. [ ] Add a UI setting to change the run mode (auto/supervised/manual) at runtime. #needs_plan
110. [ ] Chat input enhancements — drag-to-add images, copy/paste images, and `@` autocomplete for files (and possibly actors). #needs_plan
86. [ ] Integrate diffs.com diff-rendering component into the web UI — add a bundled JS diff component (from https://diffs.com/) that renders code changes as inline diffs. Show diffs in two places: (1) inline in the session activity feed alongside each code-change event, collapsed by default with a click-to-expand interaction (avoid noise in the feed), and (2) in a dedicated diff tab/panel that collects all code changes from a session (expanded by default). Ship-ready: fully integrated, styled, and tested. [[plans/86-diffs-integration/overview]]
111. [ ] Allow viewing backlog items in the UI. #needs_plan

## Backend

97. [x] Adapter schema validator: validate adapter output against the expected schema. If invalid, raise a warning that surfaces in the UI and backend logs, and inject the warning into the scheduler prompt so it can create a task to fix the broken adapter. Update adapters docs page with validation behavior. [[archive/plans/97-adapter-schema-validator/overview]]
95. [ ] Backend should exclusively own `orders.json` — prevent agents from writing to it directly. The loop promotes `orders-next.json` into `orders.json`, and this should be enforced at the backend level (e.g. file permissions, validation gate) rather than relying on skill instructions. #needs_plan
84. [ ] Sub-agent tracking — parse Claude/Codex sub-agent lifecycle into canonical events, build agent tree in snapshots, stream activity to UI, and enable user steering. [[plans/84-subagent-tracking/overview]] — define canonical backend failure classes (hard invariant, recoverable backend, scheduler/cook agent mistake, agent-start unrecoverable vs retryable), map loop/start/dispatcher boundaries, and surface typed recoverability metadata for operators. [[archive/plans/83-error-recoverability-taxonomy/overview]]
85. [ ] Add `.noodle.toml` fsnotify live reload in the running loop with safe apply semantics (debounce, parse/validation gate, partial-apply vs restart-required classification, and observability for rejected reloads). #needs_plan
88. [ ] Sub-agent tracking v2 — add out-of-band ingestion (Codex child sessions + Claude team inbox), harden canonical identity reconciliation, lifecycle-safe bounded pollers, robust `steer-agent` control behavior, and expanded hardening tests. [[plans/88-subagent-tracking-v2/overview]]

## Extensibility

96. [ ] Custom dispatcher/runtime plugins — allow users to write their own dispatcher or runtime plugin so they can add a custom VM by implementing Noodle's interface. [[plans/96-101-runtime-plugins/overview]]
101. [ ] Split out Sprites runtime into its own runtime plugin (`noodle-runtime-sprites`). Sprites should not be a built-in default, it should be an installable plugin that uses the custom runtime interface (#96). [[plans/96-101-runtime-plugins/overview]]

## Design Explorations


## Features

90. [ ] Interactive agent sessions — spawn a human-in-the-loop agent session outside the order system for collaborative work (planning, exploration, design decisions) where full automation is wrong. Separate from orders: no order ID, no stage pipeline, no quality/reflect lifecycle. Backend: new control action (not `enqueue`) that spawns a provider session with user-chosen provider/model, streams canonical events, and accepts user messages mid-flight via the existing steering mechanism. Runs on primary checkout by default, opt-in worktree for code-change tasks. UI: web chat panel in Noodle UI — real-time streamed agent output with a user input field for back-and-forth conversation. Launcher placement (replace scheduler feed input vs. new top-level action vs. separate panel) is undecided — design should emerge during planning. Long-term: unify the spawn surface so both structured orders and interactive sessions launch from the same place. [[plans/90-interactive-sessions/overview]]
