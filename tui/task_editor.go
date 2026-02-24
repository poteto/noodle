package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/poteto/noodle/loop"
	"github.com/poteto/noodle/tui/components"
)

// TaskEditor is the inline task creator/editor overlay.
type TaskEditor struct {
	open bool

	title    string
	taskType int
	model    int
	provider int
	skill    string
	field    int // active field index (0-4)
}

var (
	taskTypes = []string{"execute", "plan", "quality", "reflect", "prioritize"}
	models    = []string{"claude-opus-4-6", "claude-sonnet-4-6", "claude-haiku-4-5", "gpt-5.3-codex"}
	providers = []string{"claude", "codex"}
)

const (
	fieldTitle    = 0
	fieldType     = 1
	fieldModel    = 2
	fieldProvider = 3
	fieldSkill    = 4
	fieldCount    = 5
)

// OpenNew opens the task editor in create mode.
func (e *TaskEditor) OpenNew() {
	e.open = true
	e.title = ""
	e.taskType = 0
	e.model = 0
	e.provider = 0
	e.skill = ""
	e.field = 0
}

// Close closes the editor without submitting.
func (e *TaskEditor) Close() {
	e.open = false
}

// HandleKey processes key events when the editor is open.
func (e *TaskEditor) HandleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case msg.Code == tea.KeyEsc:
		e.Close()
		return nil, true
	case isShiftTab(msg):
		e.field = (e.field - 1 + fieldCount) % fieldCount
		return nil, true
	case msg.Code == tea.KeyTab:
		e.field = (e.field + 1) % fieldCount
		return nil, true
	case msg.Code == tea.KeyLeft:
		e.cyclePrev()
		return nil, true
	case msg.Code == tea.KeyRight:
		e.cycleNext()
		return nil, true
	case msg.Code == tea.KeyBackspace || isCtrlH(msg):
		if e.field == fieldTitle {
			e.title = dropLastRune(e.title)
		} else if e.field == fieldSkill {
			e.skill = dropLastRune(e.skill)
		}
		return nil, true
	case msg.Code == tea.KeySpace:
		if e.field == fieldTitle {
			e.title += " "
		} else if e.field == fieldSkill {
			e.skill += " "
		}
		return nil, true
	case msg.Text != "":
		if e.field == fieldTitle {
			e.title += msg.Text
		} else if e.field == fieldSkill {
			e.skill += msg.Text
		}
		return nil, true
	}
	return nil, false
}

func isShiftTab(msg tea.KeyPressMsg) bool {
	key := msg.Key()
	return key.Code == tea.KeyTab && (key.Mod&tea.ModShift) != 0
}

func (e *TaskEditor) cyclePrev() {
	switch e.field {
	case fieldType:
		e.taskType = (e.taskType - 1 + len(taskTypes)) % len(taskTypes)
	case fieldModel:
		e.model = (e.model - 1 + len(models)) % len(models)
	case fieldProvider:
		e.provider = (e.provider - 1 + len(providers)) % len(providers)
	}
}

func (e *TaskEditor) cycleNext() {
	switch e.field {
	case fieldType:
		e.taskType = (e.taskType + 1) % len(taskTypes)
	case fieldModel:
		e.model = (e.model + 1) % len(models)
	case fieldProvider:
		e.provider = (e.provider + 1) % len(providers)
	}
}

// Submit creates the control command for the task.
func (e *TaskEditor) Submit(runtimeDir string, now func() time.Time) tea.Cmd {
	title := strings.TrimSpace(e.title)
	if title == "" {
		return nil
	}

	taskKey := taskTypes[e.taskType]
	itemID := fmt.Sprintf("%s-%d", taskKey, now().UnixMilli())
	cmd := loop.ControlCommand{
		Action:   "enqueue",
		Item:     itemID,
		Prompt:   title,
		TaskKey:  taskKey,
		Provider: providers[e.provider],
		Model:    models[e.model],
		Skill:    strings.TrimSpace(e.skill),
	}

	e.Close()
	return sendControlCmd(runtimeDir, now, cmd)
}

// Render renders the task editor overlay.
func (e *TaskEditor) Render(width int) string {
	t := components.DefaultTheme
	if width < 30 {
		width = 30
	}
	innerWidth := width - 4

	mode := "New Task"

	header := lipgloss.NewStyle().
		Foreground(t.Brand).
		Bold(true).
		Render(mode)

	fields := []struct {
		label    string
		value    string
		editable bool
	}{
		{"Title", e.title, true},
		{"Type", taskTypes[e.taskType], false},
		{"Model", models[e.model], false},
		{"Provider", providers[e.provider], false},
		{"Skill", e.skill, true},
	}

	var rows []string
	for i, f := range fields {
		label := lipgloss.NewStyle().Foreground(t.Dim).Width(10).Render(f.label)
		value := f.value
		if value == "" && f.editable {
			value = "(empty)"
		}
		if !f.editable {
			value = "← " + value + " →"
		}
		if i == e.field {
			value = lipgloss.NewStyle().Foreground(t.Brand).Bold(true).Render(value)
		} else {
			value = lipgloss.NewStyle().Foreground(t.Secondary).Render(value)
		}
		rows = append(rows, label+" "+value)
	}

	body := strings.Join(rows, "\n")
	footer := dimStyle.Render("tab: next field · ←→: cycle · enter: submit · esc: cancel")

	card := &components.Card{
		Title:       header,
		Body:        body,
		Footer:      footer,
		BorderColor: t.Brand,
	}

	_ = innerWidth
	return card.Render(width)
}
