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
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/testutil/fixturedir"
	"github.com/poteto/noodle/mise"
	loopruntime "github.com/poteto/noodle/runtime"
)

type loopFixtureSetup struct {
	DispatcherError          string                     `json:"dispatcher_error"`
	WorktreeCreateError      string                     `json:"worktree_create_error"`
	WorktreeCreateErrorNames []string                   `json:"worktree_create_error_names"`
	FailedTargets            map[string]string          `json:"failed_targets"`
	ActiveSessions           []loopFixtureActiveSession `json:"active_sessions"`
	AdoptedTargets           []loopFixtureAdoptedTarget `json:"adopted_targets"`
	RecoveryMaxRetries       *int                       `json:"recovery_max_retries"`
}

type loopFixtureActiveSession struct {
	ID     string `json:"id"`
	Target string `json:"target"`
}

type loopFixtureAdoptedTarget struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
}

type loopFixtureMiseRun struct {
	Backlog  []adapter.BacklogItem `json:"backlog"`
	Plans    []mise.PlanSummary    `json:"plans"`
	Warnings []string              `json:"warnings"`
	Error    string                `json:"error"`
}

type loopFixtureStateInput struct {
	MiseResult loopFixtureMiseRun `json:"mise_result"`
}

type loopFixtureRuntimeDump struct {
	States map[string]loopFixtureStateDump `json:"states"`
}

type loopFixtureStateDump struct {
	CycleError          string                `json:"cycle_error,omitempty"`
	Transition          string                `json:"transition"`
	NormalTaskScheduled bool                  `json:"normal_task_scheduled"`
	SpawnCalls          int                   `json:"spawn_calls"`
	NormalSpawnCalls    int                   `json:"normal_spawn_calls"`
	CreatedWorktrees    int                   `json:"created_worktrees"`
	FirstSpawn          *loopFixtureSpawnDump `json:"first_spawn,omitempty"`
}

type loopFixtureSpawnDump struct {
	Name     string `json:"name,omitempty"`
	Skill    string `json:"skill,omitempty"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

type fixtureConfigOverride struct {
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
				Worktree:   wt,
				Adapter:    &fakeAdapterRunner{},
				Mise:       &fakeMise{results: miseResults},
				Monitor:    fakeMonitor{},
				Registry:   testLoopRegistry(),
				Now:        time.Now,
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
			applyFixtureActiveSessions(l, setup.ActiveSessions)
			applyFixtureAdoptedTargets(t, l, setup.AdoptedTargets)

			observed := loopFixtureRuntimeDump{
				States: make(map[string]loopFixtureStateDump, len(fixtureCase.States)),
			}
			for _, state := range fixtureCase.States {
				applyStateRuntimeSnapshot(t, state, runtimeDir)
					cfg := cloneConfig(baseCfg)
				if path := strings.TrimSpace(state.ConfigScope.StateOverridePath); path != "" {
					applyConfigOverride(t, &cfg, path)
				}
				l.config = cfg

				err := l.Cycle(context.Background())

				stateDump := loopFixtureStateDump{
					CycleError:          normalizeDynamicText(errorString(err)),
					Transition:          strings.ToLower(strings.TrimSpace(string(l.state))),
					NormalTaskScheduled: len(sp.calls) > 0,
					SpawnCalls:          len(sp.calls),
					NormalSpawnCalls:    len(sp.calls),
					CreatedWorktrees:    len(wt.created),
				}
				if len(sp.calls) > 0 {
					stateDump.FirstSpawn = requestDump(sp.calls[0])
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

func applySetupConfig(cfg *config.Config, setup loopFixtureSetup) {
	if setup.RecoveryMaxRetries != nil {
		cfg.Recovery.MaxRetries = *setup.RecoveryMaxRetries
	}
}

func applyFixtureActiveSessions(l *Loop, sessions []loopFixtureActiveSession) {
	for _, session := range sessions {
		sessionID := strings.TrimSpace(session.ID)
		targetID := strings.TrimSpace(session.Target)
		if sessionID == "" || targetID == "" {
			continue
		}
		cook := &cookHandle{
			orderID: targetID,
			session: &fakeSession{
				id:     sessionID,
				status: "running",
				done:   make(chan struct{}),
			},
		}
		l.activeCooksByOrder[targetID] = cook
	}
}

func applyFixtureAdoptedTargets(t *testing.T, l *Loop, targets []loopFixtureAdoptedTarget) {
	t.Helper()
	if len(targets) == 0 {
		return
	}
	sessionNames := make([]string, 0, len(targets))
	for _, target := range targets {
		targetID := strings.TrimSpace(target.ID)
		sessionID := strings.TrimSpace(target.SessionID)
		if targetID == "" || sessionID == "" {
			continue
		}
		l.adoptedTargets[targetID] = sessionID
		l.adoptedSessions = append(l.adoptedSessions, sessionID)
		sessionNames = append(sessionNames, loopruntime.TmuxSessionName(sessionID))

		sessionDir := filepath.Join(l.runtimeDir, "sessions", sessionID)
		if err := os.MkdirAll(sessionDir, 0o755); err != nil {
			t.Fatalf("mkdir adopted session dir %s: %v", sessionDir, err)
		}
		if err := os.WriteFile(filepath.Join(sessionDir, "meta.json"), []byte(`{"status":"running"}`), 0o644); err != nil {
			t.Fatalf("write adopted session meta %s: %v", sessionDir, err)
		}
	}
	if len(sessionNames) == 0 {
		return
	}
	installFixtureTmuxStub(t, sessionNames)
}

func installFixtureTmuxStub(t *testing.T, sessionNames []string) {
	t.Helper()
	binDir := t.TempDir()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir tmux stub dir %s: %v", binDir, err)
	}

	tmuxPath := filepath.Join(binDir, "tmux")
	content := fixtureTmuxScript(sessionNames)
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		tmuxPath = filepath.Join(binDir, "tmux.bat")
		content = fixtureTmuxBatchScript(sessionNames)
		mode = 0o644
	}
	if err := os.WriteFile(tmuxPath, []byte(content), mode); err != nil {
		t.Fatalf("write tmux stub %s: %v", tmuxPath, err)
	}
	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)
}

func fixtureTmuxScript(sessionNames []string) string {
	var body strings.Builder
	body.WriteString("#!/bin/sh\n")
	body.WriteString("if [ \"$1\" = \"list-sessions\" ]; then\n")
	for _, name := range sessionNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		body.WriteString("  printf '%s\\n' '")
		body.WriteString(strings.ReplaceAll(trimmed, "'", "'\\''"))
		body.WriteString("'\n")
	}
	body.WriteString("  exit 0\n")
	body.WriteString("fi\n")
	body.WriteString("if [ \"$1\" = \"kill-session\" ]; then\n")
	body.WriteString("  exit 0\n")
	body.WriteString("fi\n")
	body.WriteString("exit 0\n")
	return body.String()
}

func fixtureTmuxBatchScript(sessionNames []string) string {
	var body strings.Builder
	body.WriteString("@echo off\r\n")
	body.WriteString("if \"%1\"==\"list-sessions\" (\r\n")
	for _, name := range sessionNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		body.WriteString("  echo ")
		body.WriteString(trimmed)
		body.WriteString("\r\n")
	}
	body.WriteString("  exit /b 0\r\n")
	body.WriteString(")\r\n")
	body.WriteString("if \"%1\"==\"kill-session\" exit /b 0\r\n")
	body.WriteString("exit /b 0\r\n")
	return body.String()
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
	if provider := strings.TrimSpace(override.Routing.Defaults.Provider); provider != "" {
		cfg.Routing.Defaults.Provider = provider
	}
	if model := strings.TrimSpace(override.Routing.Defaults.Model); model != "" {
		cfg.Routing.Defaults.Model = model
	}
}

func cloneConfig(in config.Config) config.Config {
	return in
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

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func requestDump(request dispatcher.DispatchRequest) *loopFixtureSpawnDump {
	dump := &loopFixtureSpawnDump{
		Name:     strings.TrimSpace(request.Name),
		Skill:    strings.TrimSpace(request.Skill),
		Provider: strings.TrimSpace(request.Provider),
		Model:    strings.TrimSpace(request.Model),
	}
	return dump
}

func normalizeDynamicText(input string) string {
	return strings.TrimSpace(input)
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
