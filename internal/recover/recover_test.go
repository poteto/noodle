package recover

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/event"
)

func TestRecoveryChainLength(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{name: "task-1", want: 0},
		{name: "task-1-recover-1", want: 1},
		{name: "task-1-recover-3", want: 3},
	}
	for _, test := range cases {
		if got := RecoveryChainLength(test.name); got != test.want {
			t.Fatalf("chain length for %q = %d, want %d", test.name, got, test.want)
		}
	}
}

func TestBuildResumeContext(t *testing.T) {
	ctx := BuildResumeContext(RecoveryInfo{
		ExitReason:   "tmux session died",
		LastAction:   "Edit src/auth/token.ts",
		FilesChanged: []string{"src/auth/token.ts", "src/auth/middleware.ts"},
	}, 2, 3)

	if ctx.Attempt != 2 {
		t.Fatalf("attempt = %d", ctx.Attempt)
	}
	if !strings.Contains(ctx.Summary, "Failure: tmux session died") {
		t.Fatalf("summary missing failure: %s", ctx.Summary)
	}
	if !strings.Contains(ctx.Summary, "Attempt: 2/3") {
		t.Fatalf("summary missing attempt: %s", ctx.Summary)
	}
}

func TestCollectRecoveryInfo(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	writer, err := event.NewEventWriter(runtimeDir, "cook-a")
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}

	mustAppend := func(record event.Event) {
		t.Helper()
		if err := writer.Append(context.Background(), record); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	mustAppend(event.Event{
		Type:      event.EventAction,
		Payload:   mustPayload(t, map[string]any{"message": "Edit src/auth/token.ts"}),
		Timestamp: time.Date(2026, 2, 22, 22, 0, 0, 0, time.UTC),
	})
	mustAppend(event.Event{
		Type:      event.EventStateChange,
		Payload:   mustPayload(t, map[string]any{"reason": "provider timeout"}),
		Timestamp: time.Date(2026, 2, 22, 22, 1, 0, 0, time.UTC),
	})

	info, err := CollectRecoveryInfo(context.Background(), runtimeDir, "cook-a")
	if err != nil {
		t.Fatalf("collect recovery info: %v", err)
	}
	if info.LastAction != "Edit src/auth/token.ts" {
		t.Fatalf("last action = %q", info.LastAction)
	}
	if info.ExitReason != "provider timeout" {
		t.Fatalf("exit reason = %q", info.ExitReason)
	}
	if len(info.FilesChanged) == 0 {
		t.Fatal("expected file hints from action payload")
	}
}

func TestCollectRecoveryInfoFallsBackToStderrReason(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	sessionID := "cook-a"
	sessionDir := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	stderr := "Booting codex runtime...\nError: unable to start codex agent: missing API key\n"
	if err := os.WriteFile(filepath.Join(sessionDir, "stderr.log"), []byte(stderr), 0o644); err != nil {
		t.Fatalf("write stderr log: %v", err)
	}

	info, err := CollectRecoveryInfo(context.Background(), runtimeDir, sessionID)
	if err != nil {
		t.Fatalf("collect recovery info: %v", err)
	}
	if info.ExitReason != "Error: unable to start codex agent: missing API key" {
		t.Fatalf("exit reason = %q", info.ExitReason)
	}
}

func mustPayload(t *testing.T, value map[string]any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return encoded
}
