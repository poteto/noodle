package loop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/testutil/fixturemd"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/spawner"
)

type loopFixtureSetup struct {
	QueueItems                    []QueueItem             `json:"queue_items"`
	MiseResults                   []loopFixtureMiseRun    `json:"mise_results"`
	SpawnerError                  string                  `json:"spawner_error"`
	ExtraCycles                   int                     `json:"extra_cycles"`
	CycleInputs                   []loopFixtureCycleInput `json:"cycle_inputs"`
	Phases                        map[string]string       `json:"phases"`
	RoutingProvider               string                  `json:"routing_provider"`
	RoutingModel                  string                  `json:"routing_model"`
	RunningRuntimeRepairSessionID string                  `json:"running_runtime_repair_session_id"`
}

type loopFixtureCycleInput struct {
	RuntimeRepairSessionStatus string `json:"runtime_repair_session_status"`
	RuntimeRepairSessionIndex  *int   `json:"runtime_repair_session_index"`
}

type loopFixtureMiseRun struct {
	Backlog  []adapter.BacklogItem `json:"backlog"`
	Warnings []string              `json:"warnings"`
	Error    string                `json:"error"`
}

type loopFixtureExpected struct {
	StepErrors  []fixturemd.ErrorExpectation           `json:"step_errors"`
	Actions     map[string]bool                        `json:"actions"`
	State       map[string]bool                        `json:"state"`
	Transitions []string                               `json:"transitions"`
	Counts      map[string]fixturemd.CountExpectation  `json:"counts"`
	Absence     map[string]bool                        `json:"absence"`
	Routing     map[string]fixturemd.StringExpectation `json:"routing"`
	Idempotence map[string]bool                        `json:"idempotence"`
}

func TestLoopMarkdownFixtures(t *testing.T) {
	paths := fixturemd.Paths(t, "testdata")

	for _, fixturePath := range paths {
		fixturePath := fixturePath
		t.Run(filepath.Base(fixturePath), func(t *testing.T) {
			setup := fixturemd.ParseSectionJSON[loopFixtureSetup](t, fixturePath, "Setup")
			expected := fixturemd.ParseSectionJSON[loopFixtureExpected](t, fixturePath, "Expected")

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
			cfg := config.DefaultConfig()
			if len(setup.Phases) > 0 {
				for key, value := range setup.Phases {
					cfg.Phases[strings.TrimSpace(key)] = value
				}
			}
			if value := strings.TrimSpace(setup.RoutingProvider); value != "" {
				cfg.Routing.Defaults.Provider = value
			}
			if value := strings.TrimSpace(setup.RoutingModel); value != "" {
				cfg.Routing.Defaults.Model = value
			}

			l := New(projectDir, "noodle", config.DefaultConfig(), Dependencies{
				Spawner:   sp,
				Worktree:  wt,
				Adapter:   &fakeAdapterRunner{},
				Mise:      &fakeMise{results: miseResults},
				Monitor:   fakeMonitor{},
				Now:       time.Now,
				QueueFile: queuePath,
			})
			l.config = cfg

			stateTransitions := make([]string, 0, 1+len(setup.CycleInputs)+setup.ExtraCycles)

			firstErr := l.Cycle(context.Background())
			stateTransitions = append(stateTransitions, strings.ToLower(string(l.state)))
			fixturemd.AssertError(t, "first cycle", firstErr, fixturemd.ExpectedError(t, fixturePath))

			for index, input := range setup.CycleInputs {
				sessionIndex := index
				if input.RuntimeRepairSessionIndex != nil {
					sessionIndex = *input.RuntimeRepairSessionIndex
				}
				status := strings.TrimSpace(input.RuntimeRepairSessionStatus)
				if status != "" {
					if sessionIndex >= len(sp.sessions) {
						t.Fatalf("missing runtime repair session at index %d for status %q", sessionIndex, status)
					}
					sp.sessions[sessionIndex].status = status
					select {
					case <-sp.sessions[sessionIndex].done:
					default:
						close(sp.sessions[sessionIndex].done)
					}
				}

				err := l.Cycle(context.Background())
				stateTransitions = append(stateTransitions, strings.ToLower(string(l.state)))
				if index < len(expected.StepErrors) {
					stepExpectation := expected.StepErrors[index]
					fixturemd.AssertError(t, fmt.Sprintf("step cycle %d", index+2), err, &stepExpectation)
				} else {
					fixturemd.AssertError(t, fmt.Sprintf("step cycle %d", index+2), err, nil)
				}
			}

			spawnsBeforeExtra := len(sp.calls)
			repairsBeforeExtra := len(runtimeRepairCalls(sp.calls))
			for i := 0; i < setup.ExtraCycles; i++ {
				err := l.Cycle(context.Background())
				fixturemd.AssertError(t, fmt.Sprintf("extra cycle %d", i+1), err, nil)
				stateTransitions = append(stateTransitions, strings.ToLower(string(l.state)))
			}

			repairCalls := runtimeRepairCalls(sp.calls)
			normalCalls := normalSpawnCalls(sp.calls)
			firstSpawn := spawner.SpawnRequest{}
			if len(sp.calls) > 0 {
				firstSpawn = sp.calls[0]
			}
			firstRepair := spawner.SpawnRequest{}
			if len(repairCalls) > 0 {
				firstRepair = repairCalls[0]
			}

			if len(expected.Transitions) > 0 {
				fixturemd.AssertSequence(t, "transitions", stateTransitions, expected.Transitions)
			}
			if len(expected.Actions) > 0 {
				fixturemd.AssertBools(t, "actions", map[string]bool{
					"repair_task_scheduled": len(repairCalls) > 0,
					"oops_task_scheduled":   hasSkill(repairCalls, "oops"),
					"normal_task_scheduled": len(normalCalls) > 0,
				}, expected.Actions)
			}
			if len(expected.State) > 0 {
				fixturemd.AssertBools(t, "state", map[string]bool{
					"runtime_repair_in_flight": l.runtimeRepairInFlight != nil,
					"running":                  l.state == StateRunning,
					"paused":                   l.state == StatePaused,
					"draining":                 l.state == StateDraining,
				}, expected.State)
			}
			if len(expected.Counts) > 0 {
				fixturemd.AssertCounts(t, "counts", map[string]int{
					"spawn_calls":                len(sp.calls),
					"runtime_repair_spawn_calls": len(repairCalls),
					"normal_spawn_calls":         len(normalCalls),
					"created_worktrees":          len(wt.created),
				}, expected.Counts)
			}
			if len(expected.Absence) > 0 {
				fixturemd.AssertBools(t, "absence", map[string]bool{
					"repair_task_scheduled": len(repairCalls) == 0,
					"oops_task_scheduled":   !hasSkill(repairCalls, "oops"),
					"normal_task_scheduled": len(normalCalls) == 0,
				}, expected.Absence)
			}
			if len(expected.Routing) > 0 {
				fixturemd.AssertStrings(t, "routing", map[string]string{
					"first_spawn_name":        firstSpawn.Name,
					"first_spawn_skill":       firstSpawn.Skill,
					"first_spawn_provider":    firstSpawn.Provider,
					"first_spawn_model":       firstSpawn.Model,
					"runtime_repair_name":     firstRepair.Name,
					"runtime_repair_skill":    firstRepair.Skill,
					"runtime_repair_provider": firstRepair.Provider,
					"runtime_repair_model":    firstRepair.Model,
					"runtime_repair_prompt":   firstRepair.Prompt,
				}, expected.Routing)
			}
			if len(expected.Idempotence) > 0 {
				fixturemd.AssertBools(t, "idempotence", map[string]bool{
					"no_new_spawns_on_extra_cycles":                len(sp.calls) == spawnsBeforeExtra,
					"no_duplicate_runtime_repairs_on_extra_cycles": len(repairCalls) == repairsBeforeExtra,
				}, expected.Idempotence)
			}
		})
	}
}

func runtimeRepairCalls(calls []spawner.SpawnRequest) []spawner.SpawnRequest {
	out := make([]spawner.SpawnRequest, 0, len(calls))
	for _, call := range calls {
		if strings.HasPrefix(call.Name, "repair-runtime-") {
			out = append(out, call)
		}
	}
	return out
}

func normalSpawnCalls(calls []spawner.SpawnRequest) []spawner.SpawnRequest {
	out := make([]spawner.SpawnRequest, 0, len(calls))
	for _, call := range calls {
		if !strings.HasPrefix(call.Name, "repair-runtime-") {
			out = append(out, call)
		}
	}
	return out
}

func hasSkill(calls []spawner.SpawnRequest, skill string) bool {
	for _, call := range calls {
		if strings.EqualFold(strings.TrimSpace(call.Skill), strings.TrimSpace(skill)) {
			return true
		}
	}
	return false
}
