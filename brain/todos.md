---
priority: [84, 88, 20, 69]
# 84 sub-agent tracking (foundational infra),
# 88 sub-agent tracking v2 (extends 84),
# 20 skill-path defaults (small docs fix),
# 69 cursor dispatcher (feature, benefits from cleaned dispatcher package)
---

# Todos

<!-- next-id: 90 -->
<!-- completed todos live in archive/completed_todos.md -->
<!-- completed plans live in archive/plans/ -->

## Bootstrap Skill Fixes

20. [ ] Clarify skill-path defaults for repo vs user projects — current default is `.agents/skills`, but bootstrap/docs should explicitly explain repo-internal development vs user-project bootstrapping expectations and where skills are expected to live in each mode.

## Remote Dispatchers

69. [ ] Cursor dispatcher — implement CursorBackend (real HTTP client replacing stub), PollingDispatcher + pollingSession (deferred from plan 27 phase 4), webhook receiver endpoint on Noodle's HTTP server for status change notifications, factory wiring in loop/defaults.go. Flow: push worktree branch → launch Cursor Cloud Agent via API → agent pushes to target branch (no PR) → detect completion via webhook or polling → pull target branch back to worktree. [[plans/69-cursor-dispatcher/overview]]

## UI

86. [ ] Integrate diffs.com diff-rendering component into the web UI — add a bundled JS diff component (from https://diffs.com/) that renders code changes as inline diffs. Show diffs in two places: (1) inline in the session activity feed alongside each code-change event, and (2) in a dedicated diff tab/panel that collects all code changes from a session. Ship-ready: fully integrated, styled, and tested.

## Backend
84. [ ] Sub-agent tracking — parse Claude/Codex sub-agent lifecycle into canonical events, build agent tree in snapshots, stream activity to UI, and enable user steering. [[plans/84-subagent-tracking/overview]] — define canonical backend failure classes (hard invariant, recoverable backend, scheduler/cook agent mistake, agent-start unrecoverable vs retryable), map loop/start/dispatcher boundaries, and surface typed recoverability metadata for operators. [[plans/83-error-recoverability-taxonomy/overview]]
85. [ ] Add `.noodle.toml` fsnotify live reload in the running loop with safe apply semantics (debounce, parse/validation gate, partial-apply vs restart-required classification, and observability for rejected reloads).
88. [ ] Sub-agent tracking v2 — add out-of-band ingestion (Codex child sessions + Claude team inbox), harden canonical identity reconciliation, lifecycle-safe bounded pollers, robust `steer-agent` control behavior, and expanded hardening tests. [[plans/88-subagent-tracking-v2/overview]]
89. [ ] Runtime merge detection — replace static `permissions.merge` frontmatter with runtime `HasUnmergedCommits` check on the worktree. Removes `Permissions` struct, `CanMerge` from taskreg/mise, and simplifies skill frontmatter. [[plans/89-runtime-merge-detection/overview]]
