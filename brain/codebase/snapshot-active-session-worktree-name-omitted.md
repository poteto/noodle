# Snapshot Active Session Worktree Name Omitted

- Symptom: context panel shows `Worktree: Not available` for active agent sessions, including execute stages that do run in linked worktrees.
- UI fallback is in `ui/src/components/ContextPanel.tsx`: `session.worktree_name?.trim() || "Not available"`.
- Root cause: `internal/snapshot.LoadSnapshot` builds `snapshot.Session` from `loop.CookSummary`, but `loop.CookSummary` originally did not carry `worktree_name`.
- Fix: add `WorktreeName` to `loop.CookSummary`, populate it from active cook handles, and map it into `snapshot.Session`.
- E2E coverage: `e2e/ui/smoke.spec.ts` now asserts active agent context shows a non-empty worktree label; `TestSmokeAgentLoop` runs Playwright after session spawn (Phase B) and before completion (Phase C) so active-session assertions are reliable.
- Related nuance: schedule sessions intentionally run on primary checkout and use empty worktree names, so they should still render as not available unless UI changes that behavior.

See also [[codebase/worktree-gotchas]], [[codebase/ws-http-snapshot-race-after-control-ack]]
