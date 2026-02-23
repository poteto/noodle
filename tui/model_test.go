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

func TestTabSwitching(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})
	if m.activeTab != TabFeed {
		t.Fatalf("default tab = %v, want TabFeed", m.activeTab)
	}

	m = pressRune(t, m, '2')
	if m.activeTab != TabQueue {
		t.Fatalf("tab after 2 = %v, want TabQueue", m.activeTab)
	}

	m = pressRune(t, m, '3')
	if m.activeTab != TabBrain {
		t.Fatalf("tab after 3 = %v, want TabBrain", m.activeTab)
	}

	m = pressRune(t, m, '4')
	if m.activeTab != TabConfig {
		t.Fatalf("tab after 4 = %v, want TabConfig", m.activeTab)
	}

	m = pressRune(t, m, '1')
	if m.activeTab != TabFeed {
		t.Fatalf("tab after 1 = %v, want TabFeed", m.activeTab)
	}
}

func TestHelpToggle(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})

	m = pressRune(t, m, '?')
	if !m.helpOpen {
		t.Fatal("expected helpOpen after ?")
	}
	m = pressRune(t, m, '?')
	if m.helpOpen {
		t.Fatal("expected helpOpen=false after ? toggle")
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
	m.steerOpen = true
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

	if m.steerOpen {
		t.Fatal("expected steerOpen=false after submit")
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
		[]string{"prioritize", "cook-a", "cook-b"},
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
	m.steerOpen = true
	m.steerInput = "@"
	m.snapshot.Active = []Session{{ID: "cook-a"}}
	m.refreshSteerMentions()
	if !m.steerMentionOpen {
		t.Fatal("expected mention dropdown to be open")
	}

	m = pressKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if !m.steerOpen {
		t.Fatal("expected steer still open after first esc (only closes mentions)")
	}
	if m.steerMentionOpen {
		t.Fatal("expected mention dropdown to close on first esc")
	}

	m = pressKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.steerOpen {
		t.Fatal("expected steerOpen=false after second esc")
	}
}

func TestSteerMentionSelectionInsertsTarget(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})
	m.steerOpen = true
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

func TestLayoutRendersAtVariousWidths(t *testing.T) {
	now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             func() time.Time { return now },
	})
	m.snapshot = Snapshot{
		Active: []Session{
			{ID: "cook-a", Status: "running", Health: "green", Model: "claude-opus-4-6", DurationSeconds: 120},
		},
		Queue: []QueueItem{{ID: "1", Title: "Test task"}},
	}

	for _, width := range []int{80, 120, 200} {
		m.width = width
		m.height = 24
		view := m.View()
		if view == "" {
			t.Fatalf("empty view at width %d", width)
		}
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

func TestVerdictCardShowsActionsInReviewMode(t *testing.T) {
	now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	v := Verdict{
		SessionID: "cook-a",
		Accept:    true,
		Feedback:  "looks good",
		Timestamp: now.Add(-5 * time.Minute),
	}

	// Review mode: should show action pills.
	card := renderVerdictCard(v, 80, now, true)
	if card == "" {
		t.Fatal("expected non-empty verdict card")
	}
	if !containsStr(card, "Merge") {
		t.Fatal("expected Merge pill in review mode")
	}

	// Full mode: no action pills.
	card = renderVerdictCard(v, 80, now, false)
	if containsStr(card, "Merge") {
		t.Fatal("expected no Merge pill in full mode")
	}
}

func TestMergeWritesControlCommand(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	fixed := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)

	m := NewModel(Options{
		RuntimeDir:      runtimeDir,
		RefreshInterval: time.Hour,
		Now:             func() time.Time { return fixed },
	})
	m.snapshot.Verdicts = []Verdict{
		{SessionID: "cook-a", Accept: true},
	}
	m.activeTab = TabFeed

	cmd := m.mergeSelectedVerdict()
	if cmd == nil {
		t.Fatal("expected merge command")
	}
	msg := cmd()
	result, ok := msg.(controlResultMsg)
	if !ok {
		t.Fatalf("command msg type = %T, want controlResultMsg", msg)
	}
	if result.err != nil {
		t.Fatalf("merge command failed: %v", result.err)
	}

	data, err := os.ReadFile(filepath.Join(runtimeDir, "control.ndjson"))
	if err != nil {
		t.Fatalf("read control.ndjson: %v", err)
	}
	var wrote loop.ControlCommand
	if err := json.Unmarshal(data[:len(data)-1], &wrote); err != nil {
		t.Fatalf("parse control command: %v", err)
	}
	if wrote.Action != "merge" {
		t.Fatalf("action = %q, want merge", wrote.Action)
	}
	if wrote.Target != "cook-a" {
		t.Fatalf("target = %q, want cook-a", wrote.Target)
	}
}

func TestPendingCountMatchesApprovedVerdicts(t *testing.T) {
	verdicts := []Verdict{
		{SessionID: "cook-a", Accept: true},
		{SessionID: "cook-b", Accept: false},
		{SessionID: "cook-c", Accept: true},
	}
	count := 0
	for _, v := range verdicts {
		if v.Accept {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("pending count = %d, want 2", count)
	}
}

func TestMergeAllApprovedSkipsRejected(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	fixed := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)

	m := NewModel(Options{
		RuntimeDir:      runtimeDir,
		RefreshInterval: time.Hour,
		Now:             func() time.Time { return fixed },
	})
	m.snapshot.Verdicts = []Verdict{
		{SessionID: "cook-a", Accept: true},
		{SessionID: "cook-b", Accept: false},
		{SessionID: "cook-c", Accept: true},
	}

	cmd := m.mergeAllApproved()
	if cmd == nil {
		t.Fatal("expected batch command")
	}

	// tea.Batch returns a BatchMsg ([]tea.Cmd) — execute each inner command.
	batchMsg := cmd()
	innerCmds, ok := batchMsg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("batch result type = %T, want tea.BatchMsg", batchMsg)
	}
	for _, inner := range innerCmds {
		if inner != nil {
			inner()
		}
	}

	data, err := os.ReadFile(filepath.Join(runtimeDir, "control.ndjson"))
	if err != nil {
		t.Fatalf("read control.ndjson: %v", err)
	}
	lines := splitNDJSON(string(data))
	if len(lines) != 2 {
		t.Fatalf("control commands = %d, want 2 (skipping rejected)", len(lines))
	}

	for _, line := range lines {
		var cmd loop.ControlCommand
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			t.Fatalf("parse control command: %v", err)
		}
		if cmd.Action != "merge" {
			t.Fatalf("action = %q, want merge", cmd.Action)
		}
		if cmd.Target == "cook-b" {
			t.Fatal("expected cook-b (rejected) to be skipped")
		}
	}
}

func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitNDJSON(data string) []string {
	var lines []string
	for _, line := range splitLines(data) {
		line = trimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r' || s[start] == '\n') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
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
