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

	activeTab Tab
	steerOpen bool
	helpOpen  bool

	snapshot Snapshot
	err      error

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
	ID        string `json:"id"`
	TaskKey   string `json:"task_key,omitempty"`
	Title     string `json:"title,omitempty"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Skill     string `json:"skill,omitempty"`
	Review    *bool  `json:"review,omitempty"`
	Rationale string `json:"rationale,omitempty"`
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
		activeTab:       TabFeed,
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
	return m.renderLayout()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		if m.steerOpen {
			if m.steerMentionOpen {
				m.closeSteerMentions()
				return m, nil
			}
			m.closeSteer()
			return m, nil
		}
		if m.helpOpen {
			m.helpOpen = false
			return m, nil
		}
		return m, nil
	}

	if m.steerOpen {
		return m.handleSteerKey(msg)
	}
	if m.helpOpen {
		if strings.ToLower(msg.String()) == "?" {
			m.helpOpen = false
		}
		return m, nil
	}

	key := strings.ToLower(msg.String())
	switch key {
	case "1":
		m.activeTab = TabFeed
	case "2":
		m.activeTab = TabQueue
	case "3":
		m.activeTab = TabBrain
	case "4":
		m.activeTab = TabConfig
	case "?":
		m.helpOpen = true
	case "s":
		return m, m.openSteer()
	case "p":
		action := "pause"
		if strings.EqualFold(m.snapshot.LoopState, "paused") {
			action = "resume"
		}
		return m, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{Action: action})
	}

	return m, nil
}

func (m Model) handleSteerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
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
