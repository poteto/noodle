package loop

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/config"
	loopruntime "github.com/poteto/noodle/runtime"
)

func TestRefreshAdoptedTargetsPrunesDeadPIDs(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")

	// Create two adopted sessions: one alive (our PID), one dead (bogus PID).
	aliveSessionID := "cook-alive"
	deadSessionID := "cook-dead"
	for _, sid := range []string{aliveSessionID, deadSessionID} {
		sessionDir := filepath.Join(runtimeDir, "sessions", sid)
		if err := os.MkdirAll(sessionDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sessionDir, err)
		}

		// Write meta.json with "running" status.
		metaJSON, _ := json.Marshal(map[string]any{
			"session_id": sid,
			"status":     "running",
		})
		if err := os.WriteFile(filepath.Join(sessionDir, "meta.json"), metaJSON, 0o644); err != nil {
			t.Fatalf("write meta.json for %s: %v", sid, err)
		}
	}

	// Write process.json: alive session gets our PID, dead session gets bogus PID.
	writeProcessJSON(t, filepath.Join(runtimeDir, "sessions", aliveSessionID), os.Getpid())
	writeProcessJSON(t, filepath.Join(runtimeDir, "sessions", deadSessionID), 2147483647)

	l := &Loop{
		projectDir: projectDir,
		runtimeDir: runtimeDir,
		config:     config.DefaultConfig(),
		logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
		deps: Dependencies{
			Runtimes: map[string]loopruntime.Runtime{"process": newMockRuntime()},
			Now:      time.Now,
		},
		cooks: cookTracker{
			activeCooksByOrder: map[string]*cookHandle{},
			adoptedTargets: map[string]string{
				"order-a": aliveSessionID,
				"order-b": deadSessionID,
			},
				adoptedSessions: []string{aliveSessionID, deadSessionID},
				failedTargets:   map[string]string{},
				pendingReview:   map[string]*pendingReviewCook{},
			},
		}

	l.refreshAdoptedTargets()

	// Alive session should be retained.
	if _, ok := l.cooks.adoptedTargets["order-a"]; !ok {
		t.Fatal("expected alive session to be retained in adoptedTargets")
	}

	// Dead session should be pruned.
	if _, ok := l.cooks.adoptedTargets["order-b"]; ok {
		t.Fatal("expected dead session to be pruned from adoptedTargets")
	}

	if len(l.cooks.adoptedSessions) != 1 {
		t.Fatalf("adoptedSessions = %d, want 1", len(l.cooks.adoptedSessions))
	}
	if l.cooks.adoptedSessions[0] != aliveSessionID {
		t.Fatalf("adoptedSessions[0] = %q, want %q", l.cooks.adoptedSessions[0], aliveSessionID)
	}
}

func TestRefreshAdoptedTargetsNoOpWhenEmpty(t *testing.T) {
	l := &Loop{
		runtimeDir: t.TempDir(),
		logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
		cooks: cookTracker{
			adoptedTargets:  map[string]string{},
			adoptedSessions: []string{},
		},
	}

	l.refreshAdoptedTargets()

	if len(l.cooks.adoptedTargets) != 0 {
		t.Fatalf("adoptedTargets = %d, want 0", len(l.cooks.adoptedTargets))
	}
}

func TestRefreshAdoptedTargetsMissingProcessJSONPrunes(t *testing.T) {
	projectDir := t.TempDir()
	runtimeDir := filepath.Join(projectDir, ".noodle")
	sessionID := "cook-no-process"
	sessionDir := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// meta.json with running status but no process.json.
	metaJSON, _ := json.Marshal(map[string]any{
		"session_id": sessionID,
		"status":     "running",
	})
	if err := os.WriteFile(filepath.Join(sessionDir, "meta.json"), metaJSON, 0o644); err != nil {
		t.Fatalf("write meta.json: %v", err)
	}

	l := &Loop{
		projectDir: projectDir,
		runtimeDir: runtimeDir,
		logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
		cooks: cookTracker{
			adoptedTargets:  map[string]string{"order-x": sessionID},
			adoptedSessions: []string{sessionID},
		},
	}

	l.refreshAdoptedTargets()

	// Missing process.json means SessionPIDAlive returns false — session is pruned.
	if len(l.cooks.adoptedTargets) != 0 {
		t.Fatalf("adoptedTargets = %d, want 0 (missing process.json)", len(l.cooks.adoptedTargets))
	}
}

func writeProcessJSON(t *testing.T, sessionDir string, pid int) {
	t.Helper()
	meta, _ := json.Marshal(map[string]any{
		"pid":        pid,
		"session_id": filepath.Base(sessionDir),
		"started_at": time.Now().UTC().Format(time.RFC3339),
	})
	if err := os.WriteFile(filepath.Join(sessionDir, "process.json"), meta, 0o644); err != nil {
		t.Fatalf("write process.json: %v", err)
	}
}
