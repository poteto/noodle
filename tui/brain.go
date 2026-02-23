package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// BrainActivity represents a single knowledge-base change.
type BrainActivity struct {
	Agent       string
	At          time.Time
	Tag         string // "new", "edit", or "delete"
	FilePath    string
	Description string
}

// BrainTab renders the brain activity feed with list and preview modes.
type BrainTab struct {
	items     []BrainActivity
	selected  int
	preview   bool
	previewMD string // cached glamour output
}

// SetBrainActivity replaces the activity list and resets selection.
func (b *BrainTab) SetBrainActivity(items []BrainActivity) {
	b.items = items
	if b.selected >= len(items) {
		b.selected = 0
	}
	b.preview = false
	b.previewMD = ""
}

// Render returns the brain tab content for the given dimensions.
func (b *BrainTab) Render(width, height int) string {
	if b.preview {
		return b.renderPreview(width, height)
	}
	return b.renderList(width, height)
}

func (b *BrainTab) renderList(width, height int) string {
	if len(b.items) == 0 {
		return b.renderEmpty(width)
	}

	var out strings.Builder

	// Stats line
	out.WriteString(b.renderStats(width))
	out.WriteString("\n\n")

	// Group by agent
	groups := b.groupByAgent()
	for i, group := range groups {
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString(sectionStyle.Render(group.agent))
		out.WriteString("\n")
		for j, item := range group.items {
			globalIdx := group.indices[j]
			line := b.renderEntry(item, width-2, globalIdx == b.selected)
			out.WriteString(line)
			out.WriteString("\n")
		}
	}

	return out.String()
}

func (b *BrainTab) renderPreview(width, height int) string {
	var out strings.Builder
	out.WriteString(dimStyle.Render("esc to return"))
	out.WriteString("\n\n")

	if b.selected >= 0 && b.selected < len(b.items) {
		item := b.items[b.selected]
		out.WriteString(sectionStyle.Render(item.FilePath))
		out.WriteString("\n\n")
	}

	if b.previewMD != "" {
		out.WriteString(b.previewMD)
	} else {
		out.WriteString(dimStyle.Render("(loading preview...)"))
	}
	return out.String()
}

func (b *BrainTab) renderEmpty(width int) string {
	return dimStyle.Render("(no brain activity)")
}

func (b *BrainTab) renderStats(width int) string {
	noteCount := 0
	principleCount := 0
	planCount := 0
	for _, item := range b.items {
		switch {
		case strings.Contains(item.FilePath, "principles/"):
			principleCount++
		case strings.Contains(item.FilePath, "plans/"):
			planCount++
		default:
			noteCount++
		}
	}
	return dimStyle.Render(fmt.Sprintf("%d notes  %d principles  %d plans", noteCount, principleCount, planCount))
}

func (b *BrainTab) renderEntry(item BrainActivity, width int, selected bool) string {
	tag := tagLabel(item.Tag)
	path := trimTo(item.FilePath, 40)
	desc := item.Description
	if desc == "" {
		desc = "(no description)"
	}

	line := fmt.Sprintf("  %s %s  %s", tag, mutedStyle.Render(path), dimStyle.Render(desc))
	if selected {
		line = selectedRowStyle.Render(line)
	}
	return line
}

func tagLabel(tag string) string {
	style := lipgloss.NewStyle().Bold(true)
	switch strings.ToLower(tag) {
	case "new":
		return style.Foreground(theme.Success).Render("new")
	case "edit":
		return style.Foreground(theme.Info).Render("edit")
	case "delete":
		return style.Foreground(theme.Error).Render("del")
	default:
		return dimStyle.Render(tag)
	}
}

type brainGroup struct {
	agent   string
	items   []BrainActivity
	indices []int
}

func (b *BrainTab) groupByAgent() []brainGroup {
	order := make([]string, 0)
	byAgent := make(map[string]*brainGroup)

	for i, item := range b.items {
		agent := item.Agent
		if agent == "" {
			agent = "unknown"
		}
		g, ok := byAgent[agent]
		if !ok {
			g = &brainGroup{agent: agent}
			byAgent[agent] = g
			order = append(order, agent)
		}
		g.items = append(g.items, item)
		g.indices = append(g.indices, i)
	}

	groups := make([]brainGroup, 0, len(order))
	for _, agent := range order {
		groups = append(groups, *byAgent[agent])
	}
	return groups
}
