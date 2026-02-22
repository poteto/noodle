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

	"github.com/BurntSushi/toml"
	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/testutil/fixturedir"
	"github.com/poteto/noodle/mise"
	"github.com/poteto/noodle/spawner"
)

type loopFixtureSetup struct {
	SpawnerError                  string            `json:"spawner_error"`
	WorktreeCreateError           string            `json:"worktree_create_error"`
	WorktreeCreateErrorNames      []string          `json:"worktree_create_error_names"`
	FailedTargets                 map[string]string `json:"failed_targets"`
	Phases                        map[string]string `json:"phases"`
	RunningRuntimeRepairSessionID string            `json:"running_runtime_repair_session_id"`
}

type loopFixtureMiseRun struct {
	Backlog  []adapter.BacklogItem `json:"backlog"`
	Warnings []string              `json:"warnings"`
	Error    string                `json:"error"`
}

type loopFixtureStateInput struct {
	MiseResult                 loopFixtureMiseRun `json:"mise_result"`
	RuntimeRepairSessionStatus string             `json:"runtime_repair_session_status"`
	RuntimeRepairSessionIndex  *int               `json:"runtime_repair_session_index"`
}

type loopFixtureStateExpectation struct {
	Error       *fixturedir.ErrorExpectation            `json:"error"`
	Transition  string                                  `json:"transition"`
	Actions     map[string]bool                         `json:"actions"`
	State       map[string]bool                         `json:"state"`
	Counts      map[string]fixturedir.CountExpectation  `json:"counts"`
	Absence     map[string]bool                         `json:"absence"`
	Routing     map[string]fixturedir.StringExpectation `json:"routing"`
	Idempotence map[string]bool                         `json:"idempotence"`
}

type loopFixtureExpected struct {
	States map[string]loopFixtureStateExpectation `json:"states"`
}

type fixtureConfigOverride struct {
	Phases  map[string]string `toml:"phases"`
	Routing struct {
		Defaults struct {
			Provider string `toml:"provider"`
			Model    string `toml:"model"`
		} `toml:"defaults"`
	} `toml:"routing"`
}

type loopObservedCounts struct {
	SpawnCalls              int
	RuntimeRepairSpawnCalls int
}

func TestLoopDirectoryFixtures(t *testing.T) {
	inventory := fixturedir.LoadInventory(t, "testdata")
	fixturedir.AssertValidFixtureRoot(t, "testdata")

	for _, fixtureCase := range inventory.Cases {
		fixtureCase := fixtureCase
		t.Run(fixtureCase.Name, func(t *testing.T) {
			setup, _ := fixturedir.ParseOptionalStateJSON[loopFixtureSetup](t, fixtureCase.States[0], "setup.json")
			expected := fixturedir.ParseJSON[loopFixtureExpected](
				t,
				[]byte(fixturedir.MustSection(t, fixtureCase, "Expected")),
				"expected",
			)
			assertStateExpectationCoverage(t, fixtureCase, expected)

			stateInputs := make([]loopFixtureStateInput, 0, len(fixtureCase.States))
			for _, state := range fixtureCase.States {
				input, _ := fixturedir.ParseOptionalStateJSON[loopFixtureStateInput](t, state, "input.json")
				stateInputs = append(stateInputs, input)
			}

			projectDir := t.TempDir()
			runtimeDir := filepath.Join(projectDir, ".noodle")
			if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
				t.Fatalf("mkdir runtime: %v", err)
			}
			queuePath := filepath.Join(runtimeDir, "queue.json")

			miseResults := buildMiseResults(stateInputs)
			sp := &fakeSpawner{}
			if strings.TrimSpace(setup.SpawnerError) != "" {
				sp.spawnErr = errors.New(strings.TrimSpace(setup.SpawnerError))
			}
			wt := &fakeWorktree{}
			if strings.TrimSpace(setup.WorktreeCreateError) != "" {
				createErr := errors.New(strings.TrimSpace(setup.WorktreeCreateError))
				if len(setup.WorktreeCreateErrorNames) == 0 {
					wt.createErr = createErr
				} else {
					wt.createErrByName = make(map[string]error, len(setup.WorktreeCreateErrorNames))
					for _, name := range setup.WorktreeCreateErrorNames {
						name = strings.TrimSpace(name)
						if name == "" {
							continue
						}
						wt.createErrByName[name] = createErr
					}
				}
			}
			baseCfg := config.DefaultConfig()
			applySetupConfig(&baseCfg, setup)
			if path := strings.TrimSpace(fixtureCase.Layout.BaseConfigPath); path != "" {
				applyConfigOverride(t, &baseCfg, path)
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
			if len(setup.FailedTargets) > 0 {
				l.failedTargets = make(map[string]string, len(setup.FailedTargets))
				for id, reason := range setup.FailedTargets {
					id = strings.TrimSpace(id)
					if id == "" {
						continue
					}
					l.failedTargets[id] = strings.TrimSpace(reason)
				}
			}

			prevCounts := loopObservedCounts{}
			for index, state := range fixtureCase.States {
				applyStateRuntimeSnapshot(t, state, runtimeDir)
				cfg := cloneConfig(baseCfg)
				if path := strings.TrimSpace(state.ConfigScope.StateOverridePath); path != "" {
					applyConfigOverride(t, &cfg, path)
				}
				l.config = cfg

				applySessionStatusInput(t, sp, stateInputs[index], index)

				err := l.Cycle(context.Background())
				expectation := expected.States[state.ID]
				fixturedir.AssertError(t, fmt.Sprintf("%s cycle", state.ID), err, expectation.Error)

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

				if value := strings.TrimSpace(expectation.Transition); value != "" {
					actual := strings.ToLower(strings.TrimSpace(string(l.state)))
					if actual != strings.ToLower(value) {
						t.Fatalf("transition for %s = %q, want %q", state.ID, actual, strings.ToLower(value))
					}
				}
				if len(expectation.Actions) > 0 {
					fixturedir.AssertBools(t, "actions", map[string]bool{
						"repair_task_scheduled": len(repairCalls) > 0,
						"oops_task_scheduled":   hasSkill(repairCalls, "oops"),
						"normal_task_scheduled": len(normalCalls) > 0,
					}, expectation.Actions)
				}
				if len(expectation.State) > 0 {
					fixturedir.AssertBools(t, "state", map[string]bool{
						"runtime_repair_in_flight": l.runtimeRepairInFlight != nil,
						"running":                  l.state == StateRunning,
						"paused":                   l.state == StatePaused,
						"draining":                 l.state == StateDraining,
					}, expectation.State)
				}
				if len(expectation.Counts) > 0 {
					fixturedir.AssertCounts(t, "counts", map[string]int{
						"spawn_calls":                len(sp.calls),
						"runtime_repair_spawn_calls": len(repairCalls),
						"normal_spawn_calls":         len(normalCalls),
						"created_worktrees":          len(wt.created),
					}, expectation.Counts)
				}
				if len(expectation.Absence) > 0 {
					fixturedir.AssertBools(t, "absence", map[string]bool{
						"repair_task_scheduled": len(repairCalls) == 0,
						"oops_task_scheduled":   !hasSkill(repairCalls, "oops"),
						"normal_task_scheduled": len(normalCalls) == 0,
					}, expectation.Absence)
				}
				if len(expectation.Routing) > 0 {
					fixturedir.AssertStrings(t, "routing", map[string]string{
						"first_spawn_name":        firstSpawn.Name,
						"first_spawn_skill":       firstSpawn.Skill,
						"first_spawn_provider":    firstSpawn.Provider,
						"first_spawn_model":       firstSpawn.Model,
						"runtime_repair_name":     firstRepair.Name,
						"runtime_repair_skill":    firstRepair.Skill,
						"runtime_repair_provider": firstRepair.Provider,
						"runtime_repair_model":    firstRepair.Model,
						"runtime_repair_prompt":   firstRepair.Prompt,
					}, expectation.Routing)
				}
				if len(expectation.Idempotence) > 0 {
					fixturedir.AssertBools(t, "idempotence", map[string]bool{
						"no_new_spawns_on_extra_cycles":                len(sp.calls) == prevCounts.SpawnCalls,
						"no_duplicate_runtime_repairs_on_extra_cycles": len(repairCalls) == prevCounts.RuntimeRepairSpawnCalls,
					}, expectation.Idempotence)
				}

				prevCounts = loopObservedCounts{
					SpawnCalls:              len(sp.calls),
					RuntimeRepairSpawnCalls: len(repairCalls),
				}
			}
		})
	}
}

func buildMiseResults(stateInputs []loopFixtureStateInput) []fakeMiseResult {
	results := make([]fakeMiseResult, 0, len(stateInputs))
	for _, input := range stateInputs {
		result := input.MiseResult
		var resultErr error
		if strings.TrimSpace(result.Error) != "" {
			resultErr = errors.New(strings.TrimSpace(result.Error))
		}
		results = append(results, fakeMiseResult{
			brief:    mise.Brief{Backlog: result.Backlog},
			warnings: result.Warnings,
			err:      resultErr,
		})
	}
	return results
}

func applySessionStatusInput(t *testing.T, sp *fakeSpawner, input loopFixtureStateInput, defaultIndex int) {
	t.Helper()
	status := strings.TrimSpace(input.RuntimeRepairSessionStatus)
	if status == "" {
		return
	}
	sessionIndex := defaultIndex - 1
	if sessionIndex < 0 {
		sessionIndex = 0
	}
	if input.RuntimeRepairSessionIndex != nil {
		sessionIndex = *input.RuntimeRepairSessionIndex
	}
	if sessionIndex < 0 || sessionIndex >= len(sp.sessions) {
		t.Fatalf("missing runtime repair session at index %d for status %q", sessionIndex, status)
	}
	sp.sessions[sessionIndex].status = status
	select {
	case <-sp.sessions[sessionIndex].done:
	default:
		close(sp.sessions[sessionIndex].done)
	}
}

func applySetupConfig(cfg *config.Config, setup loopFixtureSetup) {
	if cfg.Phases == nil {
		cfg.Phases = map[string]string{}
	}
	for key, value := range setup.Phases {
		cfg.Phases[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
}

func applyConfigOverride(t *testing.T, cfg *config.Config, overridePath string) {
	t.Helper()
	data, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatalf("read config override %s: %v", overridePath, err)
	}
	var override fixtureConfigOverride
	if _, err := toml.Decode(string(data), &override); err != nil {
		t.Fatalf("parse config override %s: %v", overridePath, err)
	}
	if cfg.Phases == nil {
		cfg.Phases = map[string]string{}
	}
	for key, value := range override.Phases {
		cfg.Phases[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if provider := strings.TrimSpace(override.Routing.Defaults.Provider); provider != "" {
		cfg.Routing.Defaults.Provider = provider
	}
	if model := strings.TrimSpace(override.Routing.Defaults.Model); model != "" {
		cfg.Routing.Defaults.Model = model
	}
}

func cloneConfig(in config.Config) config.Config {
	out := in
	if in.Phases != nil {
		out.Phases = make(map[string]string, len(in.Phases))
		for key, value := range in.Phases {
			out.Phases[key] = value
		}
	}
	return out
}

func applyStateRuntimeSnapshot(t *testing.T, state fixturedir.FixtureState, runtimeDir string) {
	t.Helper()
	for _, relPath := range state.FileOrder {
		normalized := filepath.ToSlash(strings.TrimSpace(relPath))
		if !strings.HasPrefix(normalized, ".noodle/") {
			continue
		}
		destRel := strings.TrimPrefix(normalized, ".noodle/")
		destination := filepath.Join(runtimeDir, filepath.FromSlash(destRel))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatalf("mkdir snapshot parent %s: %v", destination, err)
		}
		if err := os.WriteFile(destination, state.MustReadFile(t, relPath), 0o644); err != nil {
			t.Fatalf("write runtime snapshot file %s: %v", destination, err)
		}
	}
}

func assertStateExpectationCoverage(t *testing.T, fixtureCase fixturedir.FixtureCase, expected loopFixtureExpected) {
	t.Helper()
	for _, state := range fixtureCase.States {
		if _, ok := expected.States[state.ID]; !ok {
			t.Fatalf("fixture %s missing expected.states key for %s", fixtureCase.Name, state.ID)
		}
	}
	for stateID := range expected.States {
		if _, ok := fixtureCase.State(stateID); !ok {
			t.Fatalf("fixture %s has extra expected.states key %s with no matching state directory", fixtureCase.Name, stateID)
		}
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
