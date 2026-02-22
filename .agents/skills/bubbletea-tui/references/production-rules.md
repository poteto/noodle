# Bubble Tea Production Rules

Canonical coding guidelines for production Bubble Tea applications.

## Hard Rules

- Never do IO or expensive work in `Update`; always use a `tea.Cmd`.
- Never change model state inside a command — use messages, then update in the
  main Update loop.
- Never use commands to send messages when you can directly mutate children or
  state.
- Use `github.com/charmbracelet/x/ansi` for string manipulation with ANSI
  codes: `ansi.Cut`, `ansi.StringWidth`, `ansi.Strip`, `ansi.Truncate`.

## Design Rules

- Keep things simple; do not overcomplicate.
- Create files to separate logic; do not nest models.
- Components don't handle bubbletea messages directly — expose methods for
  state changes, return `tea.Cmd` for side effects, render via
  `Render(width int) string`.
- All styles in one place, accessed via a shared struct. Use semantic color
  fields, not hardcoded colors.
- Always account for padding/borders in width calculations.
- Use `tea.Batch()` when returning multiple commands.

## Composition Patterns

- **Struct embedding** for shared behavior (caching, highlighting, focus).
- **Interfaces** for capabilities (Item, Focusable, Highlightable, Expandable).
- **Cache rendered output** — invalidate on data change, not every render.
  Embed a cache struct with width-keyed lookup.
- **Overlay system** for dialogs and panels — render to string, composite
  on top of background.
