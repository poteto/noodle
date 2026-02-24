package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/tui/components"
)

var autonomyModes = []string{
	config.AutonomyApprove,
	config.AutonomyReview,
	config.AutonomyFull,
}

// ConfigTab renders the config view with the autonomy dial.
type ConfigTab struct {
	autonomy string
	index    int // position in autonomyModes
}

// SetAutonomy updates the displayed autonomy mode.
func (c *ConfigTab) SetAutonomy(mode string) {
	c.autonomy = mode
	for i, m := range autonomyModes {
		if m == mode {
			c.index = i
			return
		}
	}
	c.index = 1 // default to review
}

// CycleLeft moves the autonomy dial left (more restrictive).
func (c *ConfigTab) CycleLeft() string {
	if c.index > 0 {
		c.index--
	}
	c.autonomy = autonomyModes[c.index]
	return c.autonomy
}

// CycleRight moves the autonomy dial right (more autonomous).
func (c *ConfigTab) CycleRight() string {
	if c.index < len(autonomyModes)-1 {
		c.index++
	}
	c.autonomy = autonomyModes[c.index]
	return c.autonomy
}

// Render returns the config tab content.
func (c *ConfigTab) Render(width, height int) string {
	t := components.DefaultTheme

	var b strings.Builder
	b.WriteString(sectionStyle.Render("Autonomy"))
	b.WriteString("\n\n")

	// Render the 3-position dial.
	for i, mode := range autonomyModes {
		label := mode
		if i == c.index {
			b.WriteString(lipgloss.NewStyle().Foreground(t.Brand).Bold(true).Render("[" + label + "]"))
		} else {
			b.WriteString(dimStyle.Render(" " + label + " "))
		}
		if i < len(autonomyModes)-1 {
			b.WriteString(dimStyle.Render(" ── "))
		}
	}
	b.WriteString("\n\n")

	// Description of current mode.
	var desc string
	switch c.autonomy {
	case config.AutonomyFull:
		desc = "Auto-merge on success. No human in the loop."
	case config.AutonomyReview:
		desc = "Quality review runs. Auto-merge on approve."
	case config.AutonomyApprove:
		desc = "Quality review runs. Human must confirm merge."
	}
	b.WriteString(dimStyle.Render(desc))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("← / → to change"))

	return b.String()
}
