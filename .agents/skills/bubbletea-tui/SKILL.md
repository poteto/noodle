---
name: bubbletea-tui
description: >
  Write high-quality Bubble Tea terminal UIs following production patterns.
  Covers component design, styling, animations, state management, and
  rendering. Use when building or refactoring Bubble Tea
  TUI code — new components, views, styling, list items, animations, or when
  reviewing TUI code quality. Triggers: "build a TUI", "bubbletea component",
  "TUI styling", "terminal UI", "improve the TUI", "refactor the view", or
  any Bubble Tea implementation work.
---

# Bubble Tea TUI — Production Patterns

Apply when writing or reviewing Bubble Tea code.

## Components Are Dumb

Components don't handle `tea.Msg` directly. Instead:

1. Expose **methods** for state changes that return `tea.Cmd` for side effects
2. Render via `Render(width int) string`
3. Let the **main model** own message routing, focus, layout, and orchestration

```go
// Component — no Update(), no tea.Msg handling
type StatusBar struct { ... }
func (s *StatusBar) SetStatus(text string)      { s.text = text; s.clearCache() }
func (s *StatusBar) Render(width int) string     { ... }

// Main model — the only place that handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case StatusMsg:
        m.statusBar.SetStatus(msg.Text)
    }
    ...
}
```

## Styling

### Centralize All Styles

Define all styles in one place. Components receive styles via a shared struct
or package — never define styles inline or in component files.

### Semantic Colors

Name colors by meaning, not appearance. Four tiers of text hierarchy:

```go
colorFg     // primary — headings, active text
colorDim    // secondary — timestamps, labels
colorMuted  // tertiary — separators, continuation markers
colorSubtle // quaternary — barely visible, decorative
```

Plus semantic status colors: `success`, `error`, `warning`, `info`.

### Focus States

Every interactive element needs focused and blurred variants. Define them as
paired styles, not ad-hoc conditionals in render code.

### Reusable Element Functions

Build composable rendering primitives (see `references/elements.md`):

- **Section(title, width)** — title + `─` rule filling remaining width
- **Status(icon, title, description, width)** — auto-truncating status line
- **Badge(label, color)** — colored inline label

These are pure functions: styles + data + width in, string out.

## Rendering

### Width Flows Down

Every render method takes `width int`. Parent calculates available width after
its own padding/borders, passes remainder to children. Never hard-code widths.

### Cache Rendered Output

For items that re-render frequently (list items, log lines), cache the rendered
string keyed by width. Invalidate on data change, not on every frame:

```go
type cached struct {
    rendered string
    width    int
}
func (c *cached) get(width int) (string, bool) {
    if c.rendered != "" && c.width == width { return c.rendered, true }
    return "", false
}
```

### ANSI-Safe String Operations

Use `github.com/charmbracelet/x/ansi` — never `len()` or `string[:n]` on
styled strings:

- `ansi.StringWidth(s)` — visible width (not byte length)
- `ansi.Truncate(s, width, "…")` — truncate preserving escape codes
- `ansi.Cut(s, start, end)` — substring preserving escape codes
- `ansi.Strip(s)` — remove all ANSI codes

Also: `lipgloss.Width(s)` for measuring styled output width.

## State Management

### IO Never in Update

`Update()` reads messages and mutates model state. All side effects (file IO,
network, process spawning) go in `tea.Cmd`:

```go
// WRONG
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    data, _ := os.ReadFile("config.json")  // blocks render loop
    m.config = parse(data)
    return m, nil
}

// RIGHT
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, loadConfig  // tea.Cmd runs async
}
func loadConfig() tea.Msg { ... }
```

### State Never in Commands

Commands return messages. Model mutation happens only in `Update`:

```go
// WRONG — mutating model in closure
cmd := func() tea.Msg { m.loading = true; return fetchData() }

// RIGHT — message triggers state change in Update
cmd := fetchData  // returns DataMsg
// Update: case DataMsg: m.loading = false; m.data = msg.Data
```

### Batch Commands

Use `tea.Batch()` when returning multiple commands from a single handler.

## Composition via Embedding

Use struct embedding for cross-cutting concerns (caching, focus, selection):

```go
type focusable struct{ focused bool }
func (f *focusable) SetFocused(b bool) { f.focused = b }

type MyItem struct {
    *cached                // render caching
    *focusable             // focus state
    data   SomeDomainType  // actual content
}
```

### Capability Interfaces

```go
type Item interface      { Render(width int) string }
type Focusable interface { SetFocused(bool) }
type Expandable interface{ ToggleExpanded() bool }
```

Components implement only what they need. Containers check via type assertion.

## Animation

### Pre-Render Frames

Generate all styled frames at init. Render just indexes into the array — no
lipgloss processing in the hot path:

```go
// Init
frames[i] = lipgloss.NewStyle().Foreground(color).Render(char)

// Render
return frames[step.Load()]
```

### Tick-Based Advancement

Drive animations with `tea.Tick` returning a typed message:

```go
func tickCmd() tea.Cmd {
    return tea.Tick(time.Second/20, func(time.Time) tea.Msg {
        return AnimTickMsg{}
    })
}
```

Handle `AnimTickMsg` in `Update` to advance state and schedule the next tick.
Stop ticking when no animations are active.

### Thread Safety

Use `atomic.Int64` / `atomic.Bool` for counters and flags that may be read by
`View()` while `Update()` advances them.

## References

- `references/elements.md` — reusable UI element patterns (Section, Status, Badge, DialogTitle)
- `references/production-rules.md` — production Bubble Tea coding rules
