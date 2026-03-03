# Shutdown Summary Buffer Race

- In `session_base.go`'s dropped-event summary emission, a best-effort non-blocking send can lose the warning when `events` is still full at shutdown.
- Root cause: summary is emitted before closing the channel, but there may be no active consumer draining at that moment.
- Fix: at shutdown, if summary enqueue hits a full buffer, drop one oldest buffered event and retry so the summary remains observable.
- Durable logging to session event logs is the fallback source of truth.

See also [[principles/fix-root-causes]], [[principles/prove-it-works]], [[codebase/adopted-session-reconciliation]]
