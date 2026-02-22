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
	"github.com/poteto/noodle/internal/testutil/fixturemd"
	"github.com/poteto/noodle/mise"
)

type loopFixtureSetup struct {
	QueueItems                             []QueueItem          `json:"queue_items"`
	MiseResults                            []loopFixtureMiseRun `json:"mise_results"`
	SpawnerError                           string               `json:"spawner_error"`
	RunningRuntimeRepairSessionID          string               `json:"running_runtime_repair_session_id"`
	CompleteRuntimeRepairSessionWithStatus string               `json:"complete_runtime_repair_session_with_status"`
}

type loopFixtureMiseRun struct {
	Backlog  []adapter.BacklogItem `json:"backlog"`
	Warnings []string              `json:"warnings"`
	Error    string                `json:"error"`
}

type loopFixtureExpected struct {
	SpawnCalls               int    `json:"spawn_calls"`
	FirstSpawnName           string `json:"first_spawn_name"`
	FirstSpawnNamePrefix     string `json:"first_spawn_name_prefix"`
	CreatedWorktrees         int    `json:"created_worktrees"`
	RuntimeRepairInFlight    *bool  `json:"runtime_repair_in_flight"`
	FirstCycleErrorContains  string `json:"first_cycle_error_contains"`
	SecondCycleErrorContains string `json:"second_cycle_error_contains"`
}

func TestLoopMarkdownFixtures(t *testing.T) {
	paths := fixturemd.Paths(t, "testdata")

	for _, fixturePath := range paths {
		fixturePath := fixturePath
		t.Run(filepath.Base(fixturePath), func(t *testing.T) {
			expectErrorFixture := fixturemd.IsErrorFixture(fixturePath)
			setup := parseLoopFixtureSetup(t, fixturePath)
			expected := parseLoopFixtureExpected(t, fixturePath)

			projectDir := t.TempDir()
			runtimeDir := filepath.Join(projectDir, ".noodle")
			if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
				t.Fatalf("mkdir runtime: %v", err)
			}
			queuePath := filepath.Join(runtimeDir, "queue.json")
			if err := writeQueueAtomic(queuePath, Queue{Items: setup.QueueItems}); err != nil {
				t.Fatalf("write queue: %v", err)
			}

			miseResults := make([]fakeMiseResult, 0, len(setup.MiseResults))
			for _, result := range setup.MiseResults {
				var resultErr error
				if strings.TrimSpace(result.Error) != "" {
					resultErr = errors.New(strings.TrimSpace(result.Error))
				}
				miseResults = append(miseResults, fakeMiseResult{
					brief:    mise.Brief{Backlog: result.Backlog},
					warnings: result.Warnings,
					err:      resultErr,
				})
			}

			sp := &fakeSpawner{}
			if strings.TrimSpace(setup.SpawnerError) != "" {
				sp.spawnErr = errors.New(strings.TrimSpace(setup.SpawnerError))
			}
			if sessionID := strings.TrimSpace(setup.RunningRuntimeRepairSessionID); sessionID != "" {
				if err := os.MkdirAll(filepath.Join(runtimeDir, "sessions", sessionID), 0o755); err != nil {
					t.Fatalf("mkdir running repair session: %v", err)
				}
				meta := `{"session_id":"` + sessionID + `","status":"running"}`
				if err := os.WriteFile(
					filepath.Join(runtimeDir, "sessions", sessionID, "meta.json"),
					[]byte(meta),
					0o644,
				); err != nil {
					t.Fatalf("write running repair session meta: %v", err)
				}
			}

			wt := &fakeWorktree{}
			l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
				Spawner:   sp,
				Worktree:  wt,
				Adapter:   &fakeAdapterRunner{},
				Mise:      &fakeMise{results: miseResults},
				Monitor:   fakeMonitor{},
				Now:       time.Now,
				QueueFile: queuePath,
			})

			firstErr := l.Cycle(context.Background())
			if expectErrorFixture && strings.TrimSpace(expected.FirstCycleErrorContains) == "" {
				if firstErr == nil {
					t.Fatalf("first cycle expected an error for fixture %s", filepath.Base(fixturePath))
				}
			} else {
				assertLoopFixtureError(t, "first cycle", firstErr, expected.FirstCycleErrorContains)
			}

			runSecondCycle := strings.TrimSpace(setup.CompleteRuntimeRepairSessionWithStatus) != "" ||
				strings.TrimSpace(expected.SecondCycleErrorContains) != ""
			if runSecondCycle {
				if len(sp.sessions) == 0 {
					t.Fatal("fixture expected a spawned runtime repair session before second cycle")
				}
				status := strings.TrimSpace(setup.CompleteRuntimeRepairSessionWithStatus)
				if status == "" {
					status = "completed"
				}
				sp.sessions[0].status = status
				select {
				case <-sp.sessions[0].done:
				default:
					close(sp.sessions[0].done)
				}
				secondErr := l.Cycle(context.Background())
				assertLoopFixtureError(t, "second cycle", secondErr, expected.SecondCycleErrorContains)
			}

			if len(sp.calls) != expected.SpawnCalls {
				t.Fatalf("spawn calls = %d, want %d", len(sp.calls), expected.SpawnCalls)
			}
			if len(wt.created) != expected.CreatedWorktrees {
				t.Fatalf("created worktrees = %d, want %d", len(wt.created), expected.CreatedWorktrees)
			}
			if expected.SpawnCalls > 0 {
				firstName := sp.calls[0].Name
				if expected.FirstSpawnName != "" && firstName != expected.FirstSpawnName {
					t.Fatalf("first spawn name = %q, want %q", firstName, expected.FirstSpawnName)
				}
				if expected.FirstSpawnNamePrefix != "" && !strings.HasPrefix(firstName, expected.FirstSpawnNamePrefix) {
					t.Fatalf("first spawn name = %q, want prefix %q", firstName, expected.FirstSpawnNamePrefix)
				}
			}
			if expected.RuntimeRepairInFlight != nil {
				got := l.runtimeRepairInFlight != nil
				if got != *expected.RuntimeRepairInFlight {
					t.Fatalf("runtimeRepairInFlight = %v, want %v", got, *expected.RuntimeRepairInFlight)
				}
			}

		})
	}
}

func parseLoopFixtureSetup(t *testing.T, fixturePath string) loopFixtureSetup {
	t.Helper()
	raw := strings.Join(fixturemd.ReadSectionLines(t, fixturePath, "Setup"), "\n")
	var setup loopFixtureSetup
	if err := json.Unmarshal([]byte(raw), &setup); err != nil {
		t.Fatalf("parse setup fixture %s: %v", fixturePath, err)
	}
	return setup
}

func parseLoopFixtureExpected(t *testing.T, fixturePath string) loopFixtureExpected {
	t.Helper()
	raw := strings.Join(fixturemd.ReadSectionLines(t, fixturePath, "Expected"), "\n")
	var expected loopFixtureExpected
	if err := json.Unmarshal([]byte(raw), &expected); err != nil {
		t.Fatalf("parse expected fixture %s: %v", fixturePath, err)
	}
	return expected
}

func assertLoopFixtureError(t *testing.T, phase string, err error, wantContains string) {
	t.Helper()
	if strings.TrimSpace(wantContains) == "" {
		if err != nil {
			t.Fatalf("%s error: %v", phase, err)
		}
		return
	}
	if err == nil {
		t.Fatalf("%s expected error containing %q", phase, wantContains)
	}
	if !strings.Contains(err.Error(), wantContains) {
		t.Fatalf("%s error = %q, want contains %q", phase, err.Error(), wantContains)
	}
}
