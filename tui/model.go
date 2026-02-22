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
	"github.com/poteto/noodle/event"
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
		for _, line := range lines[start:] {
			fmt.Fprintf(
				&b,
				"  %s  %-14s | %s\n",
				line.At.Format("15:04:05"),
				eventLabel(line.Label),
				trimTo(line.Body, 70),
			)
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

func parseSteerInput(raw string, validTargets []string) ([]string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, "", fmt.Errorf("type @target and an instruction")
	}

	valid := map[string]string{}
	for _, target := range validTargets {
		key := strings.ToLower(strings.TrimSpace(target))
		if key == "" {
			continue
		}
		valid[key] = target
	}

	mentions := make([]string, 0, 2)
	words := make([]string, 0, 8)
	for _, token := range strings.Fields(raw) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if !strings.HasPrefix(token, "@") {
			words = append(words, token)
			continue
		}
		mention := strings.TrimPrefix(token, "@")
		mention = strings.TrimSpace(strings.TrimRight(mention, ",.;:"))
		if mention == "" {
			continue
		}
		mentions = append(mentions, mention)
	}
	if len(mentions) == 0 {
		return nil, "", fmt.Errorf("missing @target mention")
	}

	prompt := strings.TrimSpace(strings.Join(words, " "))
	if prompt == "" {
		return nil, "", fmt.Errorf("instruction text is required")
	}

	resolved := make([]string, 0, len(validTargets))
	for _, mention := range mentions {
		mentionKey := strings.ToLower(mention)
		if mentionKey == "everyone" {
			resolved = append(resolved, validTargets...)
			continue
		}
		canonical, ok := valid[mentionKey]
		if !ok {
			return nil, "", fmt.Errorf("unknown target @%s", mention)
		}
		resolved = append(resolved, canonical)
	}
	resolved = uniqueStrings(resolved)
	if len(resolved) == 0 {
		return nil, "", fmt.Errorf("no valid targets selected")
	}
	return resolved, prompt, nil
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (m *Model) refreshSteerMentions() {
	start, query, ok := mentionQuery(m.steerInput)
	if !ok {
		m.closeSteerMentions()
		return
	}
	candidates := mentionCandidates(query, m.steerTargets())
	if len(candidates) == 0 {
		m.closeSteerMentions()
		return
	}
	m.steerMentionOpen = true
	m.steerMentionStart = start
	m.steerMentionItems = candidates
	if m.steerMentionIndex >= len(candidates) {
		m.steerMentionIndex = len(candidates) - 1
	}
	if m.steerMentionIndex < 0 {
		m.steerMentionIndex = 0
	}
}

func mentionQuery(input string) (int, string, bool) {
	if input == "" {
		return 0, "", false
	}
	start := len(input) - 1
	for start >= 0 && input[start] != ' ' && input[start] != '\t' && input[start] != '\n' {
		start--
	}
	start++
	if start >= len(input) || input[start] != '@' {
		return 0, "", false
	}
	return start, strings.ToLower(strings.TrimSpace(input[start+1:])), true
}

func mentionCandidates(query string, targets []string) []string {
	all := []string{"@everyone"}
	for _, target := range targets {
		all = append(all, "@"+target)
	}
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return all
	}
	out := make([]string, 0, len(all))
	for _, candidate := range all {
		if strings.HasPrefix(strings.ToLower(strings.TrimPrefix(candidate, "@")), query) {
			out = append(out, candidate)
		}
	}
	return out
}

func (m *Model) applySteerMention(selection string) {
	if m.steerMentionStart < 0 || m.steerMentionStart > len(m.steerInput) {
		m.steerInput = strings.TrimSpace(m.steerInput + " " + selection + " ")
		m.closeSteerMentions()
		return
	}
	prefix := m.steerInput[:m.steerMentionStart]
	m.steerInput = prefix + selection + " "
	m.closeSteerMentions()
}

func (m *Model) closeSteerMentions() {
	m.steerMentionOpen = false
	m.steerMentionItems = nil
	m.steerMentionIndex = 0
	m.steerMentionStart = -1
}

func dropLastRune(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 1 {
		return ""
	}
	return string(runes[:len(runes)-1])
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

func loadSnapshot(runtimeDir string, now time.Time) (Snapshot, error) {
	sessions, err := readSessions(runtimeDir)
	if err != nil {
		return Snapshot{}, err
	}
	queue, err := readQueue(filepath.Join(runtimeDir, "queue.json"))
	if err != nil {
		return Snapshot{}, err
	}

	eventsBySession := make(map[string][]EventLine, len(sessions))
	reader := event.NewEventReader(runtimeDir)
	for i := range sessions {
		events, err := reader.ReadSession(sessions[i].ID, event.EventFilter{})
		if err != nil {
			return Snapshot{}, err
		}
		lines := mapEventLines(events)
		eventsBySession[sessions[i].ID] = lines
		if sessions[i].CurrentAction == "" && len(lines) > 0 {
			sessions[i].CurrentAction = lines[len(lines)-1].Body
		}
	}

	active := make([]Session, 0, len(sessions))
	recent := make([]Session, 0, len(sessions))
	totalCost := 0.0
	loopState := "running"
	for _, session := range sessions {
		totalCost += session.TotalCostUSD
		loopState = pickLoopState(loopState, session.LoopState)
		if isActiveStatus(session.Status) {
			active = append(active, session)
		} else {
			recent = append(recent, session)
		}
	}

	sort.SliceStable(active, func(i, j int) bool {
		if active[i].LastActivity.Equal(active[j].LastActivity) {
			return active[i].ID < active[j].ID
		}
		return active[i].LastActivity.After(active[j].LastActivity)
	})
	sort.SliceStable(recent, func(i, j int) bool {
		if recent[i].LastActivity.Equal(recent[j].LastActivity) {
			return recent[i].ID < recent[j].ID
		}
		return recent[i].LastActivity.After(recent[j].LastActivity)
	})

	return Snapshot{
		UpdatedAt:       now.UTC(),
		LoopState:       loopState,
		Sessions:        sessions,
		Active:          active,
		Recent:          recent,
		Queue:           queue,
		EventsBySession: eventsBySession,
		TotalCostUSD:    totalCost,
	}, nil
}

func readSessions(runtimeDir string) ([]Session, error) {
	path := filepath.Join(runtimeDir, "sessions")
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}

	sessions := make([]Session, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionID := strings.TrimSpace(entry.Name())
		if sessionID == "" {
			continue
		}
		metaPath := filepath.Join(path, sessionID, "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read session meta %s: %w", sessionID, err)
		}

		var raw struct {
			SessionID             string  `json:"session_id"`
			Status                string  `json:"status"`
			Provider              string  `json:"provider"`
			Model                 string  `json:"model"`
			TotalCostUSD          float64 `json:"total_cost_usd"`
			DurationSeconds       int64   `json:"duration_seconds"`
			LastActivity          string  `json:"last_activity"`
			CurrentAction         string  `json:"current_action"`
			Health                string  `json:"health"`
			ContextWindowUsagePct float64 `json:"context_window_usage_pct"`
			RetryCount            int     `json:"retry_count"`
			IdleSeconds           int64   `json:"idle_seconds"`
			StuckThresholdSeconds int64   `json:"stuck_threshold_seconds"`
			LoopState             string  `json:"loop_state"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse session meta %s: %w", sessionID, err)
		}

		if strings.TrimSpace(raw.SessionID) != "" {
			sessionID = strings.TrimSpace(raw.SessionID)
		}
		lastActivity := parseTime(raw.LastActivity)
		status := strings.ToLower(strings.TrimSpace(raw.Status))
		health := deriveHealth(
			status,
			raw.Health,
			raw.ContextWindowUsagePct,
			raw.IdleSeconds,
			raw.StuckThresholdSeconds,
		)

		sessions = append(sessions, Session{
			ID:                    sessionID,
			Status:                nonEmpty(status, "running"),
			Provider:              strings.TrimSpace(raw.Provider),
			Model:                 strings.TrimSpace(raw.Model),
			TotalCostUSD:          raw.TotalCostUSD,
			DurationSeconds:       raw.DurationSeconds,
			LastActivity:          lastActivity,
			CurrentAction:         strings.TrimSpace(raw.CurrentAction),
			Health:                health,
			ContextWindowUsagePct: raw.ContextWindowUsagePct,
			RetryCount:            raw.RetryCount,
			IdleSeconds:           raw.IdleSeconds,
			StuckThresholdSeconds: raw.StuckThresholdSeconds,
			LoopState:             strings.TrimSpace(raw.LoopState),
		})
	}

	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].LastActivity.Equal(sessions[j].LastActivity) {
			return sessions[i].ID < sessions[j].ID
		}
		return sessions[i].LastActivity.After(sessions[j].LastActivity)
	})
	return sessions, nil
}

func readQueue(path string) ([]QueueItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read queue.json: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}

	var wrapped struct {
		Items []QueueItem `json:"items"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil {
		return wrapped.Items, nil
	}

	var items []QueueItem
	if err := json.Unmarshal(data, &items); err == nil {
		return items, nil
	}
	return nil, fmt.Errorf("parse queue.json")
}

func mapEventLines(events []event.Event) []EventLine {
	lines := make([]EventLine, 0, len(events))
	for _, ev := range events {
		line := EventLine{
			At:       ev.Timestamp,
			Label:    "Event",
			Body:     "",
			Category: traceFilterAll,
		}
		switch ev.Type {
		case event.EventCost:
			line.Label = "Cost"
			line.Category = traceFilterAll
			line.Body = formatCost(ev.Payload)
		case event.EventTicketClaim, event.EventTicketProgress, event.EventTicketDone, event.EventTicketBlocked, event.EventTicketRelease:
			line.Label = "Ticket"
			line.Category = traceFilterTicket
			line.Body = formatTicket(ev.Payload, string(ev.Type))
		case event.EventAction:
			label, body, category := formatAction(ev.Payload)
			line.Label = label
			line.Body = body
			line.Category = category
		default:
			line.Label = strings.Title(strings.ReplaceAll(string(ev.Type), "_", " "))
			line.Body = summarizePayload(ev.Payload)
			line.Category = traceFilterAll
		}
		if line.Body == "" {
			line.Body = "(no details)"
		}
		lines = append(lines, line)
	}
	return lines
}

func formatCost(payload json.RawMessage) string {
	var body struct {
		CostUSD   float64 `json:"cost_usd"`
		TokensIn  int     `json:"tokens_in"`
		TokensOut int     `json:"tokens_out"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return summarizePayload(payload)
	}
	if body.TokensIn == 0 && body.TokensOut == 0 {
		return fmt.Sprintf("$%.2f", body.CostUSD)
	}
	return fmt.Sprintf("$%.2f | %s in / %s out", body.CostUSD, shortInt(body.TokensIn), shortInt(body.TokensOut))
}

func formatTicket(payload json.RawMessage, fallback string) string {
	var body struct {
		Target  string `json:"target"`
		Summary string `json:"summary"`
		Outcome string `json:"outcome"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return fallback
	}
	if body.Summary != "" {
		return body.Summary
	}
	if body.Outcome != "" {
		return body.Outcome
	}
	if body.Reason != "" {
		return body.Reason
	}
	if body.Target != "" {
		return body.Target
	}
	return fallback
}

func formatAction(payload json.RawMessage) (label string, body string, category TraceFilter) {
	var action struct {
		Tool    string `json:"tool"`
		Action  string `json:"action"`
		Summary string `json:"summary"`
		Message string `json:"message"`
		Command string `json:"command"`
		Path    string `json:"path"`
	}
	_ = json.Unmarshal(payload, &action)

	tool := strings.ToLower(strings.TrimSpace(nonEmpty(action.Tool, action.Action)))
	switch tool {
	case "read":
		label = "Read"
		category = traceFilterTools
	case "edit", "write":
		label = "Edit"
		category = traceFilterTools
	case "bash", "shell":
		label = "Bash"
		category = traceFilterTools
	case "glob":
		label = "Glob"
		category = traceFilterTools
	case "grep":
		label = "Grep"
		category = traceFilterTools
	case "think":
		label = "Think"
		category = traceFilterThink
	default:
		// If no tool was recorded, this is usually assistant narration.
		label = "Think"
		category = traceFilterThink
	}

	body = firstNonEmpty(
		strings.TrimSpace(action.Summary),
		strings.TrimSpace(action.Message),
		strings.TrimSpace(action.Command),
		strings.TrimSpace(action.Path),
	)
	if body == "" {
		body = summarizePayload(payload)
	}
	return label, body, category
}

func summarizePayload(payload json.RawMessage) string {
	value := strings.TrimSpace(string(payload))
	if value == "" || value == "null" {
		return ""
	}
	return trimTo(value, 80)
}

func deriveHealth(status, explicit string, contextUsage float64, idleSeconds, stuckThresholdSeconds int64) string {
	explicit = strings.ToLower(strings.TrimSpace(explicit))
	if explicit == "green" || explicit == "yellow" || explicit == "red" {
		return explicit
	}

	status = strings.ToLower(strings.TrimSpace(status))
	if status == "failed" || status == "stuck" {
		return "red"
	}
	if status == "exited" || status == "killed" || status == "completed" {
		return "yellow"
	}

	if stuckThresholdSeconds <= 0 {
		stuckThresholdSeconds = defaultStuckThreshold
	}
	if idleSeconds > stuckThresholdSeconds {
		return "red"
	}
	if contextUsage >= 80 {
		return "yellow"
	}
	if idleSeconds > stuckThresholdSeconds/2 {
		return "yellow"
	}
	return "green"
}

func healthDot(health string) string {
	dot := "●"
	switch strings.ToLower(strings.TrimSpace(health)) {
	case "red":
		return errorStyle.Render(dot)
	case "yellow":
		return warnStyle.Render(dot)
	default:
		return successStyle.Render(dot)
	}
}

func statusIcon(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed":
		return errorStyle.Render("x")
	case "completed", "exited":
		return successStyle.Render("ok")
	default:
		return dimStyle.Render("~")
	}
}

func modelLabel(session Session) string {
	switch {
	case session.Provider != "" && session.Model != "":
		return session.Provider + "/" + session.Model
	case session.Model != "":
		return session.Model
	case session.Provider != "":
		return session.Provider
	default:
		return "-"
	}
}

func shortInt(v int) string {
	if v >= 1000 {
		return fmt.Sprintf("%.1fk", float64(v)/1000.0)
	}
	return fmt.Sprintf("%d", v)
}

func pickLoopState(current, candidate string) string {
	current = normalizeLoopState(current)
	candidate = normalizeLoopState(candidate)
	if candidate == "" {
		return current
	}
	if loopStateRank(candidate) > loopStateRank(current) {
		return candidate
	}
	return current
}

func normalizeLoopState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "running":
		return "running"
	case "paused":
		return "paused"
	case "draining":
		return "draining"
	default:
		return ""
	}
}

func loopStateRank(state string) int {
	switch state {
	case "draining":
		return 3
	case "paused":
		return 2
	case "running":
		return 1
	default:
		return 0
	}
}

func isActiveStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "spawning", "stuck":
		return true
	default:
		return false
	}
}

func ageLabel(now, at time.Time) string {
	if at.IsZero() {
		return "-"
	}
	if now.Before(at) {
		return "0s"
	}
	seconds := int64(now.Sub(at).Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	return fmt.Sprintf("%dh", hours)
}

func durationLabel(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func trimTo(value string, width int) string {
	value = strings.TrimSpace(value)
	if width <= 0 || len(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	return value[:width-1] + "..."
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return timestamp.UTC()
}

func nonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
