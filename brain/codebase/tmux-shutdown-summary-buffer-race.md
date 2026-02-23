# Tmux Shutdown Summary Buffer Race

- In `tmuxSession.emitDroppedEventSummary`, a best-effort non-blocking send can
  lose the dropped-events warning when `events` is still full at shutdown.
- Root cause: `closeEventsWhenDone` emits the summary before closing the
  channel, but there may be no active consumer draining at that moment.
- Fix: at shutdown, if summary enqueue hits a full buffer, drop one oldest
  buffered event and retry so the summary remains observable.
- Keep durable logging to session event logs as the fallback source of truth.

See also [[principles/fix-root-causes]], [[principles/trust-the-output-not-the-report]]
