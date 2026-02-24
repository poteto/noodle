package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/poteto/noodle/loop"
)

const (
	defaultRefreshInterval = 2 * time.Second
	defaultStuckThreshold  = int64(120)
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

	activeTab        Tab
	steerOpen        bool
	detailSession    string // session ID shown in actor detail view
	detailScroll     int    // line offset for actor detail scroll
	detailAutoScroll bool   // true when auto-scrolling to newest events
	detailTotalLines int    // total rendered lines (set during render)

	snapshot  Snapshot
	feedTab   FeedTab
	queueTab  QueueTab
	configTab ConfigTab
	err       error

	steerInput        string
	steerMentionOpen  bool
	steerMentionItems []string
	steerMentionIndex int
	steerMentionStart int
	statusLine        string

	taskEditor   TaskEditor
	quitPending  bool
	quitDeadline time.Time
	shimmerIndex int
}

type Snapshot struct {
	UpdatedAt time.Time
	LoopState string

	Sessions []Session
	Active   []Session
	Recent   []Session
	Queue    []QueueItem

	ActiveQueueIDs  []string
	ActionNeeded    []string
	EventsBySession map[string][]EventLine
	FeedEvents   []FeedEvent
	TotalCostUSD float64

	Verdicts           []Verdict
	PendingReviews     []loop.PendingReviewItem
	PendingReviewCount int
	Autonomy           string
}

type Session struct {
	ID                    string
	DisplayName           string
	Status                string
	Runtime               string
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
	ID        string   `json:"id"`
	TaskKey   string   `json:"task_key,omitempty"`
	Title     string   `json:"title,omitempty"`
	Prompt    string   `json:"prompt,omitempty"`
	Provider  string   `json:"provider"`
	Model     string   `json:"model"`
	Skill     string   `json:"skill,omitempty"`
	Plan      []string `json:"plan,omitempty"`
	Rationale string   `json:"rationale,omitempty"`
}

type EventLine struct {
	At       time.Time
	Label    string
	Body     string
	Category TraceFilter
}

type tickMsg time.Time
type quitResetMsg struct{}
type shimmerMsg struct{}

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
		activeTab:       TabFeed,
		queueTab:        NewQueueTab(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		refreshSnapshotCmd(m.runtimeDir, m.now),
		tickCmd(m.refreshInterval),
		shimmerCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.detailSession != "" {
			m.detailTotalLines = m.countDetailLines()
		}
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
		m.feedTab.SetSnapshot(m.snapshot)
		m.queueTab.SetQueue(m.snapshot.Queue, m.snapshot.ActiveQueueIDs, m.snapshot.ActionNeeded, m.snapshot.LoopState)
		m.configTab.SetAutonomy(m.snapshot.Autonomy)
		if m.detailSession != "" {
			m.detailTotalLines = m.countDetailLines()
		}
		return m, nil
	case controlResultMsg:
		if msg.err != nil {
			m.statusLine = fmt.Sprintf("%s failed: %v", msg.action, msg.err)
			return m, nil
		}
		m.statusLine = fmt.Sprintf("%s command sent", msg.action)
		return m, nil
	case shimmerMsg:
		if len(m.snapshot.Active) > 0 {
			m.shimmerIndex = (m.shimmerIndex + 1) % 14 // len("noodle cooking")
		}
		return m, shimmerCmd()
	case quitResetMsg:
		m.quitPending = false
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	default:
		return m, nil
	}
}

func (m Model) View() tea.View {
	view := tea.NewView(m.renderLayout())
	view.AltScreen = true
	return view
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if isCtrlC(msg) {
		if m.quitPending && m.now().Before(m.quitDeadline) {
			return m, tea.Quit
		}
		m.quitPending = true
		m.quitDeadline = m.now().Add(2 * time.Second)
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return quitResetMsg{}
		})
	}

	if msg.Code == tea.KeyEsc {
		if m.taskEditor.open {
			m.taskEditor.Close()
			return m, nil
		}
		if m.steerOpen {
			if m.steerMentionOpen {
				m.closeSteerMentions()
				return m, nil
			}
			m.closeSteer()
			return m, nil
		}
		if m.detailSession != "" {
			m.detailSession = ""
			return m, nil
		}
	}

	if m.steerOpen {
		return m.handleSteerKey(msg)
	}
	if m.taskEditor.open {
		return m.handleTaskEditorKey(msg)
	}
	if m.detailSession != "" {
		key := strings.ToLower(msg.String())
		switch key {
		case "1", "2", "3":
			m.detailSession = ""
			// Fall through to tab switching below.
		case "j", "down":
			m.detailScroll++
			if m.detailTotalLines > 0 {
				maxScroll := m.detailTotalLines - m.detailVisibleHeight()
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.detailScroll >= maxScroll {
					m.detailScroll = maxScroll
					m.detailAutoScroll = true
				}
			}
			return m, nil
		case "k", "up":
			m.detailScroll--
			if m.detailScroll < 0 {
				m.detailScroll = 0
			}
			m.detailAutoScroll = false
			return m, nil
		case "h", "left":
			m.navigateActor(-1)
			return m, nil
		case "l", "right":
			m.navigateActor(1)
			return m, nil
		default:
			return m, nil
		}
	}

	key := strings.ToLower(msg.String())
	switch key {
	case "1":
		m.activeTab = TabFeed
	case "2":
		m.activeTab = TabQueue
	case "3":
		m.activeTab = TabConfig
	case "?":
		// reserved
	case "`":
		return m, m.openSteer()
	case "n":
		m.taskEditor.OpenNew()
		return m, nil
	case "e":
		if m.activeTab == TabQueue {
			if item, ok := m.queueTab.SelectedItem(); ok {
				m.taskEditor.OpenEdit(item)
			}
		}
		return m, nil
	case "j", "down":
		switch m.activeTab {
		case TabFeed:
			m.feedTab.SelectDown()
		case TabQueue:
			m.queueTab.table.MoveDown(1)
		}
	case "k", "up":
		switch m.activeTab {
		case TabFeed:
			m.feedTab.SelectUp()
		case TabQueue:
			m.queueTab.table.MoveUp(1)
		}
	case "enter":
		switch m.activeTab {
		case TabFeed:
			if sid := m.feedTab.SelectedSessionID(); sid != "" {
				m.openDetail(sid)
			}
		case TabQueue:
			if queueID := m.queueTab.SelectedSessionID(); queueID != "" {
				if sid := m.findSessionForQueueItem(queueID); sid != "" {
					m.openDetail(sid)
				}
			}
		}
	case "m":
		if m.activeTab == TabFeed && len(m.snapshot.Verdicts) > 0 {
			return m, m.mergeSelectedVerdict()
		}
	case "x":
		if m.activeTab == TabFeed && len(m.snapshot.Verdicts) > 0 {
			return m, m.rejectSelectedVerdict()
		}
	case "a":
		if m.activeTab == TabFeed {
			return m, m.mergeAllApproved()
		}
	case "left", "h":
		if m.activeTab == TabConfig {
			mode := m.configTab.CycleLeft()
			return m, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{Action: "autonomy", Value: mode})
		}
		if m.activeTab != TabFeed {
			m.navigateActor(-1)
		}
	case "right", "l":
		if m.activeTab == TabConfig {
			mode := m.configTab.CycleRight()
			return m, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{Action: "autonomy", Value: mode})
		}
		if m.activeTab != TabFeed {
			m.navigateActor(1)
		}
	case "p":
		action := "pause"
		if strings.EqualFold(m.snapshot.LoopState, "paused") {
			action = "resume"
		}
		return m, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{Action: action})
	}

	return m, nil
}

func (m Model) handleSteerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEnter:
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
	case msg.Code == tea.KeyUp:
		if m.steerMentionOpen && len(m.steerMentionItems) > 0 && m.steerMentionIndex > 0 {
			m.steerMentionIndex--
		}
		return m, nil
	case msg.Code == tea.KeyDown:
		if m.steerMentionOpen && m.steerMentionIndex < len(m.steerMentionItems)-1 {
			m.steerMentionIndex++
		}
		return m, nil
	case msg.Code == tea.KeyBackspace || isCtrlH(msg):
		m.steerInput = dropLastRune(m.steerInput)
		m.refreshSteerMentions()
		return m, nil
	case msg.Code == tea.KeySpace:
		m.steerInput += " "
		m.refreshSteerMentions()
		return m, nil
	case msg.Text != "":
		m.steerInput += msg.Text
		m.refreshSteerMentions()
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) handleTaskEditorKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.Code == tea.KeyEnter {
		cmd := m.taskEditor.Submit(m.runtimeDir, m.now)
		if cmd != nil {
			return m, cmd
		}
		return m, nil
	}
	cmd, handled := m.taskEditor.HandleKey(msg)
	if handled {
		return m, cmd
	}
	return m, nil
}

func isCtrlC(msg tea.KeyPressMsg) bool {
	key := msg.Key()
	return key.Code == 'c' && (key.Mod&tea.ModCtrl) != 0
}

func isCtrlH(msg tea.KeyPressMsg) bool {
	key := msg.Key()
	return key.Code == 'h' && (key.Mod&tea.ModCtrl) != 0
}

// paneWidth returns the content pane width, accounting for whether the rail
// is visible on the current tab.
func (m Model) paneWidth() int {
	if m.activeTab == TabFeed {
		w := m.width
		if w < 20 {
			w = 20
		}
		return w
	}
	compact := m.width < 80
	effectiveRailWidth := railWidth
	if compact {
		effectiveRailWidth = 8
	}
	w := m.width - effectiveRailWidth - 1
	if w < 20 {
		w = 20
	}
	return w
}

func (m *Model) openSteer() tea.Cmd {
	m.steerOpen = true
	m.steerInput = ""
	m.steerMentionOpen = false
	m.steerMentionItems = nil
	m.steerMentionIndex = 0
	m.steerMentionStart = -1
	return nil
}

func (m *Model) closeSteer() {
	m.steerOpen = false
	m.steerInput = ""
	m.closeSteerMentions()
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
	scheduler := loop.PrioritizeTaskKey()
	targets := []string{scheduler}
	seen := map[string]struct{}{scheduler: {}}
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

func (m *Model) openDetail(sessionID string) {
	m.detailSession = sessionID
	m.detailScroll = 0
	m.detailAutoScroll = true
	m.detailTotalLines = 0
}

func (m Model) detailVisibleHeight() int {
	// Matches contentHeight calculation in renderLayout minus header (3 lines).
	bottomReserve := 2
	layoutHeight := m.height - bottomReserve
	if layoutHeight < 6 {
		layoutHeight = 6
	}
	contentHeight := layoutHeight - 4
	if contentHeight < 4 {
		contentHeight = 4
	}
	return contentHeight - 3
}

// countDetailLines estimates the total rendered lines for the current detail
// session events, accounting for word-wrap at the current width.
func (m Model) countDetailLines() int {
	events := m.snapshot.EventsBySession[m.detailSession]
	if len(events) == 0 {
		return 0
	}
	const prefixWidth = 8 + 2 + 10 + 2 // ts + gap + label + gap
	msgWidth := m.paneWidth() - prefixWidth
	if msgWidth < 20 {
		msgWidth = 20
	}
	total := 0
	for _, ev := range events {
		total += len(wrapText(ev.Body, msgWidth))
	}
	return total
}

// navigateActor cycles through active sessions by direction (-1 prev, +1 next).
// When not in detail view, opens the first (or last) active session.
func (m *Model) navigateActor(direction int) {
	sessions := m.snapshot.Active
	if len(sessions) == 0 {
		return
	}
	if m.detailSession == "" {
		if direction > 0 {
			m.openDetail(sessions[0].ID)
		} else {
			m.openDetail(sessions[len(sessions)-1].ID)
		}
		return
	}
	current := -1
	for i, s := range sessions {
		if s.ID == m.detailSession {
			current = i
			break
		}
	}
	if current == -1 {
		m.openDetail(sessions[0].ID)
		return
	}
	next := current + direction
	if next < 0 {
		next = len(sessions) - 1
	} else if next >= len(sessions) {
		next = 0
	}
	m.openDetail(sessions[next].ID)
}

func (m *Model) findSessionForQueueItem(queueItemID string) string {
	// Session IDs are prefixed with the queue item ID (e.g., "execute-1-fix-bug-...").
	for _, s := range m.snapshot.Sessions {
		if strings.HasPrefix(s.ID, queueItemID+"-") || s.ID == queueItemID {
			return s.ID
		}
	}
	return ""
}

func (m *Model) isActionable(targetID string) bool {
	for _, id := range m.snapshot.ActionNeeded {
		if id == targetID {
			return true
		}
	}
	return false
}

func (m *Model) mergeSelectedVerdict() tea.Cmd {
	for _, v := range m.snapshot.Verdicts {
		if m.isActionable(v.TargetID) {
			return sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{
				Action: "merge",
				Item:   v.TargetID,
			})
		}
	}
	return nil
}

func (m *Model) rejectSelectedVerdict() tea.Cmd {
	for _, v := range m.snapshot.Verdicts {
		if m.isActionable(v.TargetID) {
			return sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{
				Action: "reject",
				Item:   v.TargetID,
			})
		}
	}
	return nil
}

func (m *Model) mergeAllApproved() tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	for _, v := range m.snapshot.Verdicts {
		if v.Accept && m.isActionable(v.TargetID) {
			cmds = append(cmds, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{
				Action: "merge",
				Item:   v.TargetID,
			}))
		}
	}
	if len(cmds) == 0 {
		m.statusLine = "no approved verdicts to merge"
		return nil
	}
	return tea.Batch(cmds...)
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

func shimmerCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return shimmerMsg{}
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
