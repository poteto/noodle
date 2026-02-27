package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/testutil/fixturedir"
	"github.com/poteto/noodle/loop"
	"github.com/poteto/noodle/mise"
)

type snapshotFixtureInput struct {
	Now time.Time `json:"now"`
}

func TestSnapshotDirectoryFixtures(t *testing.T) {
	inventory := fixturedir.LoadInventory(t, "testdata")
	fixturedir.AssertValidFixtureRoot(t, "testdata")

	mode := strings.ToLower(strings.TrimSpace(os.Getenv("NOODLE_SNAPSHOT_FIXTURE_MODE")))
	if mode == "" {
		mode = "check"
	}
	if mode != "check" && mode != "record" {
		t.Fatalf("invalid NOODLE_SNAPSHOT_FIXTURE_MODE %q (expected check|record)", mode)
	}

	for _, fixtureCase := range inventory.Cases {
		fixtureCase := fixtureCase
		t.Run(fixtureCase.Name, func(t *testing.T) {
			state := fixturedir.RequireSingleState(t, fixtureCase)

			input, _ := fixturedir.ParseOptionalStateJSON[snapshotFixtureInput](t, state, "input.json")
			now := input.Now
			if now.IsZero() {
				now = time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
			}

			projectDir := t.TempDir()
			runtimeDir := filepath.Join(projectDir, ".noodle")
			if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
				t.Fatalf("mkdir runtime: %v", err)
			}

			fixturedir.ApplyRuntimeSnapshot(t, state, runtimeDir)

			loopState := loadFixtureLoopState(t, runtimeDir)
			snap, err := LoadSnapshot(runtimeDir, now, loopState)
			if mode == "check" {
				fixturedir.AssertError(t, fixtureCase.Name, err, fixtureCase.ExpectedError)
				if err != nil {
					return
				}

				expected := fixturedir.ParseSectionJSON[Snapshot](t, fixtureCase, "Expected Snapshot")
				if !snapshotsEqual(snap, expected) {
					actualJSON := mustJSONIndent(snap)
					expectedJSON := mustJSONIndent(expected)
					t.Fatalf("snapshot mismatch\nactual:\n%s\nexpected:\n%s", actualJSON, expectedJSON)
				}
				return
			}

			// Record mode.
			if err != nil {
				t.Fatalf("LoadSnapshot failed in record mode: %v", err)
			}
			if err := fixturedir.WriteSectionToExpected(fixtureCase.Layout.ExpectedPath, "Expected Snapshot", snap); err != nil {
				t.Fatalf("write expected snapshot: %v", err)
			}
		})
	}
}

// loadFixtureLoopState builds a LoopState from a fixture's .noodle/ directory.
// This bridges the old fixture format (filesystem-based session/order data) to
// the current LoadSnapshot API which takes LoopState as input.
func loadFixtureLoopState(t *testing.T, runtimeDir string) loop.LoopState {
	t.Helper()
	var state loop.LoopState

	// Read status.json for loop metadata.
	if data, err := os.ReadFile(filepath.Join(runtimeDir, "status.json")); err == nil {
		var s struct {
			LoopState string `json:"loop_state"`
			MaxCooks  int    `json:"max_cooks"`
			Autonomy  string `json:"autonomy"`
		}
		if json.Unmarshal(data, &s) == nil {
			state.Status = s.LoopState
			state.MaxCooks = s.MaxCooks
			state.Autonomy = s.Autonomy
		}
	}

	// Read orders.json.
	if data, err := os.ReadFile(filepath.Join(runtimeDir, "orders.json")); err == nil {
		var of orderx.OrdersFile
		if json.Unmarshal(data, &of) == nil {
			state.Orders = of.Orders
		}
	}

	// Read pending-review.json.
	if data, err := os.ReadFile(filepath.Join(runtimeDir, "pending-review.json")); err == nil {
		var pr struct {
			Items []loop.PendingReviewItem `json:"items"`
		}
		if json.Unmarshal(data, &pr) == nil {
			state.PendingReviews = pr.Items
			state.PendingReviewCount = len(pr.Items)
		}
	}

	// Read sessions.
	sessionsDir := filepath.Join(runtimeDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return state
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionID := entry.Name()
		sessionDir := filepath.Join(sessionsDir, sessionID)

		var meta struct {
			Status          string    `json:"status"`
			Provider        string    `json:"provider"`
			Model           string    `json:"model"`
			TotalCostUSD    float64   `json:"total_cost_usd"`
			DurationSeconds int64     `json:"duration_seconds"`
			LastActivity    time.Time `json:"last_activity"`
		}
		if data, err := os.ReadFile(filepath.Join(sessionDir, "meta.json")); err == nil {
			json.Unmarshal(data, &meta)
		}

		var spawn struct {
			DisplayName  string `json:"display_name"`
			WorktreePath string `json:"worktree_path"`
			Title        string `json:"title"`
		}
		if data, err := os.ReadFile(filepath.Join(sessionDir, "spawn.json")); err == nil {
			json.Unmarshal(data, &spawn)
		}

		status := strings.ToLower(strings.TrimSpace(meta.Status))
		if status == "completed" || status == "failed" {
			state.RecentHistory = append(state.RecentHistory, mise.HistoryItem{
				SessionID:   sessionID,
				Status:      status,
				DurationS:   meta.DurationSeconds,
				CompletedAt: meta.LastActivity,
			})
		} else {
			orderID, taskKey := parseWorktreeOrderTask(spawn.WorktreePath)
			state.ActiveCooks = append(state.ActiveCooks, loop.CookSummary{
				SessionID:    sessionID,
				OrderID:      orderID,
				TaskKey:      taskKey,
				Provider:     meta.Provider,
				Model:        meta.Model,
				DisplayName:  spawn.DisplayName,
				Status:       status,
				TotalCostUSD: meta.TotalCostUSD,
			})
		}
	}

	// Sort for deterministic output.
	sort.Slice(state.ActiveCooks, func(i, j int) bool {
		return state.ActiveCooks[i].SessionID < state.ActiveCooks[j].SessionID
	})
	sort.Slice(state.RecentHistory, func(i, j int) bool {
		if state.RecentHistory[i].CompletedAt.Equal(state.RecentHistory[j].CompletedAt) {
			return state.RecentHistory[i].SessionID < state.RecentHistory[j].SessionID
		}
		return state.RecentHistory[i].CompletedAt.After(state.RecentHistory[j].CompletedAt)
	})

	return state
}

// parseWorktreeOrderTask extracts orderID and taskKey from a worktree path.
// Worktree names follow the pattern: {orderID}-{stageIndex}-{taskKey}
// e.g., /tmp/.worktrees/order-1-0-execute → ("order-1", "execute")
func parseWorktreeOrderTask(worktreePath string) (orderID, taskKey string) {
	if worktreePath == "" {
		return "", ""
	}
	name := filepath.Base(worktreePath)
	parts := strings.Split(name, "-")
	if len(parts) < 3 {
		return "", ""
	}
	// Scan from second-to-last rightward to find the stage index (a number).
	for i := len(parts) - 2; i >= 1; i-- {
		if _, err := strconv.Atoi(parts[i]); err == nil {
			return strings.Join(parts[:i], "-"), strings.Join(parts[i+1:], "-")
		}
	}
	return "", ""
}

func snapshotsEqual(a, b Snapshot) bool {
	// Compare via JSON to handle nil vs empty slice differences.
	// reflect.DeepEqual distinguishes nil from []T{} but JSON treats both as [].
	aj, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bj, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(aj) == string(bj)
}

func mustJSONIndent(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(data)
}
