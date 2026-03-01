---
priority: [84, 87, 88, 20, 69, 77]
# 84 sub-agent tracking (foundational infra, in progress),
# 87 go simplification (subtract before you add — phases 1-2 delete dead
#    backend stubs that 69 will redesign, phase 3 consolidates state
#    utilities dispatchers depend on),
# 88 sub-agent tracking v2 (extends 84),
# 20 skill-path defaults (small docs fix),
# 69 cursor dispatcher (feature, benefits from cleaned dispatcher package),
# 77 react-window (UI polish)
---

# Todos

<!-- next-id: 89 -->
<!-- completed todos live in archive/completed_todos.md -->
<!-- completed plans live in archive/plans/ -->

## Bootstrap Skill Fixes

20. [ ] Clarify skill-path defaults for repo vs user projects — current default is `.agents/skills`, but bootstrap/docs should explicitly explain repo-internal development vs user-project bootstrapping expectations and where skills are expected to live in each mode.

## Remote Dispatchers

69. [ ] Cursor dispatcher — implement CursorBackend (real HTTP client replacing stub), PollingDispatcher + pollingSession (deferred from plan 27 phase 4), webhook receiver endpoint on Noodle's HTTP server for status change notifications, factory wiring in loop/defaults.go. Flow: push worktree branch → launch Cursor Cloud Agent via API → agent pushes to target branch (no PR) → detect completion via webhook or polling → pull target branch back to worktree. [[plans/69-cursor-dispatcher/overview]]

## UI

77. [ ] Add react-window (https://github.com/bvaughn/react-window) — virtualized list rendering for large datasets
86. [ ] Investigate adding https://diffs.com/ to the web UI — every code change can be represented as a diff, making it easier for users to review code changes inline

## Backend
84. [ ] Sub-agent tracking — parse Claude/Codex sub-agent lifecycle into canonical events, build agent tree in snapshots, stream activity to UI, and enable user steering. [[plans/84-subagent-tracking/overview]] — define canonical backend failure classes (hard invariant, recoverable backend, scheduler/cook agent mistake, agent-start unrecoverable vs retryable), map loop/start/dispatcher boundaries, and surface typed recoverability metadata for operators. [[plans/83-error-recoverability-taxonomy/overview]]
85. [ ] Add `.noodle.toml` fsnotify live reload in the running loop with safe apply semantics (debounce, parse/validation gate, partial-apply vs restart-required classification, and observability for rejected reloads).
87. [ ] Go codebase simplification — deduplicate state utilities, extract loop helpers, decompose long functions, delete dead code, create shared jsonx/stringx utilities. [[plans/87-go-codebase-simplification/overview]]
88. [ ] Sub-agent tracking v2 — add out-of-band ingestion (Codex child sessions + Claude team inbox), harden canonical identity reconciliation, lifecycle-safe bounded pollers, robust `steer-agent` control behavior, and expanded hardening tests. [[plans/88-subagent-tracking-v2/overview]]
