# Reusable UI Elements

Composable rendering functions for TUI views. Each takes a styles reference
and width, returning a styled string.

## Section Header

Title with a horizontal rule filling the remaining width. Optional info text
right-aligned.

```go
func Section(t *styles.Styles, text string, width int, info ...string) string {
    char := "─"  // or custom separator
    length := lipgloss.Width(text) + 1
    remainingWidth := width - length

    var infoText string
    if len(info) > 0 {
        infoText = " " + strings.Join(info, " ")
        remainingWidth -= lipgloss.Width(infoText)
    }

    text = t.Section.Title.Render(text)
    if remainingWidth > 0 {
        text += " " + t.Section.Line.Render(strings.Repeat(char, remainingWidth)) + infoText
    }
    return text
}
```

Output: `Session ──────────────────────── 3 workers`

## Status Line

Icon + title + auto-truncating description + optional extra content. Fits any
width by truncating the description first.

```go
type StatusOpts struct {
    Icon             string
    Title            string
    TitleColor       color.Color
    Description      string
    DescriptionColor color.Color
    ExtraContent     string
}

func Status(t *styles.Styles, opts StatusOpts, width int) string {
    // ... title styled with TitleColor, description truncated to fit,
    // extra content appended at end
    // Uses ansi.Truncate for ANSI-safe truncation
}
```

Output: `✓ Build  compiled 42 files in 1.2s  $0.03`

## Model Info Badge

Multi-line badge showing model name, provider, reasoning mode, token usage,
and cost. Lines wrap to second line if they don't fit.

```go
func ModelInfo(t *styles.Styles, modelName, providerName, reasoningInfo string,
    context *ModelContextInfo, width int) string
```

Output:
```
◆ claude-opus-4-6 via anthropic
    extended thinking
    47% (12.3K) $0.42
```

Token formatting: K for thousands, M for millions. Warning icon at >80% context.

## Dialog Title

Title with gradient-colored decorative separator:

```go
func DialogTitle(t *styles.Styles, title string, width int,
    fromColor, toColor color.Color) string
```

Output: `Settings ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱` (with color gradient)

## Tool Status Icons

```go
const (
    ToolPendingIcon  = "●"   // dimmed
    ToolSuccessIcon  = "✓"   // green
    ToolErrorIcon    = "×"   // red
    ToolCanceledIcon = "○"   // dimmed
)
```

## Patterns

- **Width always flows in as parameter** — never hard-code
- **Use `ansi.Truncate(s, width, "…")`** for safe truncation of styled strings
- **Compose elements** — Section + Status lines build complete views
- **Return strings** — elements are pure rendering functions, no state
