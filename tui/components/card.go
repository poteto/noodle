package components

import (
	"github.com/charmbracelet/lipgloss"
)

// Card renders a bordered card with optional title, body, and footer.
type Card struct {
	Title       string
	Body        string
	Footer      string
	BorderColor lipgloss.Color

	cache      string
	cacheWidth int
}

// Render returns the card as a bordered box constrained to width.
func (c *Card) Render(width int) string {
	if c.cache != "" && c.cacheWidth == width {
		return c.cache
	}

	t := DefaultTheme
	borderColor := c.BorderColor
	if borderColor == "" {
		borderColor = t.Border
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Background(t.CardBG).
		Foreground(t.Text).
		Padding(0, 1).
		Width(width - 2) // subtract border

	var content string
	if c.Title != "" {
		titleStyle := lipgloss.NewStyle().Foreground(t.Text).Bold(true)
		content = titleStyle.Render(c.Title)
		if c.Body != "" {
			content += "\n" + c.Body
		}
	} else {
		content = c.Body
	}

	if c.Footer != "" {
		footerStyle := lipgloss.NewStyle().Foreground(t.Dim)
		content += "\n" + footerStyle.Render(c.Footer)
	}

	c.cache = style.Render(content)
	c.cacheWidth = width
	return c.cache
}

// ClearCache invalidates the render cache (call on data change).
func (c *Card) ClearCache() {
	c.cache = ""
	c.cacheWidth = 0
}
