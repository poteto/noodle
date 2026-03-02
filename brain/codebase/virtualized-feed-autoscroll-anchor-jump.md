# Virtualized Feed Autoscroll Anchor Jump

- Symptom: Actor feed can jump upward and lose autoscroll while messages continue streaming.
- Root cause: `VirtualizedFeed` only re-pinned to bottom when `items.length` changed.
  Height changes without list-length changes (streaming tail growth, row remeasurement)
  could move the viewport away from bottom and flip autoscroll off.
- Fix: keep a `ResizeObserver` on feed content and re-pin to bottom while autoscroll is enabled.
  This preserves user intent: if the user scrolls away, observer is disconnected and no forced jump occurs.

See also [[principles/fix-root-causes]]
