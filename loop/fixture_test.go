package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/testutil/fixturedir"
	"github.com/poteto/noodle/mise"
)

var runtimeRepairNamePattern = regexp.MustCompile(`repair-runtime-\d{8}-\d{6}-\d+`)

type loopFixtureSetup struct {
	DispatcherError                  string            `json:"dispatcher_error"`
	WorktreeCreateError           string            `json:"worktree_create_error"`
	WorktreeCreateErrorNames      []string          `json:"worktree_create_error_names"`
	FailedTargets                 map[string]string `json:"failed_targets"`
	Phases                        map[string]string `json:"phases"`
	RecoveryMaxRetries            *int              `json:"recovery_max_retries"`
	RunningRuntimeRepairSessionID string            `json:"running_runtime_repair_session_id"`
}

type loopFixtureMiseRun struct {
	Backlog  []adapter.BacklogItem `json:"backlog"`
	Plans    []mise.PlanSummary    `json:"plans"`
	Warnings []string              `json:"warnings"`
	Error    string                `json:"error"`
}

type loopFixtureStateInput struct {
	MiseResult                 loopFixtureMiseRun `json:"mise_result"`
	RuntimeRepairSessionStatus string             `json:"runtime_repair_session_status"`
	RuntimeRepairSessionIndex  *int               `json:"runtime_repair_session_index"`
}

type loopFixtureRuntimeDump struct {
	States map[string]loopFixtureStateDump `json:"states"`
}

type loopFixtureStateDump struct {
	CycleError             string                `json:"cycle_error,omitempty"`
	Transition             string                `json:"transition"`
	RuntimeRepairInFlight  bool                  `json:"runtime_repair_in_flight"`
	RepairTaskScheduled    bool                  `json:"repair_task_scheduled"`
	OopsTaskScheduled      bool                  `json:"oops_task_scheduled"`
	NormalTaskScheduled    bool                  `json:"normal_task_scheduled"`
	SpawnCalls             int                   `json:"spawn_calls"`
	RuntimeRepairSpawnCall int                   `json:"runtime_repair_spawn_calls"`
	NormalSpawnCalls       int                   `json:"normal_spawn_calls"`
	CreatedWorktrees       int                   `json:"created_worktrees"`
	FirstSpawn             *loopFixtureSpawnDump `json:"first_spawn,omitempty"`
	RuntimeRepairSpawn     *loopFixtureSpawnDump `json:"runtime_repair_spawn,omitempty"`
}

type loopFixtureSpawnDump struct {
	Name     string `json:"name,omitempty"`
	Skill    string `json:"skill,omitempty"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
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

func TestLoopDirectoryFixtures(t *testing.T) {
	inventory := fixturedir.LoadInventory(t, "testdata")
	fixturedir.AssertValidFixtureRoot(t, "testdata")

	mode := strings.ToLower(strings.TrimSpace(os.Getenv("NOODLE_LOOP_FIXTURE_MODE")))
	if mode == "" {
		mode = "check"
	}
	if mode != "check" && mode != "record" {
		t.Fatalf("invalid NOODLE_LOOP_FIXTURE_MODE %q (expected check|record)", mode)
	}

	for _, fixtureCase := range inventory.Cases {
		fixtureCase := fixtureCase
		t.Run(fixtureCase.Name, func(t *testing.T) {
			setup, _ := fixturedir.ParseOptionalStateJSON[loopFixtureSetup](t, fixtureCase.States[0], "setup.json")

			expected := loopFixtureRuntimeDump{}
			if mode == "check" {
				expected = fixturedir.ParseJSON[loopFixtureRuntimeDump](
					t,
					[]byte(fixturedir.MustSection(t, fixtureCase, "Runtime Dump")),
					"runtime dump",
				)
				assertRuntimeDumpCoverage(t, fixtureCase, expected)
			}

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
			sp := &fakeDispatcher{}
			if strings.TrimSpace(setup.DispatcherError) != "" {
				sp.dispatchErr = errors.New(strings.TrimSpace(setup.DispatcherError))
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

			l := New(projectDir, "noodle", baseCfg, Dependencies{
				Dispatcher: sp,
				Worktree:  wt,
				Adapter:   &fakeAdapterRunner{},
				Mise:      &fakeMise{results: miseResults},
				Monitor:   fakeMonitor{},
				Registry:  testLoopRegistry(),
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

			observed := loopFixtureRuntimeDump{
				States: make(map[string]loopFixtureStateDump, len(fixtureCase.States)),
			}
			for index, state := range fixtureCase.States {
				applyStateRuntimeSnapshot(t, state, runtimeDir)
				cfg := cloneConfig(baseCfg)
				if path := strings.TrimSpace(state.ConfigScope.StateOverridePath); path != "" {
					applyConfigOverride(t, &cfg, path)
				}
				l.config = cfg

				applySessionStatusInput(t, sp, stateInputs[index], index)

				err := l.Cycle(context.Background())
				repairCalls := runtimeRepairCalls(sp.calls)
				normalCalls := normalSpawnCalls(sp.calls)

				stateDump := loopFixtureStateDump{
					CycleError:             normalizeDynamicText(errorString(err)),
					Transition:             strings.ToLower(strings.TrimSpace(string(l.state))),
					RuntimeRepairInFlight:  l.runtimeRepairInFlight != nil,
					RepairTaskScheduled:    len(repairCalls) > 0,
					OopsTaskScheduled:      hasSkill(repairCalls, "oops"),
					NormalTaskScheduled:    len(normalCalls) > 0,
					SpawnCalls:             len(sp.calls),
					RuntimeRepairSpawnCall: len(repairCalls),
					NormalSpawnCalls:       len(normalCalls),
					CreatedWorktrees:       len(wt.created),
				}
				if len(normalCalls) > 0 {
					stateDump.FirstSpawn = requestDump(normalCalls[0])
				}
				if len(repairCalls) > 0 {
					stateDump.RuntimeRepairSpawn = requestDump(repairCalls[0])
				}

				observed.States[state.ID] = stateDump
			}

			if mode == "record" {
				if err := writeRuntimeDumpSection(fixtureCase.Layout.ExpectedPath, observed); err != nil {
					t.Fatalf("write runtime dump section for %s: %v", fixtureCase.Layout.ExpectedPath, err)
				}
				return
			}

			if !reflect.DeepEqual(observed, expected) {
				t.Fatalf("runtime dump mismatch\nactual:\n%s\nexpected:\n%s", mustJSON(observed), mustJSON(expected))
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
			brief:    mise.Brief{Backlog: result.Backlog, Plans: result.Plans},
			warnings: result.Warnings,
			err:      resultErr,
		})
	}
	return results
}

func applySessionStatusInput(t *testing.T, sp *fakeDispatcher, input loopFixtureStateInput, defaultIndex int) {
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
	if setup.RecoveryMaxRetries != nil {
		cfg.Recovery.MaxRetries = *setup.RecoveryMaxRetries
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

func assertRuntimeDumpCoverage(t *testing.T, fixtureCase fixturedir.FixtureCase, expected loopFixtureRuntimeDump) {
	t.Helper()
	for _, state := range fixtureCase.States {
		if _, ok := expected.States[state.ID]; !ok {
			t.Fatalf("fixture %s missing runtime dump key for %s", fixtureCase.Name, state.ID)
		}
	}
	for stateID := range expected.States {
		if _, ok := fixtureCase.State(stateID); !ok {
			t.Fatalf("fixture %s has extra runtime dump key %s with no matching state directory", fixtureCase.Name, stateID)
		}
	}
}

func runtimeRepairCalls(calls []dispatcher.DispatchRequest) []dispatcher.DispatchRequest {
	out := make([]dispatcher.DispatchRequest, 0, len(calls))
	for _, call := range calls {
		if strings.HasPrefix(call.Name, "repair-runtime-") {
			out = append(out, call)
		}
	}
	return out
}

func normalSpawnCalls(calls []dispatcher.DispatchRequest) []dispatcher.DispatchRequest {
	out := make([]dispatcher.DispatchRequest, 0, len(calls))
	for _, call := range calls {
		if !strings.HasPrefix(call.Name, "repair-runtime-") {
			out = append(out, call)
		}
	}
	return out
}

func hasSkill(calls []dispatcher.DispatchRequest, skill string) bool {
	for _, call := range calls {
		if strings.EqualFold(strings.TrimSpace(call.Skill), strings.TrimSpace(skill)) {
			return true
		}
	}
	return false
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func requestDump(request dispatcher.DispatchRequest) *loopFixtureSpawnDump {
	dump := &loopFixtureSpawnDump{
		Name:     normalizeSpawnName(request.Name),
		Skill:    strings.TrimSpace(request.Skill),
		Provider: strings.TrimSpace(request.Provider),
		Model:    strings.TrimSpace(request.Model),
	}
	return dump
}

func normalizeSpawnName(name string) string {
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, "repair-runtime-") {
		return "repair-runtime-*"
	}
	return name
}

func normalizeDynamicText(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	return runtimeRepairNamePattern.ReplaceAllString(input, "repair-runtime-*")
}

func mustJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":"%v"}`, err)
	}
	return string(data)
}

func writeRuntimeDumpSection(expectedPath string, dump loopFixtureRuntimeDump) error {
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", expectedPath, err)
	}
	frontmatter, body, err := splitExpectedMarkdown(string(content))
	if err != nil {
		return err
	}
	runtimeJSON := mustJSON(dump)
	expectedErrorJSON, hasExpectedError := extractJSONSection(body, "Expected Error")

	var out strings.Builder
	out.WriteString(strings.TrimRight(frontmatter, "\n"))
	out.WriteString("\n\n## Runtime Dump\n\n```json\n")
	out.WriteString(runtimeJSON)
	out.WriteString("\n```\n")
	if hasExpectedError {
		out.WriteString("\n## Expected Error\n\n```json\n")
		out.WriteString(strings.TrimSpace(expectedErrorJSON))
		out.WriteString("\n```\n")
	}
	return os.WriteFile(expectedPath, []byte(fixturedir.NormalizeFixtureMarkdown(out.String())), 0o644)
}

func splitExpectedMarkdown(content string) (string, string, error) {
	content = fixturedir.NormalizeFixtureMarkdown(content)
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("expected.md must start with frontmatter delimited by ---")
	}
	closingIndex := -1
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			closingIndex = index
			break
		}
	}
	if closingIndex < 0 {
		return "", "", fmt.Errorf("expected.md frontmatter is missing closing --- delimiter")
	}
	frontmatter := strings.Join(lines[:closingIndex+1], "\n")
	body := strings.Join(lines[closingIndex+1:], "\n")
	return frontmatter, body, nil
}

func extractJSONSection(body string, heading string) (string, bool) {
	heading = regexp.QuoteMeta(strings.TrimSpace(heading))
	if heading == "" {
		return "", false
	}
	pattern := regexp.MustCompile(
		`(?ms)^##\s+` + heading + `\s*\n\s*` + "```json" + `\s*\n(.*?)\n` + "```" + `\s*`,
	)
	matches := pattern.FindStringSubmatch(body)
	if len(matches) != 2 {
		return "", false
	}
	return strings.TrimSpace(matches[1]), true
}
