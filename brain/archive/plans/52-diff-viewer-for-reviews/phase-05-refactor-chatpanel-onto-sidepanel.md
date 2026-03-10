Back to [[archive/plans/52-diff-viewer-for-reviews/overview]]

# Phase 5: Refactor ChatPanel onto SidePanel

## Goal

Refactor the existing ChatPanel to compose the new SidePanel for its layout, removing duplicated overlay/backdrop/close logic. This is a pure refactor — no behavior change.

## Changes

**`ui/src/components/ChatPanel.tsx`**
- Remove: backdrop div, fixed positioning, escape key handler, backdrop click handler, width styling, animation classes
- Add: wrap content in `<SidePanel defaultWidth={560} onClose={onClose}>`
- Keep: header content, ChatMessages, ChatInput — these become children of SidePanel
- The ChatPanel becomes a pure content component that SidePanel wraps

After this phase, ChatPanel should look and behave exactly as before but with less code.

## Data structures

No new types.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Mechanical refactor with clear before/after |

## Verification

### Static
- `cd ui && npx tsc --noEmit`

### Runtime
- Open a cooking or done card — ChatPanel should open exactly as before
- Verify: escape closes, backdrop click closes, panel width is 560px
- Verify: resize handle works on the ChatPanel (inherited from SidePanel)
- No visual regressions
