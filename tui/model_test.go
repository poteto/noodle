package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
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
	if m.activeTab != TabReviews {
		t.Fatalf("tab after 3 = %v, want TabReviews", m.activeTab)
	}

	m = pressRune(t, m, '1')
	if m.activeTab != TabFeed {
		t.Fatalf("tab after 1 = %v, want TabFeed", m.activeTab)
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

	m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if !m.steerOpen {
		t.Fatal("expected steer still open after first esc (only closes mentions)")
	}
	if m.steerMentionOpen {
		t.Fatal("expected mention dropdown to close on first esc")
	}

	m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
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

	m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
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
		view := m.View().Content
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

func TestRenderEventBodyHighlightsCodeBlocks(t *testing.T) {
	body := "some prose\n```go\nfunc main() {}\n```\nmore prose"
	lines := renderEventBody(body, 80)
	// Should have at least: prose line, highlighted code line, more prose line.
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d: %v", len(lines), lines)
	}
	// The highlighted code line should contain ANSI escape codes.
	found := false
	for _, line := range lines {
		if strings.Contains(line, "\x1b[") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ANSI codes in highlighted output, got: %v", lines)
	}
}

func TestRenderEventBodyNoCodeBlock(t *testing.T) {
	body := "just plain text with no code"
	lines := renderEventBody(body, 40)
	if len(lines) == 0 {
		t.Fatal("expected non-empty output")
	}
	// Should behave like wrapText when there are no code blocks.
	wrapped := wrapText(body, 40)
	if len(lines) != len(wrapped) {
		t.Fatalf("lines = %d, wrapText = %d", len(lines), len(wrapped))
	}
}

func TestRenderEventBodyUnknownLanguageFallsBack(t *testing.T) {
	body := "```unknownlang123\nsome code\n```"
	lines := renderEventBody(body, 80)
	// Should still produce output (falls back to plain text).
	if len(lines) == 0 {
		t.Fatal("expected non-empty output")
	}
	// Should contain the code text.
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "some code") {
		t.Fatalf("expected code in output: %v", lines)
	}
}

func TestDoubleCtrlCQuits(t *testing.T) {
	fixed := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             func() time.Time { return fixed },
	})

	// First ctrl+c sets pending.
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = updated.(Model)
	if !m.quitPending {
		t.Fatal("expected quitPending after first ctrl+c")
	}
	if cmd == nil {
		t.Fatal("expected timer command after first ctrl+c")
	}

	// Second ctrl+c within deadline should quit.
	updated, cmd = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = updated.(Model)
	_ = m
	// tea.Quit returns a quit msg; we verify by checking cmd is non-nil.
	if cmd == nil {
		t.Fatal("expected quit command after second ctrl+c")
	}
}

func TestDoubleCtrlCResetsAfterTimeout(t *testing.T) {
	fixed := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             func() time.Time { return fixed },
	})

	// First ctrl+c.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = updated.(Model)
	if !m.quitPending {
		t.Fatal("expected quitPending")
	}

	// Simulate timeout.
	updated, _ = m.Update(quitResetMsg{})
	m = updated.(Model)
	if m.quitPending {
		t.Fatal("expected quitPending to reset after timeout")
	}
}

func TestSteerSpacebarWorks(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})
	m.steerOpen = true
	m.steerInput = "hello"

	m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeySpace})
	if m.steerInput != "hello " {
		t.Fatalf("steerInput after space = %q, want %q", m.steerInput, "hello ")
	}
}

func TestTaskEditorCreateOpensEmpty(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})

	m = pressRune(t, m, 'n')
	if !m.taskEditor.open {
		t.Fatal("expected task editor to open after n")
	}
	if m.taskEditor.prompt != "" {
		t.Fatalf("expected empty title, got %q", m.taskEditor.prompt)
	}
}

func TestTaskEditorTabCyclesFields(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})
	m.taskEditor.OpenNew()

	if m.taskEditor.field != 0 {
		t.Fatalf("initial field = %d, want 0", m.taskEditor.field)
	}

	m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.taskEditor.field != 1 {
		t.Fatalf("field after tab = %d, want 1", m.taskEditor.field)
	}

	m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.taskEditor.field != 2 {
		t.Fatalf("field after 2 tabs = %d, want 2", m.taskEditor.field)
	}
}

func TestTaskEditorArrowCyclesOptions(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})
	m.taskEditor.OpenNew()
	m.taskEditor.field = fieldType // type field

	initial := m.taskEditor.taskType
	m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.taskEditor.taskType == initial {
		t.Fatal("expected task type to change after right arrow")
	}
}

func TestTaskEditorEscCancels(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})
	m.taskEditor.OpenNew()
	m.taskEditor.prompt = "some task"

	m = pressKey(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.taskEditor.open {
		t.Fatal("expected task editor to close after esc")
	}
}

func TestTaskEditorSubmitWritesEnqueue(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	fixed := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)

	m := NewModel(Options{
		RuntimeDir:      runtimeDir,
		RefreshInterval: time.Hour,
		Now:             func() time.Time { return fixed },
	})
	m.taskEditor.OpenNew()
	m.taskEditor.prompt = "Fix the tests"

	cmd := m.taskEditor.Submit(m.runtimeDir, m.now)
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	msg := cmd()
	result, ok := msg.(controlResultMsg)
	if !ok {
		t.Fatalf("command msg type = %T, want controlResultMsg", msg)
	}
	if result.err != nil {
		t.Fatalf("submit failed: %v", result.err)
	}

	data, err := os.ReadFile(filepath.Join(runtimeDir, "control.ndjson"))
	if err != nil {
		t.Fatalf("read control.ndjson: %v", err)
	}
	var wrote loop.ControlCommand
	if err := json.Unmarshal(data[:len(data)-1], &wrote); err != nil {
		t.Fatalf("parse control command: %v", err)
	}
	if wrote.Action != "enqueue" {
		t.Fatalf("action = %q, want enqueue", wrote.Action)
	}
	if wrote.Prompt != "Fix the tests" {
		t.Fatalf("prompt = %q, want 'Fix the tests'", wrote.Prompt)
	}
	if wrote.Item == "" {
		t.Fatal("expected item to be set on enqueue command")
	}
	if wrote.TaskKey != "execute" {
		t.Fatalf("task_key = %q, want execute", wrote.TaskKey)
	}
	if wrote.Provider == "" {
		t.Fatal("expected provider to be set on enqueue command")
	}
	if wrote.Model == "" {
		t.Fatal("expected model to be set on enqueue command")
	}
}

func TestTaskEditorOpenEditPrefillsFields(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})

	item := QueueItem{
		ID:       "execute-123",
		TaskKey:  "plan",
		Title:    "Fix the tests",
		Prompt:   "Fix the tests so they pass on CI",
		Provider: "codex",
		Model:    "claude-sonnet-4-6",
		Skill:    "testing",
	}
	m.taskEditor.OpenEdit(item)

	if !m.taskEditor.open {
		t.Fatal("expected task editor to be open")
	}
	if !m.taskEditor.edit {
		t.Fatal("expected edit mode")
	}
	if m.taskEditor.itemID != "execute-123" {
		t.Fatalf("itemID = %q, want execute-123", m.taskEditor.itemID)
	}
	if m.taskEditor.prompt != "Fix the tests so they pass on CI" {
		t.Fatalf("prompt = %q, want full prompt", m.taskEditor.prompt)
	}
	if taskTypes[m.taskEditor.taskType] != "plan" {
		t.Fatalf("taskType = %q, want plan", taskTypes[m.taskEditor.taskType])
	}
	if providers[m.taskEditor.provider] != "codex" {
		t.Fatalf("provider = %q, want codex", providers[m.taskEditor.provider])
	}
	if models[m.taskEditor.model] != "claude-sonnet-4-6" {
		t.Fatalf("model = %q, want claude-sonnet-4-6", models[m.taskEditor.model])
	}
	if m.taskEditor.skill != "testing" {
		t.Fatalf("skill = %q, want testing", m.taskEditor.skill)
	}
}

func TestTaskEditorEditSubmitsEditItem(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	fixed := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)

	m := NewModel(Options{
		RuntimeDir:      runtimeDir,
		RefreshInterval: time.Hour,
		Now:             func() time.Time { return fixed },
	})
	m.taskEditor.OpenEdit(QueueItem{
		ID:       "execute-42",
		TaskKey:  "execute",
		Prompt:   "Original prompt",
		Provider: "claude",
		Model:    "claude-opus-4-6",
	})
	m.taskEditor.prompt = "Updated prompt text"

	cmd := m.taskEditor.Submit(m.runtimeDir, m.now)
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	msg := cmd()
	result, ok := msg.(controlResultMsg)
	if !ok {
		t.Fatalf("command msg type = %T, want controlResultMsg", msg)
	}
	if result.err != nil {
		t.Fatalf("submit failed: %v", result.err)
	}

	data, err := os.ReadFile(filepath.Join(runtimeDir, "control.ndjson"))
	if err != nil {
		t.Fatalf("read control.ndjson: %v", err)
	}
	var wrote loop.ControlCommand
	if err := json.Unmarshal(data[:len(data)-1], &wrote); err != nil {
		t.Fatalf("parse control command: %v", err)
	}
	if wrote.Action != "edit-item" {
		t.Fatalf("action = %q, want edit-item", wrote.Action)
	}
	if wrote.Item != "execute-42" {
		t.Fatalf("item = %q, want execute-42", wrote.Item)
	}
	if wrote.Prompt != "Updated prompt text" {
		t.Fatalf("prompt = %q, want 'Updated prompt text'", wrote.Prompt)
	}
}

func TestTaskEditorEditFallsBackToTitleWhenNoPrompt(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})

	// Simulate an older queue item that only has Title, no Prompt
	item := QueueItem{
		ID:      "execute-99",
		TaskKey: "execute",
		Title:   "Legacy title only",
	}
	m.taskEditor.OpenEdit(item)

	if m.taskEditor.prompt != "Legacy title only" {
		t.Fatalf("prompt = %q, want fallback to title", m.taskEditor.prompt)
	}
}

func TestSteerOpensWithBacktick(t *testing.T) {
	m := NewModel(Options{
		RuntimeDir:      t.TempDir(),
		RefreshInterval: time.Hour,
		Now:             time.Now,
	})

	m = pressRune(t, m, '`')
	if !m.steerOpen {
		t.Fatal("expected steerOpen after backtick")
	}
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
	return pressKey(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
}

func pressKey(t *testing.T, m Model, key tea.KeyPressMsg) Model {
	t.Helper()
	updated, _ := m.Update(key)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("update model type = %T, want tui.Model", updated)
	}
	return next
}

func TestTabBarReviewsCountInTitle(t *testing.T) {
	// With 0 pending reviews, the tab title should be just "Reviews"
	bar0 := renderTabBar(TabReviews, 120, 0)
	if !strings.Contains(bar0, "Reviews") {
		t.Fatalf("expected 'Reviews' in tab bar, got %q", bar0)
	}
	if strings.Contains(bar0, "Reviews (") {
		t.Fatalf("expected no count suffix when pendingReviewCount=0, got %q", bar0)
	}

	// With 2 pending reviews, the tab title should be "Reviews (2)"
	bar2 := renderTabBar(TabReviews, 120, 2)
	if !strings.Contains(bar2, "Reviews (2)") {
		t.Fatalf("expected 'Reviews (2)' in tab bar, got %q", bar2)
	}
}
