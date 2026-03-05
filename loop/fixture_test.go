package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/testutil/fixturedir"
	"github.com/poteto/noodle/mise"
	loopruntime "github.com/poteto/noodle/runtime"
	"github.com/poteto/noodle/worktree"
)

type loopFixtureSetup struct {
	DispatcherError          string                        `json:"dispatcher_error"`
	WorktreeCreateError      string                        `json:"worktree_create_error"`
	WorktreeCreateErrorNames []string                      `json:"worktree_create_error_names"`
	WorktreeMergeError       string                        `json:"worktree_merge_error"`
	ActiveSessions           []loopFixtureActiveSession    `json:"active_sessions"`
	AdoptedTargets           []loopFixtureAdoptedTarget    `json:"adopted_targets"`
	RecoveredSessions        []loopFixtureRecoveredSession `json:"recovered_sessions"`
	RunStartupReconcile      bool                          `json:"run_startup_reconcile"`
	RecoveryMaxRetries       *int                          `json:"recovery_max_retries"`
	BootstrapAttempts        int                           `json:"bootstrap_attempts"`
	BootstrapExhausted       bool                          `json:"bootstrap_exhausted"`
	CapturePendingReview     bool                          `json:"capture_pending_review"`
	CaptureOrders            bool                          `json:"capture_orders"`
}

type loopFixtureActiveSession struct {
	ID     string `json:"id"`
	Target string `json:"target"`
}

type loopFixtureAdoptedTarget struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
}

type loopFixtureRecoveredSession struct {
	OrderID   string `json:"order_id"`
	SessionID string `json:"session_id"`
}

type loopFixtureMiseRun struct {
	Backlog  []adapter.BacklogItem `json:"backlog"`
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
	ActiveSummaryTotal  int                   `json:"active_summary_total"`
	FirstSpawn          *loopFixtureSpawnDump `json:"first_spawn,omitempty"`
	PendingReview       map[string]string     `json:"pending_review,omitempty"`
	Orders              []loopFixtureOrder    `json:"orders,omitempty"`
}

type loopFixtureOrder struct {
	ID     string                  `json:"id"`
	Status string                  `json:"status,omitempty"`
	Stages []loopFixtureOrderStage `json:"stages,omitempty"`
}

type loopFixtureOrderStage struct {
	TaskKey string `json:"task_key,omitempty"`
	Status  string `json:"status,omitempty"`
}

type loopFixtureSpawnDump struct {
	Name     string `json:"name,omitempty"`
	Skill    string `json:"skill,omitempty"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

type fixtureConfigOverride struct {
	Mode    string `toml:"mode"`
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
			rt := newMockRuntime()
			for _, rs := range setup.RecoveredSessions {
				rt.recovered = append(rt.recovered, loopruntime.RecoveredSession{
					OrderID:       rs.OrderID,
					SessionHandle: &mockSession{id: rs.SessionID, status: "running", done: make(chan struct{})},
					RuntimeName:   "process",
				})
			}
			if strings.TrimSpace(setup.DispatcherError) != "" {
				rt.dispatchErr = errors.New(strings.TrimSpace(setup.DispatcherError))
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
				Runtimes: map[string]loopruntime.Runtime{"process": rt},
				Worktree: wt,
				Adapter:  &fakeAdapterRunner{},
				Mise:     &fakeMise{results: miseResults},
				Monitor:  fakeMonitor{},
				Registry: testLoopRegistry(),
				Now:      time.Now,
			})
			applyFixtureActiveSessions(l, setup.ActiveSessions)
			applyFixtureAdoptedTargets(t, l, setup.AdoptedTargets)
			applyFixtureBootstrap(l, setup)
			if msg := strings.TrimSpace(setup.WorktreeMergeError); msg != "" {
				if strings.HasPrefix(msg, "merge_conflict:") {
					branch := strings.TrimSpace(strings.TrimPrefix(msg, "merge_conflict:"))
					wt.mergeErr = &worktree.MergeConflictError{Branch: branch, Err: errors.New("conflict")}
				} else {
					wt.mergeErr = errors.New(msg)
				}
			}

			observed := loopFixtureRuntimeDump{
				States: make(map[string]loopFixtureStateDump, len(fixtureCase.States)),
			}
			for idx, state := range fixtureCase.States {
				applyStateRuntimeSnapshot(t, state, runtimeDir)
				cfg := cloneConfig(baseCfg)
				if path := strings.TrimSpace(state.ConfigScope.StateOverridePath); path != "" {
					applyConfigOverride(t, &cfg, path)
				}
				l.config = cfg
				if setup.RunStartupReconcile && idx == 0 {
					if err := l.loadOrdersState(); err != nil {
						t.Fatalf("loadOrdersState before reconcile: %v", err)
					}
					if err := l.reconcile(context.Background()); err != nil {
						t.Fatalf("startup reconcile in fixture: %v", err)
					}
				}

				err := l.Cycle(context.Background())

				stateDump := loopFixtureStateDump{
					CycleError:          normalizeDynamicText(errorString(err)),
					Transition:          strings.ToLower(strings.TrimSpace(string(l.state))),
					NormalTaskScheduled: len(rt.calls) > 0,
					SpawnCalls:          len(rt.calls),
					NormalSpawnCalls:    len(rt.calls),
					CreatedWorktrees:    len(wt.created),
					ActiveSummaryTotal:  l.activeSummary.Total,
				}
				if len(rt.calls) > 0 {
					stateDump.FirstSpawn = requestDump(rt.calls[0])
				}
				if setup.CapturePendingReview {
					stateDump.PendingReview = pendingReviewDump(l.cooks.pendingReview)
				}
				if setup.CaptureOrders {
					stateDump.Orders = readOrdersDump(t, l.deps.OrdersFile)
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
			brief:    mise.Brief{Backlog: result.Backlog},
			warnings: result.Warnings,
			err:      resultErr,
		})
	}
	return results
}

func applySetupConfig(cfg *config.Config, setup loopFixtureSetup) {
	_ = cfg
	_ = setup
}

func applyFixtureActiveSessions(l *Loop, sessions []loopFixtureActiveSession) {
	for _, session := range sessions {
		sessionID := strings.TrimSpace(session.ID)
		targetID := strings.TrimSpace(session.Target)
		if sessionID == "" || targetID == "" {
			continue
		}
		cook := &cookHandle{
			cookIdentity: cookIdentity{orderID: targetID},
			session: &mockSession{
				id:     sessionID,
				status: "running",
				done:   make(chan struct{}),
			},
		}
		l.cooks.activeCooksByOrder[targetID] = cook
	}
}

func applyFixtureAdoptedTargets(t *testing.T, l *Loop, targets []loopFixtureAdoptedTarget) {
	t.Helper()
	if len(targets) == 0 {
		return
	}
	for _, target := range targets {
		targetID := strings.TrimSpace(target.ID)
		sessionID := strings.TrimSpace(target.SessionID)
		if targetID == "" || sessionID == "" {
			continue
		}
		l.cooks.adoptedTargets[targetID] = sessionID
		l.cooks.adoptedSessions = append(l.cooks.adoptedSessions, sessionID)

		sessionDir := filepath.Join(l.runtimeDir, "sessions", sessionID)
		if err := os.MkdirAll(sessionDir, 0o755); err != nil {
			t.Fatalf("mkdir adopted session dir %s: %v", sessionDir, err)
		}
		if err := os.WriteFile(filepath.Join(sessionDir, "meta.json"), []byte(`{"status":"running"}`), 0o644); err != nil {
			t.Fatalf("write adopted session meta %s: %v", sessionDir, err)
		}
		// Write process.json so PID-based liveness check in refreshAdoptedTargets works.
		procMeta, _ := json.Marshal(map[string]any{
			"pid":        os.Getpid(),
			"session_id": sessionID,
			"started_at": time.Now().UTC().Format(time.RFC3339),
		})
		if err := os.WriteFile(filepath.Join(sessionDir, "process.json"), procMeta, 0o644); err != nil {
			t.Fatalf("write adopted session process.json %s: %v", sessionDir, err)
		}
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
	if provider := strings.TrimSpace(override.Routing.Defaults.Provider); provider != "" {
		cfg.Routing.Defaults.Provider = provider
	}
	if model := strings.TrimSpace(override.Routing.Defaults.Model); model != "" {
		cfg.Routing.Defaults.Model = model
	}
	if mode := strings.TrimSpace(override.Mode); mode != "" {
		cfg.Mode = mode
	}
}

func cloneConfig(in config.Config) config.Config {
	return in
}

func applyStateRuntimeSnapshot(t *testing.T, state fixturedir.FixtureState, runtimeDir string) {
	t.Helper()
	fixturedir.ApplyRuntimeSnapshot(t, state, runtimeDir)
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

func requestDump(request loopruntime.DispatchRequest) *loopFixtureSpawnDump {
	dump := &loopFixtureSpawnDump{
		Name:     strings.TrimSpace(request.Name),
		Skill:    strings.TrimSpace(request.Skill),
		Provider: strings.TrimSpace(request.Provider),
		Model:    strings.TrimSpace(request.Model),
	}
	return dump
}

func pendingReviewDump(pending map[string]*pendingReviewCook) map[string]string {
	if len(pending) == 0 {
		return nil
	}
	out := make(map[string]string, len(pending))
	for orderID, item := range pending {
		if item == nil {
			continue
		}
		out[strings.TrimSpace(orderID)] = strings.TrimSpace(item.reason)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func readOrdersDump(t *testing.T, ordersPath string) []loopFixtureOrder {
	t.Helper()
	orders, err := readOrders(ordersPath)
	if err != nil {
		t.Fatalf("read orders dump %s: %v", ordersPath, err)
	}
	if len(orders.Orders) == 0 {
		return nil
	}
	out := make([]loopFixtureOrder, 0, len(orders.Orders))
	for _, order := range orders.Orders {
		dump := loopFixtureOrder{
			ID:     strings.TrimSpace(order.ID),
			Status: strings.TrimSpace(string(order.Status)),
		}
		if len(order.Stages) > 0 {
			dump.Stages = make([]loopFixtureOrderStage, 0, len(order.Stages))
			for _, stage := range order.Stages {
				dump.Stages = append(dump.Stages, loopFixtureOrderStage{
					TaskKey: strings.TrimSpace(stage.TaskKey),
					Status:  strings.TrimSpace(string(stage.Status)),
				})
			}
		}
		out = append(out, dump)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
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
	return fixturedir.WriteSectionToExpected(expectedPath, "Runtime Dump", dump)
}

func applyFixtureBootstrap(l *Loop, setup loopFixtureSetup) {
	if setup.BootstrapAttempts > 0 {
		l.bootstrapAttempts = setup.BootstrapAttempts
	}
	if setup.BootstrapExhausted {
		l.bootstrapExhausted = true
	}
}
