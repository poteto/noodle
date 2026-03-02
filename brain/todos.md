---
priority: [98, 99, 95, 96, 101, 84, 90, 86, 93, 94]
# Launch-blocking:
#   - config cleanup — small deletions, simplify surface before launch
#   98,99 relax backlog schema — simplify adapters, less rigid
#   95 orders.json ownership — correctness, agents shouldn't write orders
#   96,101 split out Sprites — delete built-in runtime, plugin interface
#   84 sub-agent tracking — visibility into agent orchestration
#   90 interactive sessions — collaborative mode, key differentiator
#   86 diffs integration — code-change review UX
#   93,94 distribution — Homebrew + cross-platform binaries
# Post-launch:
#   88 sub-agent tracking v2 — hardening, depends on 84
#   85 .noodle.toml live reload — convenience, restart works fine
#   69 cursor dispatcher — extended runtime, process+sprites cover launch
---

# Todos

<!-- next-id: 109 -->
<!-- completed todos live in archive/completed_todos.md -->
<!-- completed plans live in archive/plans/ -->

## Onboarding & Getting Started

94. [ ] Cross-platform distribution — publish Noodle binaries for Linux and Windows in addition to macOS. Set up GoReleaser (or equivalent) to cross-compile for linux/amd64, linux/arm64, windows/amd64. Publish artifacts to GitHub Releases. Add install instructions for non-Homebrew users (curl one-liner, winget/scoop, or direct download). Update getting-started docs with platform-specific instructions.

93. [ ] Publish build to Homebrew — create a Homebrew tap and formula so users can `brew install noodle`.

## Remote Dispatchers

69. [ ] Cursor dispatcher — implement CursorBackend (real HTTP client replacing stub), PollingDispatcher + pollingSession (deferred from plan 27 phase 4), webhook receiver endpoint on Noodle's HTTP server for status change notifications, factory wiring in loop/defaults.go. Flow: push worktree branch → launch Cursor Cloud Agent via API → agent pushes to target branch (no PR) → detect completion via webhook or polling → pull target branch back to worktree. [[plans/69-cursor-dispatcher/overview]]

## UI

100. [ ] Add a UI setting to change the run mode (auto/supervised/manual) at runtime.

86. [ ] Integrate diffs.com diff-rendering component into the web UI — add a bundled JS diff component (from https://diffs.com/) that renders code changes as inline diffs. Show diffs in two places: (1) inline in the session activity feed alongside each code-change event, collapsed by default with a click-to-expand interaction (avoid noise in the feed), and (2) in a dedicated diff tab/panel that collects all code changes from a session (expanded by default). Ship-ready: fully integrated, styled, and tested. [[plans/86-diffs-integration/overview]]

## Backend

97. [ ] Adapter schema validator: validate adapter output against the expected schema. If invalid, raise a warning that surfaces in the UI and backend logs, and inject the warning into the scheduler prompt so it can create a task to fix the broken adapter. Update adapters docs page with validation behavior.

98. [ ] Relax backlog schema: only require `id` and `title`. Everything else is optional and passed through to mise.json as-is for the scheduler to see. Remove hardcoded `status`/`estimate` enums and strict validation. Noodle should be flexible about what fields adapters return. Update adapters docs page schema section to match.

99. [ ] Make `section` optional in backlog `add` payload. Currently the default adapter defaults it to "Inbox" but it shouldn't be a first-class field. Pass it through like any other optional field.
95. [ ] Backend should exclusively own `orders.json` — prevent agents from writing to it directly. The loop promotes `orders-next.json` into `orders.json`, and this should be enforced at the backend level (e.g. file permissions, validation gate) rather than relying on skill instructions.
84. [ ] Sub-agent tracking — parse Claude/Codex sub-agent lifecycle into canonical events, build agent tree in snapshots, stream activity to UI, and enable user steering. [[plans/84-subagent-tracking/overview]] — define canonical backend failure classes (hard invariant, recoverable backend, scheduler/cook agent mistake, agent-start unrecoverable vs retryable), map loop/start/dispatcher boundaries, and surface typed recoverability metadata for operators. [[plans/83-error-recoverability-taxonomy/overview]]
85. [ ] Add `.noodle.toml` fsnotify live reload in the running loop with safe apply semantics (debounce, parse/validation gate, partial-apply vs restart-required classification, and observability for rejected reloads).
88. [ ] Sub-agent tracking v2 — add out-of-band ingestion (Codex child sessions + Claude team inbox), harden canonical identity reconciliation, lifecycle-safe bounded pollers, robust `steer-agent` control behavior, and expanded hardening tests. [[plans/88-subagent-tracking-v2/overview]]

## Extensibility

96. [ ] Custom dispatcher/runtime plugins — allow users to write their own dispatcher or runtime plugin so they can add a custom VM by implementing Noodle's interface. [[plans/96-101-runtime-plugins/overview]]

101. [ ] Split out Sprites runtime into its own runtime plugin (`noodle-runtime-sprites`). Sprites should not be a built-in default, it should be an installable plugin that uses the custom runtime interface (#96). [[plans/96-101-runtime-plugins/overview]]

## Design Explorations

108. [ ] Compact wire format for orders-next.json — `do`/`with` replace `task_key`/`provider`, drop `status` and `skill` from scheduler output. Expansion at the promotion boundary produces internal `orderx.OrdersFile` structs. [[plans/108-orders-dsl/overview]]

## Features

90. [ ] Interactive agent sessions — spawn a human-in-the-loop agent session outside the order system for collaborative work (planning, exploration, design decisions) where full automation is wrong. Separate from orders: no order ID, no stage pipeline, no quality/reflect lifecycle. Backend: new control action (not `enqueue`) that spawns a provider session with user-chosen provider/model, streams canonical events, and accepts user messages mid-flight via the existing steering mechanism. Runs on primary checkout by default, opt-in worktree for code-change tasks. UI: web chat panel in Noodle UI — real-time streamed agent output with a user input field for back-and-forth conversation. Launcher placement (replace scheduler feed input vs. new top-level action vs. separate panel) is undecided — design should emerge during planning. Long-term: unify the spawn surface so both structured orders and interactive sessions launch from the same place.
