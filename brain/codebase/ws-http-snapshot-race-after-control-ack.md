# WS/HTTP Snapshot Race After Control Ack

- Symptom: Scheduler/agent feeds can stay in `Thinking…` after stop because `snapshot.active` in React Query becomes stale.
- Root cause: `useSendControl` invalidated `["snapshot"]` immediately on control ack, issuing an HTTP `/api/snapshot` fetch that could resolve after a newer WebSocket snapshot push and overwrite it.
- Why this can happen:
  - Control ack only means command append/accept, not that loop processing has completed.
  - `/api/snapshot` reads current published loop state, which can still show pre-stop active cooks.
  - React Query accepted late HTTP data and replaced newer WS cache data.
- Fix:
  - Invalidate snapshot only when WS is not connected (`useSendControl` fallback behavior).
  - Make snapshot/session HTTP queries abortable via `AbortSignal`.
  - Cancel in-flight snapshot query when a WS `snapshot` message arrives before writing WS data into cache.
  - Apply `updated_at` freshness arbitration (`chooseNewerSnapshot`) so older responses are ignored even if cancellation misses.
  - UI busy indicators should gate on loop state + running status, not `active` membership alone, to avoid showing `Thinking…` when loop is idle.
  - Session turn-boundary detail: canonical `result` events are mapped into session `cost` events, so snapshot enrichment must clear stale `current_action` on `cost` to reflect interrupted-turn idle state.
- Regression coverage: `ui/src/client/hooks.test.tsx` asserts connected WS path skips invalidation and disconnected path still invalidates.

See also [[principles/fix-root-causes]], [[principles/serialize-shared-state-mutations]]
