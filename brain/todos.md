---
priority: [20, 84, 90, 86, 88, 85, 69]
# Launch-blocking:
#   20 onboarding — install, docs, examples, getting-started guides
#   84 sub-agent tracking — visibility into agent orchestration
#   90 interactive sessions — collaborative mode, key differentiator
#   86 diffs integration — code-change review UX
# Post-launch:
#   88 sub-agent tracking v2 — hardening, depends on 84
#   85 .noodle.toml live reload — convenience, restart works fine
#   69 cursor dispatcher — extended runtime, process+sprites cover launch
---

# Todos

<!-- next-id: 92 -->
<!-- completed todos live in archive/completed_todos.md -->
<!-- completed plans live in archive/plans/ -->

## Onboarding & Getting Started

20. [ ] First-run experience — make it obvious how to install, configure, and use Noodle. Simple installation path, clear documentation of core concepts (skills, scheduling, brain), example projects showing real workflows (e.g. a minimal schedule+execute setup, a multi-skill autonomous loop), and getting-started guides that take a new user from zero to a running Noodle in minutes. Includes resolving skill-path defaults (`.agents/skills` vs project-level) so bootstrapping just works.

## Remote Dispatchers

69. [ ] Cursor dispatcher — implement CursorBackend (real HTTP client replacing stub), PollingDispatcher + pollingSession (deferred from plan 27 phase 4), webhook receiver endpoint on Noodle's HTTP server for status change notifications, factory wiring in loop/defaults.go. Flow: push worktree branch → launch Cursor Cloud Agent via API → agent pushes to target branch (no PR) → detect completion via webhook or polling → pull target branch back to worktree. [[plans/69-cursor-dispatcher/overview]]

## UI

86. [ ] Integrate diffs.com diff-rendering component into the web UI — add a bundled JS diff component (from https://diffs.com/) that renders code changes as inline diffs. Show diffs in two places: (1) inline in the session activity feed alongside each code-change event, collapsed by default with a click-to-expand interaction (avoid noise in the feed), and (2) in a dedicated diff tab/panel that collects all code changes from a session (expanded by default). Ship-ready: fully integrated, styled, and tested. [[plans/86-diffs-integration/overview]]

## Backend
84. [ ] Sub-agent tracking — parse Claude/Codex sub-agent lifecycle into canonical events, build agent tree in snapshots, stream activity to UI, and enable user steering. [[plans/84-subagent-tracking/overview]] — define canonical backend failure classes (hard invariant, recoverable backend, scheduler/cook agent mistake, agent-start unrecoverable vs retryable), map loop/start/dispatcher boundaries, and surface typed recoverability metadata for operators. [[plans/83-error-recoverability-taxonomy/overview]]
85. [ ] Add `.noodle.toml` fsnotify live reload in the running loop with safe apply semantics (debounce, parse/validation gate, partial-apply vs restart-required classification, and observability for rejected reloads).
88. [ ] Sub-agent tracking v2 — add out-of-band ingestion (Codex child sessions + Claude team inbox), harden canonical identity reconciliation, lifecycle-safe bounded pollers, robust `steer-agent` control behavior, and expanded hardening tests. [[plans/88-subagent-tracking-v2/overview]]

## Features

90. [ ] Interactive agent sessions — spawn a human-in-the-loop agent session outside the order system for collaborative work (planning, exploration, design decisions) where full automation is wrong. Separate from orders: no order ID, no stage pipeline, no quality/reflect lifecycle. Backend: new control action (not `enqueue`) that spawns a provider session with user-chosen provider/model, streams canonical events, and accepts user messages mid-flight via the existing steering mechanism. Runs on primary checkout by default, opt-in worktree for code-change tasks. UI: web chat panel in Noodle UI — real-time streamed agent output with a user input field for back-and-forth conversation. Launcher placement (replace scheduler feed input vs. new top-level action vs. separate panel) is undecided — design should emerge during planning. Long-term: unify the spawn surface so both structured orders and interactive sessions launch from the same place.
