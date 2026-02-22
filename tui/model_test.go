package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/poteto/noodle/loop"
)

func TestDeriveHealth(t *testing.T) {
	cases := []struct {
		name       string
		status     string
		explicit   string
		contextPct float64
		idle       int64
		threshold  int64
		want       string
	}{
		{name: "explicit wins", status: "running", explicit: "red", want: "red"},
		{name: "failed is red", status: "failed", want: "red"},
		{name: "stuck is red", status: "stuck", want: "red"},
		{name: "high context is yellow", status: "running", contextPct: 81, want: "yellow"},
		{name: "idle over half threshold is yellow", status: "running", idle: 70, threshold: 120, want: "yellow"},
		{name: "idle over threshold is red", status: "running", idle: 121, threshold: 120, want: "red"},
		{name: "healthy running is green", status: "running", idle: 10, threshold: 120, want: "green"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := deriveHealth(tc.status, tc.explicit, tc.contextPct, tc.idle, tc.threshold)
			if got != tc.want {
				t.Fatalf("deriveHealth() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestModelTransitionsBetweenSurfaces(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             func() time.Time { return time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC) },
	})
	m.snapshot = Snapshot{
		Sessions: []Session{
			{ID: "cook-a", Status: "running", Health: "green", LastActivity: time.Date(2026, 2, 22, 11, 59, 0, 0, time.UTC)},
		},
		Active: []Session{
			{ID: "cook-a", Status: "running", Health: "green", LastActivity: time.Date(2026, 2, 22, 11, 59, 0, 0, time.UTC)},
		},
		EventsBySession: map[string][]EventLine{
			"cook-a": {
				{Label: "Edit", Category: traceFilterTools},
				{Label: "Think", Category: traceFilterThink},
			},
		},
	}

	m = pressKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.surface != surfaceSession {
		t.Fatalf("surface after enter = %q, want %q", m.surface, surfaceSession)
	}
	if m.sessionID != "cook-a" {
		t.Fatalf("sessionID after enter = %q, want cook-a", m.sessionID)
	}

	m = pressRune(t, m, 't')
	if m.surface != surfaceTrace {
		t.Fatalf("surface after t = %q, want %q", m.surface, surfaceTrace)
	}

	m = pressRune(t, m, 'f')
	if m.traceFilter != traceFilterTools {
		t.Fatalf("trace filter after f = %q, want %q", m.traceFilter, traceFilterTools)
	}

	m = pressKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.surface != surfaceSession {
		t.Fatalf("surface after esc from trace = %q, want %q", m.surface, surfaceSession)
	}

	m = pressRune(t, m, 'q')
	if m.surface != surfaceQueue {
		t.Fatalf("surface after q = %q, want %q", m.surface, surfaceQueue)
	}

	m = pressKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.surface != surfaceDashboard {
		t.Fatalf("surface after esc = %q, want %q", m.surface, surfaceDashboard)
	}

	m.surface = surfaceSession
	m.sessionID = "cook-a"
	m = pressRune(t, m, 's')
	if m.surface != surfaceSteer {
		t.Fatalf("surface after s = %q, want %q", m.surface, surfaceSteer)
	}
	if m.steerInput != "" {
		t.Fatalf("expected empty steer input, got %q", m.steerInput)
	}

	m = pressKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.surface != surfaceSession {
		t.Fatalf("surface after esc from steer = %q, want %q", m.surface, surfaceSession)
	}

	m = pressRune(t, m, '?')
	if m.surface != surfaceHelp {
		t.Fatalf("surface after ? = %q, want %q", m.surface, surfaceHelp)
	}
	m = pressRune(t, m, '?')
	if m.surface != surfaceSession {
		t.Fatalf("surface after ? toggle = %q, want %q", m.surface, surfaceSession)
	}
}

func TestSteerSubmitWritesControlCommand(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	fixed := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)

	m := NewModel(Options{
		RuntimeDir:      runtimeDir,
		RefreshInterval: time.Hour,
		Now:             func() time.Time { return fixed },
	})
	m.snapshot.Active = []Session{{ID: "cook-a"}}
	m.surface = surfaceSteer
	m.overlay = surfaceDashboard
	m.steerInput = "@cook-a focus on tests first"

	cmd, ok := m.submitSteer()
	if !ok {
		t.Fatal("expected steer submit to succeed")
	}
	if cmd == nil {
		t.Fatal("expected steer command")
	}
	msg := cmd()
	result, ok := msg.(controlResultMsg)
	if !ok {
		t.Fatalf("command msg type = %T, want controlResultMsg", msg)
	}
	if result.err != nil {
		t.Fatalf("control command failed: %v", result.err)
	}

	if m.surface != surfaceDashboard {
		t.Fatalf("surface after submit = %q, want %q", m.surface, surfaceDashboard)
	}

	data, err := os.ReadFile(filepath.Join(runtimeDir, "control.ndjson"))
	if err != nil {
		t.Fatalf("read control.ndjson: %v", err)
	}
	var wrote loop.ControlCommand
	if err := json.Unmarshal(data[:len(data)-1], &wrote); err != nil {
		t.Fatalf("parse control command: %v", err)
	}
	if wrote.Action != "steer" {
		t.Fatalf("action = %q, want steer", wrote.Action)
	}
	if wrote.Target != "cook-a" {
		t.Fatalf("target = %q, want cook-a", wrote.Target)
	}
	if wrote.Prompt != "focus on tests first" {
		t.Fatalf("prompt = %q", wrote.Prompt)
	}
}

func TestParseSteerInputEveryoneExpandsTargets(t *testing.T) {
	targets, prompt, err := parseSteerInput(
		"@everyone focus on tests first",
		[]string{"sous-chef", "cook-a", "cook-b"},
	)
	if err != nil {
		t.Fatalf("parseSteerInput returned error: %v", err)
	}
	if len(targets) != 3 {
		t.Fatalf("targets len = %d, want 3", len(targets))
	}
	if prompt != "focus on tests first" {
		t.Fatalf("prompt = %q", prompt)
	}
}

func TestSteerEscClosesMentionsBeforeModal(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})
	m.surface = surfaceSteer
	m.overlay = surfaceDashboard
	m.steerInput = "@"
	m.snapshot.Active = []Session{{ID: "cook-a"}}
	m.refreshSteerMentions()
	if !m.steerMentionOpen {
		t.Fatal("expected mention dropdown to be open")
	}

	m = pressKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.surface != surfaceSteer {
		t.Fatalf("surface after first esc = %q, want %q", m.surface, surfaceSteer)
	}
	if m.steerMentionOpen {
		t.Fatal("expected mention dropdown to close on first esc")
	}

	m = pressKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.surface != surfaceDashboard {
		t.Fatalf("surface after second esc = %q, want %q", m.surface, surfaceDashboard)
	}
}

func TestSteerMentionSelectionInsertsTarget(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})
	m.surface = surfaceSteer
	m.snapshot.Active = []Session{{ID: "cook-a"}}

	m = pressRune(t, m, '@')
	if !m.steerMentionOpen {
		t.Fatal("expected mention dropdown after typing @")
	}
	if len(m.steerMentionItems) == 0 {
		t.Fatal("expected mention candidates")
	}

	m = pressKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = pressKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.steerMentionOpen {
		t.Fatal("expected mention dropdown to close after selection")
	}
	if m.steerInput == "@" {
		t.Fatalf("expected selected mention in input, got %q", m.steerInput)
	}
	if m.steerInput == "" {
		t.Fatal("expected steer input to contain selected mention")
	}
}

func TestRenderSessionWrapsLongEventMessages(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             func() time.Time { return now },
	})
	m.width = 80
	m.surface = surfaceSession
	m.sessionID = "cook-a"
	m.snapshot = Snapshot{
		Sessions: []Session{
			{
				ID:              "cook-a",
				Status:          "running",
				Health:          "green",
				Provider:        "codex",
				Model:           "gpt-5.3-codex",
				DurationSeconds: 29,
			},
		},
		EventsBySession: map[string][]EventLine{
			"cook-a": {
				{
					At:    now,
					Label: "Think",
					Body:  "this is a deliberately long event message that should wrap across multiple lines instead of being cut off with ellipsis in the session detail view",
				},
			},
		},
	}

	view := m.renderSession()
	if strings.Contains(view, "...") {
		t.Fatalf("expected wrapped event body without truncation ellipsis, got:\n%s", view)
	}
	if !strings.Contains(view, "this is a deliberately long event message") {
		t.Fatalf("expected first wrapped segment in view, got:\n%s", view)
	}
	if !strings.Contains(view, "instead of being cut off with ellipsis") {
		t.Fatalf("expected continuation segment in view, got:\n%s", view)
	}
}

func TestWrapPlainTextSplitsVeryLongTokens(t *testing.T) {
	segments := wrapPlainText("supercalifragilisticexpialidocious", 8)
	if len(segments) < 2 {
		t.Fatalf("expected long token to split across segments, got %#v", segments)
	}
	for _, segment := range segments {
		if len([]rune(segment)) > 8 {
			t.Fatalf("segment exceeds width: %q", segment)
		}
	}
}

func TestRenderDashboardShowsUpNextTitle(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})
	m.width = 120
	m.snapshot = Snapshot{
		Queue: []QueueItem{
			{
				ID:       "22",
				Title:    "Make planning opinionated and first-class",
				Provider: "codex",
				Model:    "gpt-5.3-codex",
			},
		},
	}

	view := m.renderDashboard()
	if !strings.Contains(view, "Make planning opinionated and first-class") {
		t.Fatalf("expected queue title in dashboard up next view, got:\n%s", view)
	}
}

func TestRenderQueueShowsTitle(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})
	m.width = 120
	m.surface = surfaceQueue
	m.snapshot = Snapshot{
		Queue: []QueueItem{
			{
				ID:       "14",
				Title:    "Implement fixture hash validation",
				Provider: "claude",
				Model:    "claude-opus-4-6",
			},
		},
	}

	view := m.renderQueue()
	if !strings.Contains(view, "Implement fixture hash validation") {
		t.Fatalf("expected queue title in queue view, got:\n%s", view)
	}
}

func pressRune(t *testing.T, m Model, r rune) Model {
	t.Helper()
	return pressKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
}

func pressKey(t *testing.T, m Model, key tea.KeyMsg) Model {
	t.Helper()
	updated, _ := m.Update(key)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("update model type = %T, want tui.Model", updated)
	}
	return next
}
