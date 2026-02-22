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

	snapshot Snapshot
	err      error

	selectedActive int
	selectedQueue  int
	sessionID      string

	traceFilter TraceFilter
	traceFollow bool
	traceOffset int

	steering   bool
	steerInput string
	showHelp   bool
	statusLine string
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
	b.WriteString(m.renderMain())
	if m.showHelp {
		b.WriteString("\n\n")
		b.WriteString(renderHelp())
	}
	if m.steering {
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
		b.WriteString("error: ")
		b.WriteString(m.err.Error())
	}
	return b.String()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		if m.steering {
			m.steering = false
			m.steerInput = ""
			return m, nil
		}
		if m.showHelp {
			m.showHelp = false
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

	if m.steering {
		return m.handleSteerKey(msg)
	}

	key := strings.ToLower(msg.String())
	switch key {
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "s":
		m.steering = true
		m.steerInput = m.defaultSteerInput()
		return m, nil
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
	case tea.KeyEnter:
		target, prompt, ok := parseSteerInput(m.steerInput)
		if !ok {
			m.statusLine = "steer format: @target instruction"
			return m, nil
		}
		m.steering = false
		m.steerInput = ""
		return m, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{
			Action: "steer",
			Target: target,
			Prompt: prompt,
		})
	case tea.KeyBackspace:
		if len(m.steerInput) > 0 {
			m.steerInput = m.steerInput[:len(m.steerInput)-1]
		}
		return m, nil
	case tea.KeySpace:
		m.steerInput += " "
		return m, nil
	case tea.KeyRunes:
		m.steerInput += string(msg.Runes)
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) renderMain() string {
	switch m.surface {
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
	var b strings.Builder
	fmt.Fprintf(
		&b,
		"noodle | %s | active %d | queue %d | total $%.2f",
		nonEmpty(m.snapshot.LoopState, "running"),
		len(m.snapshot.Active),
		len(m.snapshot.Queue),
		m.snapshot.TotalCostUSD,
	)
	if !m.snapshot.UpdatedAt.IsZero() {
		fmt.Fprintf(&b, " | updated %s ago", ageLabel(m.now(), m.snapshot.UpdatedAt))
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", 72))
	b.WriteString("\n")
	b.WriteString("Active Cooks\n")
	if len(m.snapshot.Active) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for i, session := range m.snapshot.Active {
			cursor := " "
			if i == m.selectedActive {
				cursor = ">"
			}
			action := nonEmpty(session.CurrentAction, "(idle)")
			fmt.Fprintf(
				&b,
				"%s %s %-18s %-14s %-22s %6s ago\n",
				cursor,
				healthDot(session.Health),
				session.ID,
				modelLabel(session),
				trimTo(action, 22),
				ageLabel(m.now(), session.LastActivity),
			)
		}
	}

	b.WriteString("\nRecent\n")
	if len(m.snapshot.Recent) == 0 {
		b.WriteString("  (none)\n")
	} else {
		limit := 6
		if len(m.snapshot.Recent) < limit {
			limit = len(m.snapshot.Recent)
		}
		for i := 0; i < limit; i++ {
			s := m.snapshot.Recent[i]
			fmt.Fprintf(
				&b,
				"  %s %-18s %-14s %7s $%.2f\n",
				statusIcon(s.Status),
				s.ID,
				modelLabel(s),
				durationLabel(s.DurationSeconds),
				s.TotalCostUSD,
			)
		}
	}

	b.WriteString("\nUp Next\n")
	if len(m.snapshot.Queue) == 0 {
		b.WriteString("  (empty)\n")
	} else {
		limit := 6
		if len(m.snapshot.Queue) < limit {
			limit = len(m.snapshot.Queue)
		}
		for i := 0; i < limit; i++ {
			item := m.snapshot.Queue[i]
			fmt.Fprintf(
				&b,
				"  %d. %-22s %-14s %-14s\n",
				i+1,
				item.ID,
				nonEmpty(item.Provider, "claude"),
				nonEmpty(item.Model, "(default)"),
			)
		}
	}

	b.WriteString("\nenter inspect | q queue | s steer | p pause/resume | d drain | ? help | ctrl+c quit")
	return b.String()
}

func (m Model) renderSession() string {
	session, ok := m.sessionByID(m.sessionID)
	if !ok {
		return "session not found\n\nesc back"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Session Detail | %s\n", session.ID)
	b.WriteString(strings.Repeat("-", 72))
	b.WriteString("\n")

	fmt.Fprintf(&b, "Status: %s %s\n", session.Status, healthDot(session.Health))
	fmt.Fprintf(&b, "Provider: %s\n", nonEmpty(session.Provider, "-"))
	fmt.Fprintf(&b, "Model: %s\n", nonEmpty(session.Model, "-"))
	fmt.Fprintf(&b, "Duration: %s\n", durationLabel(session.DurationSeconds))
	fmt.Fprintf(&b, "Cost: $%.2f\n", session.TotalCostUSD)
	fmt.Fprintf(&b, "Retries: %d\n", session.RetryCount)
	fmt.Fprintf(&b, "Worktree: .worktrees/%s\n", session.ID)

	lines := m.snapshot.EventsBySession[session.ID]
	b.WriteString("\nRecent Events\n")
	if len(lines) == 0 {
		b.WriteString("  (none)\n")
	} else {
		start := len(lines) - 8
		if start < 0 {
			start = 0
		}
		for _, line := range lines[start:] {
			fmt.Fprintf(
				&b,
				"  %s  %-6s | %s\n",
				line.At.Format("15:04:05"),
				line.Label,
				trimTo(line.Body, 70),
			)
		}
	}

	b.WriteString("\nt trace | k kill | s steer | esc back | ? help")
	return b.String()
}

func (m Model) renderTrace() string {
	session, ok := m.sessionByID(m.sessionID)
	if !ok {
		return "trace unavailable: session not found\n\nesc back"
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
	fmt.Fprintf(&b, "Trace | %s | filter: %s\n", session.ID, m.traceFilter)
	b.WriteString(strings.Repeat("-", 72))
	b.WriteString("\n")
	if len(lines) == 0 {
		b.WriteString("(no events)\n")
	} else {
		for _, line := range lines[start:end] {
			fmt.Fprintf(&b, "%s  %-6s | %s\n", line.At.Format("15:04:05"), line.Label, line.Body)
		}
	}
	if m.traceFollow {
		b.WriteString("\n[auto-scroll]")
	}
	b.WriteString("\nf filter | G bottom | esc back | ? help")
	return b.String()
}

func (m Model) renderQueue() string {
	var b strings.Builder
	b.WriteString("Queue\n")
	b.WriteString(strings.Repeat("-", 72))
	b.WriteString("\n")
	if len(m.snapshot.Queue) == 0 {
		b.WriteString("(empty)\n")
	} else {
		for i, item := range m.snapshot.Queue {
			cursor := " "
			if i == m.selectedQueue {
				cursor = ">"
			}
			review := "default"
			if item.Review != nil {
				if *item.Review {
					review = "review"
				} else {
					review = "no-review"
				}
			}
			fmt.Fprintf(
				&b,
				"%s %2d. %-24s %-10s %-16s %s\n",
				cursor,
				i+1,
				item.ID,
				nonEmpty(item.Provider, "-"),
				nonEmpty(item.Model, "-"),
				review,
			)
		}
	}
	b.WriteString("\nesc back | s steer | ? help")
	return b.String()
}

func renderHelp() string {
	return strings.Join([]string{
		"Keys",
		"----",
		"Global: s steer | p pause/resume | d drain | ? help | ctrl+c quit",
		"Dashboard: enter inspect | q queue | up/down move",
		"Session: t trace | k kill | esc back",
		"Trace: f filter | G bottom | up/down scroll | esc back",
		"Steer: @target instructions, enter send, esc cancel",
	}, "\n")
}

func (m Model) renderSteer() string {
	return "steer> " + m.steerInput + "\nenter send | esc cancel"
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

func (m Model) defaultSteerInput() string {
	if m.surface == surfaceSession && m.sessionID != "" {
		return "@" + m.sessionID + " "
	}
	return "@"
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

func parseSteerInput(input string) (target string, prompt string, ok bool) {
	value := strings.TrimSpace(input)
	if !strings.HasPrefix(value, "@") {
		return "", "", false
	}
	parts := strings.Fields(value)
	if len(parts) < 2 {
		return "", "", false
	}
	target = strings.TrimPrefix(parts[0], "@")
	target = strings.TrimSpace(target)
	if target == "" {
		return "", "", false
	}
	prompt = strings.TrimSpace(strings.TrimPrefix(value, "@"+target))
	if prompt == "" {
		return "", "", false
	}
	return target, prompt, true
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
	switch strings.ToLower(strings.TrimSpace(health)) {
	case "red":
		return "\x1b[31m●\x1b[0m"
	case "yellow":
		return "\x1b[33m●\x1b[0m"
	default:
		return "\x1b[32m●\x1b[0m"
	}
}

func statusIcon(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed":
		return "x"
	case "completed", "exited":
		return "ok"
	default:
		return "~"
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
