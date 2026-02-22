package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	if m.steerInput != "@cook-a " {
		t.Fatalf("steer input = %q, want %q", m.steerInput, "@cook-a ")
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
	m.surface = surfaceSteer
	m.overlay = surfaceDashboard
	m.steerInput = "@cook-a focus on tests first"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
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

	m = updated.(Model)
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
