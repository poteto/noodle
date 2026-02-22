package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/poteto/noodle/loop"
)

const (
	defaultRefreshInterval = 2 * time.Second
	defaultStuckThreshold  = int64(120)
)

type Surface string

const (
	surfaceDashboard Surface = "dashboard"
	surfaceSession   Surface = "session"
	surfaceTrace     Surface = "trace"
	surfaceQueue     Surface = "queue"
	surfaceSteer     Surface = "steer"
	surfaceHelp      Surface = "help"
)

type TraceFilter string

const (
	traceFilterAll    TraceFilter = "all"
	traceFilterTools  TraceFilter = "tools"
	traceFilterThink  TraceFilter = "think"
	traceFilterTicket TraceFilter = "ticket"
)

type Options struct {
	RuntimeDir      string
	RefreshInterval time.Duration
	Now             func() time.Time
}

type Model struct {
	runtimeDir      string
	refreshInterval time.Duration
	now             func() time.Time

	width  int
	height int

	surface Surface
	overlay Surface

	snapshot Snapshot
	err      error

	selectedActive int
	selectedQueue  int
	sessionID      string

	traceFilter TraceFilter
	traceFollow bool
	traceOffset int

	steerInput        string
	steerMentionOpen  bool
	steerMentionItems []string
	steerMentionIndex int
	steerMentionStart int
	statusLine        string
}

type Snapshot struct {
	UpdatedAt time.Time
	LoopState string

	Sessions []Session
	Active   []Session
	Recent   []Session
	Queue    []QueueItem

	EventsBySession map[string][]EventLine
	TotalCostUSD    float64
}

type Session struct {
	ID                    string
	Status                string
	Provider              string
	Model                 string
	TotalCostUSD          float64
	DurationSeconds       int64
	LastActivity          time.Time
	CurrentAction         string
	Health                string
	ContextWindowUsagePct float64
	RetryCount            int
	IdleSeconds           int64
	StuckThresholdSeconds int64
	LoopState             string
}

type QueueItem struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Skill    string `json:"skill,omitempty"`
	Review   *bool  `json:"review,omitempty"`
}

type EventLine struct {
	At       time.Time
	Label    string
	Body     string
	Category TraceFilter
}

type tickMsg time.Time

type snapshotMsg struct {
	snapshot Snapshot
	err      error
}

type controlResultMsg struct {
	action string
	err    error
}

func NewModel(opts Options) Model {
	runtimeDir := strings.TrimSpace(opts.RuntimeDir)
	if runtimeDir == "" {
		runtimeDir = ".noodle"
	}
	interval := opts.RefreshInterval
	if interval <= 0 {
		interval = defaultRefreshInterval
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return Model{
		runtimeDir:      runtimeDir,
		refreshInterval: interval,
		now:             now,
		surface:         surfaceDashboard,
		traceFilter:     traceFilterAll,
		traceFollow:     true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(refreshSnapshotCmd(m.runtimeDir, m.now), tickCmd(m.refreshInterval))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		return m, tea.Batch(refreshSnapshotCmd(m.runtimeDir, m.now), tickCmd(m.refreshInterval))
	case snapshotMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.snapshot = msg.snapshot
		m.clampSelection()
		m.syncSessionSelection()
		return m, nil
	case controlResultMsg:
		if msg.err != nil {
			m.statusLine = fmt.Sprintf("%s failed: %v", msg.action, msg.err)
			return m, nil
		}
		m.statusLine = fmt.Sprintf("%s command sent", msg.action)
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	default:
		return m, nil
	}
}

func (m Model) View() string {
	var b strings.Builder
	base := m.baseSurface()
	b.WriteString(m.renderSurface(base))
	switch m.surface {
	case surfaceHelp:
		b.WriteString("\n\n")
		b.WriteString(renderHelp())
	case surfaceSteer:
		b.WriteString("\n\n")
		b.WriteString(m.renderSteer())
	}
	if m.statusLine != "" {
		b.WriteString("\n\n")
		b.WriteString("status: ")
		b.WriteString(m.statusLine)
	}
	if m.err != nil {
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render("error: "))
		b.WriteString(m.err.Error())
	}

	content := b.String()
	if m.width > 0 {
		target := m.width - 2
		if target < 40 {
			target = 40
		}
		return frameStyle.Width(target).Render(content)
	}
	return frameStyle.Render(content)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		if m.surface == surfaceSteer {
			if m.steerMentionOpen {
				m.closeSteerMentions()
				return m, nil
			}
			m.closeSteer()
			return m, nil
		}
		if m.surface == surfaceHelp {
			m.closeHelp()
			return m, nil
		}
		if m.surface == surfaceTrace {
			m.surface = surfaceSession
			m.traceFollow = true
			m.traceOffset = 0
			return m, nil
		}
		if m.surface != surfaceDashboard {
			m.surface = surfaceDashboard
			return m, nil
		}
		return m, nil
	}

	if m.surface == surfaceSteer {
		return m.handleSteerKey(msg)
	}
	if m.surface == surfaceHelp {
		if strings.ToLower(msg.String()) == "?" {
			m.closeHelp()
		}
		return m, nil
	}

	key := strings.ToLower(msg.String())
	switch key {
	case "?":
		m.openHelp()
		return m, nil
	case "s":
		return m, m.openSteer()
	case "p":
		action := "pause"
		if strings.EqualFold(m.snapshot.LoopState, "paused") {
			action = "resume"
		}
		return m, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{Action: action})
	case "d":
		return m, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{Action: "drain"})
	case "q":
		m.surface = surfaceQueue
		return m, nil
	}

	switch m.surface {
	case surfaceDashboard:
		return m.handleDashboardKey(msg)
	case surfaceSession:
		return m.handleSessionKey(msg)
	case surfaceTrace:
		return m.handleTraceKey(msg)
	case surfaceQueue:
		return m.handleQueueKey(msg)
	default:
		return m, nil
	}
}

func (m Model) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "up", "k":
		if m.selectedActive > 0 {
			m.selectedActive--
		}
		return m, nil
	case "down", "j":
		if m.selectedActive < len(m.snapshot.Active)-1 {
			m.selectedActive++
		}
		return m, nil
	}

	if msg.Type == tea.KeyEnter && len(m.snapshot.Active) > 0 {
		m.sessionID = m.snapshot.Active[m.selectedActive].ID
		m.surface = surfaceSession
	}
	return m, nil
}

func (m Model) handleSessionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "t":
		m.surface = surfaceTrace
		m.traceFollow = true
		m.traceOffset = 0
		return m, nil
	case "k":
		if m.sessionID == "" {
			return m, nil
		}
		return m, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{
			Action: "kill",
			Name:   m.sessionID,
		})
	}
	return m, nil
}

func (m Model) handleTraceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	lines := m.filteredTraceLines()
	maxStart := 0
	if len(lines) > m.traceHeight() {
		maxStart = len(lines) - m.traceHeight()
	}

	switch strings.ToLower(msg.String()) {
	case "f":
		m.traceFilter = nextTraceFilter(m.traceFilter)
		m.traceFollow = true
		m.traceOffset = 0
	case "g":
		if strings.EqualFold(msg.String(), "G") {
			m.traceFollow = true
			m.traceOffset = 0
		}
	case "up", "k":
		if m.traceFollow {
			m.traceFollow = false
			m.traceOffset = maxStart
		}
		if m.traceOffset > 0 {
			m.traceOffset--
		}
	case "down", "j":
		if m.traceFollow {
			return m, nil
		}
		if m.traceOffset < maxStart {
			m.traceOffset++
		} else {
			m.traceFollow = true
			m.traceOffset = 0
		}
	}
	return m, nil
}

func (m Model) handleQueueKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "up", "k":
		if m.selectedQueue > 0 {
			m.selectedQueue--
		}
	case "down", "j":
		if m.selectedQueue < len(m.snapshot.Queue)-1 {
			m.selectedQueue++
		}
	}
	return m, nil
}

func (m Model) handleSteerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		if m.steerMentionOpen {
			m.closeSteerMentions()
			return m, nil
		}
		m.closeSteer()
		return m, nil
	case tea.KeyEnter:
		if m.steerMentionOpen && len(m.steerMentionItems) > 0 {
			selection := m.steerMentionItems[m.steerMentionIndex]
			m.applySteerMention(selection)
			return m, nil
		}
		sendCmd, ok := m.submitSteer()
		if !ok {
			return m, nil
		}
		return m, sendCmd
	case tea.KeyUp:
		if m.steerMentionOpen && len(m.steerMentionItems) > 0 && m.steerMentionIndex > 0 {
			m.steerMentionIndex--
		}
		return m, nil
	case tea.KeyDown:
		if m.steerMentionOpen && m.steerMentionIndex < len(m.steerMentionItems)-1 {
			m.steerMentionIndex++
		}
		return m, nil
	case tea.KeyBackspace, tea.KeyCtrlH:
		m.steerInput = dropLastRune(m.steerInput)
		m.refreshSteerMentions()
		return m, nil
	case tea.KeyRunes:
		m.steerInput += string(msg.Runes)
		m.refreshSteerMentions()
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) renderSurface(surface Surface) string {
	switch surface {
	case surfaceSession:
		return m.renderSession()
	case surfaceTrace:
		return m.renderTrace()
	case surfaceQueue:
		return m.renderQueue()
	default:
		return m.renderDashboard()
	}
}

func (m Model) renderDashboard() string {
	bodyWidth := m.contentWidth()
	var b strings.Builder
	b.WriteString(titleStyle.Render("noodle"))
	b.WriteString(dimStyle.Render(" | "))
	b.WriteString(accentStyle.Render("cooking"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%s %s | %s %d | %s %d | %s %s",
		labelStyle.Render("status"),
		loopStateLabel(m.snapshot.LoopState),
		labelStyle.Render("active"),
		len(m.snapshot.Active),
		labelStyle.Render("queue"),
		len(m.snapshot.Queue),
		labelStyle.Render("total"),
		costStyle.Render(fmt.Sprintf("$%.2f", m.snapshot.TotalCostUSD)),
	)
	if !m.snapshot.UpdatedAt.IsZero() {
		fmt.Fprintf(&b, " | updated %s ago", mutedStyle.Render(ageLabel(m.now(), m.snapshot.UpdatedAt)))
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(36, bodyWidth))))
	b.WriteString("\n")
	b.WriteString(sectionLine("Active Cooks", bodyWidth))
	b.WriteString("\n")
	if len(m.snapshot.Active) == 0 {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("(none)"))
		b.WriteString("\n")
	} else {
		for i, session := range m.snapshot.Active {
			cursor := " "
			if i == m.selectedActive {
				cursor = accentStyle.Render(">")
			}
			action := nonEmpty(session.CurrentAction, "(idle)")
			line := fmt.Sprintf(
				"%s %s %-22s %-20s %-26s %6s ago\n",
				cursor,
				healthDot(session.Health),
				session.ID,
				modelLabel(session),
				trimTo(action, 22),
				ageLabel(m.now(), session.LastActivity),
			)
			if i == m.selectedActive {
				line = selectedRowStyle.Render(trimTo(strings.TrimSuffix(line, "\n"), bodyWidth))
				b.WriteString(line)
				b.WriteString("\n")
				continue
			}
			b.WriteString(line)
		}
	}

	b.WriteString("\nRecent\n")
	b.WriteString(sectionLine("Recent", bodyWidth))
	b.WriteString("\n")
	if len(m.snapshot.Recent) == 0 {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("(none)"))
		b.WriteString("\n")
	} else {
		limit := 6
		if len(m.snapshot.Recent) < limit {
			limit = len(m.snapshot.Recent)
		}
		for i := 0; i < limit; i++ {
			s := m.snapshot.Recent[i]
			fmt.Fprintf(
				&b,
				"  %s %-22s %-20s %8s %s\n",
				statusIcon(s.Status)+" "+statusLabel(s.Status),
				s.ID,
				modelLabel(s),
				durationLabel(s.DurationSeconds),
				costStyle.Render(fmt.Sprintf("$%.2f", s.TotalCostUSD)),
			)
		}
	}

	b.WriteString("\n")
	b.WriteString(sectionLine("Up Next", bodyWidth))
	b.WriteString("\n")
	if len(m.snapshot.Queue) == 0 {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("(empty)"))
		b.WriteString("\n")
	} else {
		limit := 6
		if len(m.snapshot.Queue) < limit {
			limit = len(m.snapshot.Queue)
		}
		for i := 0; i < limit; i++ {
			item := m.snapshot.Queue[i]
			fmt.Fprintf(
				&b,
				"  %d. %-22s %-12s %-20s\n",
				i+1,
				item.ID,
				infoStyle.Render(nonEmpty(item.Provider, "claude")),
				nonEmpty(item.Model, "(default)"),
			)
		}
	}

	b.WriteString("\n")
	b.WriteString(keybarStyle.Render("enter inspect | q queue | s steer | p pause/resume | d drain | ? help | ctrl+c quit"))
	return b.String()
}

func (m Model) renderSession() string {
	bodyWidth := m.contentWidth()
	session, ok := m.sessionByID(m.sessionID)
	if !ok {
		return errorStyle.Render("session not found") + "\n\n" + keybarStyle.Render("esc back")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s | %s\n", titleStyle.Render("Session Detail"), accentStyle.Render(session.ID))
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(36, bodyWidth))))
	b.WriteString("\n")

	fmt.Fprintf(&b, "%s %s %s\n", labelStyle.Render("Status:"), statusLabel(session.Status), healthDot(session.Health))
	fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Provider:"), nonEmpty(session.Provider, "-"))
	fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Model:"), nonEmpty(session.Model, "-"))
	fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Duration:"), durationLabel(session.DurationSeconds))
	fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Cost:"), costStyle.Render(fmt.Sprintf("$%.2f", session.TotalCostUSD)))
	fmt.Fprintf(&b, "%s %d\n", labelStyle.Render("Retries:"), session.RetryCount)
	fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Worktree:"), mutedStyle.Render(".worktrees/"+session.ID))

	lines := m.snapshot.EventsBySession[session.ID]
	b.WriteString("\n")
	b.WriteString(sectionLine("Recent Events", bodyWidth))
	b.WriteString("\n")
	if len(lines) == 0 {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("(none)"))
		b.WriteString("\n")
	} else {
		start := len(lines) - 8
		if start < 0 {
			start = 0
		}
		const eventPrefixWidth = 29 // "  HH:MM:SS  " + 14-char label + " | "
		eventBodyWidth := bodyWidth - eventPrefixWidth
		if eventBodyWidth < 12 {
			eventBodyWidth = 12
		}
		for _, line := range lines[start:] {
			label := strings.TrimSpace(line.Label)
			if label == "" {
				label = "-"
			}
			labelCell := padRight(label, 14)
			prefix := fmt.Sprintf("  %s  %s | ", line.At.Format("15:04:05"), eventLabel(labelCell))
			continuationPrefix := "  " + strings.Repeat(" ", 8) + "  " + strings.Repeat(" ", 14) + " | "
			wrapped := wrapPlainText(line.Body, eventBodyWidth)
			for i, segment := range wrapped {
				if i == 0 {
					b.WriteString(prefix)
				} else {
					b.WriteString(continuationPrefix)
				}
				b.WriteString(segment)
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(keybarStyle.Render("t trace | k kill | s steer | esc back | ? help"))
	return b.String()
}

func (m Model) renderTrace() string {
	bodyWidth := m.contentWidth()
	session, ok := m.sessionByID(m.sessionID)
	if !ok {
		return errorStyle.Render("trace unavailable: session not found") + "\n\n" + keybarStyle.Render("esc back")
	}
	lines := m.filteredTraceLines()
	height := m.traceHeight()
	start := 0
	if len(lines) > height {
		if m.traceFollow {
			start = len(lines) - height
		} else {
			start = m.traceOffset
			maxStart := len(lines) - height
			if start < 0 {
				start = 0
			}
			if start > maxStart {
				start = maxStart
			}
		}
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s | %s | %s %s\n",
		titleStyle.Render("Trace"),
		accentStyle.Render(session.ID),
		mutedStyle.Render("filter:"),
		infoStyle.Render(string(m.traceFilter)),
	)
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(36, bodyWidth))))
	b.WriteString("\n")
	if len(lines) == 0 {
		b.WriteString(dimStyle.Render("(no events)\n"))
	} else {
		for _, line := range lines[start:end] {
			fmt.Fprintf(&b, "%s  %-14s | %s\n", line.At.Format("15:04:05"), eventLabel(line.Label), line.Body)
		}
	}
	if m.traceFollow {
		b.WriteString("\n")
		b.WriteString(infoStyle.Render("[auto-scroll]"))
	}
	b.WriteString("\n")
	b.WriteString(keybarStyle.Render("f filter | G bottom | esc back | ? help"))
	return b.String()
}

func (m Model) renderQueue() string {
	bodyWidth := m.contentWidth()
	var b strings.Builder
	b.WriteString(titleStyle.Render("Queue\n"))
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(36, bodyWidth))))
	b.WriteString("\n")
	if len(m.snapshot.Queue) == 0 {
		b.WriteString(dimStyle.Render("(empty)\n"))
	} else {
		for i, item := range m.snapshot.Queue {
			cursor := " "
			if i == m.selectedQueue {
				cursor = accentStyle.Render(">")
			}
			review := "default"
			if item.Review != nil {
				if *item.Review {
					review = "review"
				} else {
					review = "no-review"
				}
			}
			line := fmt.Sprintf(
				"%s %2d. %-24s %-12s %-18s %s\n",
				cursor,
				i+1,
				item.ID,
				infoStyle.Render(nonEmpty(item.Provider, "-")),
				nonEmpty(item.Model, "-"),
				review,
			)
			if i == m.selectedQueue {
				line = selectedRowStyle.Render(trimTo(strings.TrimSuffix(line, "\n"), bodyWidth))
				b.WriteString(line)
				b.WriteString("\n")
				continue
			}
			b.WriteString(line)
		}
	}
	b.WriteString("\n")
	b.WriteString(keybarStyle.Render("esc back | s steer | ? help"))
	return b.String()
}

func renderHelp() string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("Keys"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 24)))
	b.WriteString("\n")
	b.WriteString("Global: s steer | p pause/resume | d drain | ? help | ctrl+c quit\n")
	b.WriteString("Dashboard: enter inspect | q queue | up/down move\n")
	b.WriteString("Session: t trace | k kill | esc back\n")
	b.WriteString("Trace: f filter | G bottom | up/down scroll | esc back\n")
	b.WriteString("Steer: type @target + instruction; @everyone for broadcast")
	return b.String()
}

func (m Model) renderSteer() string {
	bodyWidth := m.contentWidth()
	var b strings.Builder
	b.WriteString(titleStyle.Render("Steer"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", max(36, bodyWidth))))
	b.WriteString("\n\n")
	b.WriteString(sectionStyle.Render("Instruction"))
	b.WriteString("\n")
	if strings.TrimSpace(m.steerInput) == "" {
		b.WriteString(dimStyle.Render("> @cook-a focus on tests, keep commits small"))
	} else {
		b.WriteString("> ")
		b.WriteString(m.steerInput)
	}

	if m.steerMentionOpen && len(m.steerMentionItems) > 0 {
		b.WriteString("\n\n")
		b.WriteString(sectionLine("Mentions", bodyWidth))
		b.WriteString("\n")
		for i, mention := range m.steerMentionItems {
			row := "  " + mention
			if i == m.steerMentionIndex {
				row = selectedRowStyle.Render(trimTo(strings.TrimSpace(row), bodyWidth))
			}
			b.WriteString(row)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Type @ to mention cooks. Enter submits. Esc closes mentions, then closes steer."))
	return b.String()
}

func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 96
	}
	width := m.width - 6
	if width < 40 {
		return 40
	}
	return width
}

func (m *Model) clampSelection() {
	if m.selectedActive < 0 {
		m.selectedActive = 0
	}
	if m.selectedActive >= len(m.snapshot.Active) && len(m.snapshot.Active) > 0 {
		m.selectedActive = len(m.snapshot.Active) - 1
	}
	if m.selectedQueue < 0 {
		m.selectedQueue = 0
	}
	if m.selectedQueue >= len(m.snapshot.Queue) && len(m.snapshot.Queue) > 0 {
		m.selectedQueue = len(m.snapshot.Queue) - 1
	}
}

func (m *Model) syncSessionSelection() {
	if m.sessionID != "" {
		if _, ok := m.sessionByID(m.sessionID); ok {
			return
		}
	}
	if len(m.snapshot.Active) > 0 {
		m.sessionID = m.snapshot.Active[m.selectedActive].ID
	}
}

func (m Model) sessionByID(id string) (Session, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Session{}, false
	}
	for _, session := range m.snapshot.Sessions {
		if session.ID == id {
			return session, true
		}
	}
	return Session{}, false
}

func (m *Model) baseSurface() Surface {
	switch m.surface {
	case surfaceHelp, surfaceSteer:
		if m.overlay == "" {
			return surfaceDashboard
		}
		return m.overlay
	default:
		return m.surface
	}
}

func (m *Model) openHelp() {
	m.overlay = m.baseSurface()
	m.surface = surfaceHelp
}

func (m *Model) closeHelp() {
	if m.overlay == "" {
		m.surface = surfaceDashboard
		return
	}
	m.surface = m.overlay
}

func (m *Model) openSteer() tea.Cmd {
	m.overlay = m.baseSurface()
	m.surface = surfaceSteer
	m.steerInput = ""
	m.steerMentionOpen = false
	m.steerMentionItems = nil
	m.steerMentionIndex = 0
	m.steerMentionStart = -1
	return nil
}

func (m *Model) closeSteer() {
	m.steerInput = ""
	m.closeSteerMentions()
	if m.overlay == "" {
		m.surface = surfaceDashboard
		return
	}
	m.surface = m.overlay
}

func (m *Model) submitSteer() (tea.Cmd, bool) {
	targets, prompt, err := parseSteerInput(m.steerInput, m.steerTargets())
	if err != nil {
		m.statusLine = "steer failed: " + err.Error()
		return nil, false
	}
	m.closeSteer()
	cmds := make([]tea.Cmd, 0, len(targets))
	for _, target := range targets {
		cmds = append(cmds, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{
			Action: "steer",
			Target: target,
			Prompt: prompt,
		}))
	}
	if len(cmds) == 1 {
		return cmds[0], true
	}
	return tea.Batch(cmds...), true
}

func (m Model) steerTargets() []string {
	targets := []string{"sous-chef"}
	seen := map[string]struct{}{"sous-chef": {}}
	for _, session := range m.snapshot.Active {
		id := strings.TrimSpace(session.ID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		targets = append(targets, id)
	}
	sort.Strings(targets[1:])
	return targets
}

func (m Model) filteredTraceLines() []EventLine {
	lines := m.snapshot.EventsBySession[m.sessionID]
	if m.traceFilter == traceFilterAll {
		return lines
	}
	filtered := make([]EventLine, 0, len(lines))
	for _, line := range lines {
		if line.Category == m.traceFilter {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

func (m Model) traceHeight() int {
	if m.height <= 0 {
		return 16
	}
	h := m.height - 8
	if h < 4 {
		return 4
	}
	return h
}

func nextTraceFilter(filter TraceFilter) TraceFilter {
	switch filter {
	case traceFilterAll:
		return traceFilterTools
	case traceFilterTools:
		return traceFilterThink
	case traceFilterThink:
		return traceFilterTicket
	default:
		return traceFilterAll
	}
}

func refreshSnapshotCmd(runtimeDir string, now func() time.Time) tea.Cmd {
	return func() tea.Msg {
		snapshot, err := loadSnapshot(runtimeDir, now())
		return snapshotMsg{snapshot: snapshot, err: err}
	}
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func sendControlCmd(runtimeDir string, now func() time.Time, cmd loop.ControlCommand) tea.Cmd {
	return func() tea.Msg {
		action := cmd.Action
		if strings.TrimSpace(action) == "" {
			action = "unknown"
		}
		cmd.Action = action
		cmd.At = now().UTC()
		cmd.ID = fmt.Sprintf("tui-%d", cmd.At.UnixNano())
		path := filepath.Join(runtimeDir, "control.ndjson")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return controlResultMsg{action: action, err: fmt.Errorf("create control directory: %w", err)}
		}
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return controlResultMsg{action: action, err: fmt.Errorf("open control file: %w", err)}
		}
		defer file.Close()

		line, err := json.Marshal(cmd)
		if err != nil {
			return controlResultMsg{action: action, err: fmt.Errorf("encode control command: %w", err)}
		}
		if _, err := file.Write(append(line, '\n')); err != nil {
			return controlResultMsg{action: action, err: fmt.Errorf("append control command: %w", err)}
		}
		return controlResultMsg{action: action}
	}
}
