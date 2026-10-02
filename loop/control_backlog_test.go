package loop

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/mise"
	loopruntime "github.com/poteto/noodle/runtime"
)

type backlogAdapterCall struct {
	action string
	args   []string
	stdin  string
}

// recordingAdapter records backlog adapter calls and fails every call while
// err is set.
type recordingAdapter struct {
	calls []backlogAdapterCall
	err   error
}

func (r *recordingAdapter) Run(_ context.Context, adapterName, action string, opts adapter.RunOptions) (string, error) {
	if adapterName != "backlog" {
		return "", nil
	}
	r.calls = append(r.calls, backlogAdapterCall{action: action, args: opts.Args, stdin: string(opts.Stdin)})
	return "", r.err
}

func newBacklogTestLoop(t *testing.T, ar *recordingAdapter, backlog []adapter.BacklogItem) *Loop {
	t.Helper()
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	return New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
		Runtimes:   map[string]loopruntime.Runtime{"process": newMockRuntime()},
		Worktree:   &fakeWorktree{},
		Adapter:    ar,
		Mise:       &fakeMise{brief: mise.Brief{Backlog: backlog}},
		Monitor:    fakeMonitor{},
		Registry:   testLoopRegistry(),
		Now:        time.Now,
		OrdersFile: filepath.Join(runtimeDir, "orders.json"),
	})
}

// cycleWithControls writes commands to control.ndjson and runs one cycle, the
// path a UI control takes.
func cycleWithControls(t *testing.T, l *Loop, cmds ...ControlCommand) {
	t.Helper()
	var lines []byte
	for _, cmd := range cmds {
		line, err := json.Marshal(cmd)
		if err != nil {
			t.Fatalf("encode control: %v", err)
		}
		lines = append(append(lines, line...), '\n')
	}
	if err := os.WriteFile(filepath.Join(l.runtimeDir, "control.ndjson"), lines, 0o644); err != nil {
		t.Fatalf("write control: %v", err)
	}
	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
}

func TestBacklogControlsRunTheBacklogAdapter(t *testing.T) {
	ar := &recordingAdapter{}
	l := newBacklogTestLoop(t, ar, nil)

	title := `Ship "v2" / fix & polish`
	cycleWithControls(t, l,
		ControlCommand{ID: "a", Action: "backlog-add", Prompt: title},
		ControlCommand{ID: "e", Action: "backlog-edit", Target: "7", Prompt: "Renamed"},
		ControlCommand{ID: "d", Action: "backlog-done", Target: "7"},
	)

	if len(ar.calls) != 3 {
		t.Fatalf("adapter calls = %+v, want add, edit, done", ar.calls)
	}
	add, edit, done := ar.calls[0], ar.calls[1], ar.calls[2]

	var payload map[string]string
	if add.action != "add" || len(add.args) != 0 {
		t.Fatalf("add call = %+v", add)
	}
	if err := json.Unmarshal([]byte(add.stdin), &payload); err != nil || payload["title"] != title {
		t.Fatalf("add stdin = %q, want JSON title %q", add.stdin, title)
	}
	// Adapters are shell scripts that store what they read: "&" must arrive
	// as itself, not as a \u0026 escape.
	if strings.Contains(add.stdin, `\u0026`) {
		t.Fatalf("add stdin = %q, want an unescaped ampersand", add.stdin)
	}
	if edit.action != "edit" || len(edit.args) != 1 || edit.args[0] != "7" {
		t.Fatalf("edit call = %+v", edit)
	}
	if err := json.Unmarshal([]byte(edit.stdin), &payload); err != nil || payload["title"] != "Renamed" {
		t.Fatalf("edit stdin = %q, want JSON title %q", edit.stdin, "Renamed")
	}
	if done.action != "done" || len(done.args) != 1 || done.args[0] != "7" || done.stdin != "" {
		t.Fatalf("done call = %+v", done)
	}
	if got := l.State().BacklogError; got != "" {
		t.Fatalf("BacklogError = %q after successful controls", got)
	}
}

func TestBacklogControlsRejectMissingFieldsWithoutRunningTheAdapter(t *testing.T) {
	ar := &recordingAdapter{}
	l := newBacklogTestLoop(t, ar, nil)

	for _, cmd := range []ControlCommand{
		{Action: "backlog-add", Prompt: "  "},
		{Action: "backlog-edit", Target: "", Prompt: "title"},
		{Action: "backlog-edit", Target: "7", Prompt: ""},
		{Action: "backlog-done", Target: " "},
	} {
		if ack := l.applyControlCommand(cmd); ack.Status != "error" {
			t.Errorf("%s %+v: ack status = %q, want error", cmd.Action, cmd, ack.Status)
		}
	}
	if len(ar.calls) != 0 {
		t.Fatalf("adapter ran for invalid commands: %+v", ar.calls)
	}
}

func TestBacklogControlFailureIsPublishedUntilALaterOneSucceeds(t *testing.T) {
	ar := &recordingAdapter{err: errors.New("todos file not found")}
	l := newBacklogTestLoop(t, ar, nil)

	cycleWithControls(t, l, ControlCommand{ID: "fail", Action: "backlog-done", Target: "3"})
	if got := l.State().BacklogError; got != "todos file not found" {
		t.Fatalf("BacklogError = %q, want the adapter failure", got)
	}

	ar.err = nil
	cycleWithControls(t, l, ControlCommand{ID: "ok", Action: "backlog-done", Target: "3"})
	if got := l.State().BacklogError; got != "" {
		t.Fatalf("BacklogError = %q, want it cleared by the successful control", got)
	}
}

func TestPublishedStateCarriesTheSyncedBacklog(t *testing.T) {
	backlog := []adapter.BacklogItem{
		{ID: "1", Title: "Open item", Status: adapter.BacklogStatusOpen, Extra: map[string]any{"section": "Inbox"}},
		{ID: "2", Title: "In progress item", Status: adapter.BacklogStatusInProgress},
	}
	l := newBacklogTestLoop(t, &recordingAdapter{}, backlog)

	if err := l.Cycle(context.Background()); err != nil {
		t.Fatalf("cycle: %v", err)
	}
	state := l.State()
	if len(state.Backlog) != 2 || state.Backlog[0].ID != "1" || state.Backlog[1].Status != adapter.BacklogStatusInProgress {
		t.Fatalf("published backlog = %+v, want the synced items", state.Backlog)
	}

	// The published copy must not alias the loop's own items.
	state.Backlog[0].Extra["section"] = "Changed"
	if got := l.State().Backlog[0].Extra["section"]; got != "Inbox" {
		t.Fatalf("loop backlog section = %v after mutating a published copy", got)
	}
}
