package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigValues(t *testing.T) {
	config := DefaultConfig()

	if config.Phases["oops"] != "oops" {
		t.Fatalf("phases.oops default = %q", config.Phases["oops"])
	}
	if config.Phases["debugging"] != "debugging" {
		t.Fatalf("phases.debugging default = %q", config.Phases["debugging"])
	}
	if got := strings.Join(config.Skills.Paths, ","); got != ".agents/skills" {
		t.Fatalf("skills.paths default = %q", got)
	}
	if config.Prioritize.Run != "after-each" {
		t.Fatalf("prioritize.run default = %q", config.Prioritize.Run)
	}
	if config.Prioritize.Skill != "prioritize" {
		t.Fatalf("prioritize.skill default = %q", config.Prioritize.Skill)
	}
	if config.Prioritize.Model != "claude-sonnet" {
		t.Fatalf("prioritize.model default = %q", config.Prioritize.Model)
	}
	if config.Routing.Defaults.Provider != "claude" {
		t.Fatalf("routing.defaults.provider default = %q", config.Routing.Defaults.Provider)
	}
	if config.Routing.Defaults.Model != "claude-opus-4-6" {
		t.Fatalf("routing.defaults.model default = %q", config.Routing.Defaults.Model)
	}
	if config.Autonomy != "auto" {
		t.Fatalf("autonomy default = %q, want auto", config.Autonomy)
	}
	if config.Recovery.MaxRetries != 3 {
		t.Fatalf("recovery.max_retries default = %d", config.Recovery.MaxRetries)
	}
	if config.Monitor.StuckThreshold != "120s" {
		t.Fatalf("monitor.stuck_threshold default = %q", config.Monitor.StuckThreshold)
	}
	if config.Monitor.TicketStale != "30m" {
		t.Fatalf("monitor.ticket_stale default = %q", config.Monitor.TicketStale)
	}
	if config.Monitor.PollInterval != "5s" {
		t.Fatalf("monitor.poll_interval default = %q", config.Monitor.PollInterval)
	}
	if config.Concurrency.MaxCooks != 4 {
		t.Fatalf("concurrency.max_cooks default = %d", config.Concurrency.MaxCooks)
	}
	if config.Agents.Claude.Path != "" || config.Agents.Codex.Path != "" {
		t.Fatalf("agent path defaults should be empty: %#v", config.Agents)
	}
	if config.Plans.OnDone != "keep" {
		t.Fatalf("plans.on_done default = %q, want keep", config.Plans.OnDone)
	}
	if config.Runtime.Default != "tmux" {
		t.Fatalf("runtime.default = %q, want tmux", config.Runtime.Default)
	}

	backlog, ok := config.Adapters["backlog"]
	if !ok {
		t.Fatal("default backlog adapter missing")
	}
	if backlog.Scripts["sync"] != ".noodle/adapters/backlog-sync" {
		t.Fatalf("backlog sync default = %q", backlog.Scripts["sync"])
	}

}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noodle.toml")

	config, _, err := Load(path)
	if err != nil {
		t.Fatalf("Load missing config: %v", err)
	}
	if config.Prioritize.Run != "after-each" {
		t.Fatalf("expected default prioritize run, got %q", config.Prioritize.Run)
	}
	if config.Prioritize.Skill != "prioritize" {
		t.Fatalf("expected default prioritize skill, got %q", config.Prioritize.Skill)
	}
	if _, ok := config.Adapters["backlog"]; !ok {
		t.Fatal("expected default backlog adapter when config file is missing")
	}
}

func TestParseConfigRoundTrip(t *testing.T) {
	tomlPayload := `
[phases]
oops = "custom-oops"
debugging = "custom-debug"

[prioritize]
skill = "priority-chef"
run = "manual"
model = "claude-sonnet"

[routing.defaults]
provider = "codex"
model = "gpt-5.3-codex"

[routing.tags.frontend]
provider = "claude"
model = "opus"

[skills]
paths = [".agents/skills"]

[recovery]
max_retries = 5

[monitor]
stuck_threshold = "30s"
ticket_stale = "10m"
poll_interval = "3s"

[concurrency]
max_cooks = 2

[adapters.backlog]
skill = "my-backlog"

[adapters.backlog.scripts]
sync = "gh issue list"
add = "gh issue create"
done = "gh issue close"
edit = "gh issue edit"

[plans]
on_done = "remove"
`

	config, err := Parse([]byte(tomlPayload))
	if err != nil {
		t.Fatalf("Parse config: %v", err)
	}

	if config.Phases["oops"] != "custom-oops" {
		t.Fatalf("oops phase = %q", config.Phases["oops"])
	}
	if config.Prioritize.Run != "manual" {
		t.Fatalf("prioritize.run = %q", config.Prioritize.Run)
	}
	if config.Prioritize.Skill != "priority-chef" {
		t.Fatalf("prioritize.skill = %q", config.Prioritize.Skill)
	}
	if config.Routing.Defaults.Provider != "codex" {
		t.Fatalf("routing.defaults.provider = %q", config.Routing.Defaults.Provider)
	}
	if config.Routing.Tags["frontend"].Model != "opus" {
		t.Fatalf("routing.tags.frontend.model = %q", config.Routing.Tags["frontend"].Model)
	}
	if config.Autonomy != "auto" {
		t.Fatalf("expected default autonomy=auto, got %q", config.Autonomy)
	}
	if config.Recovery.MaxRetries != 5 {
		t.Fatalf("recovery.max_retries = %d", config.Recovery.MaxRetries)
	}
	if config.Concurrency.MaxCooks != 2 {
		t.Fatalf("concurrency.max_cooks = %d", config.Concurrency.MaxCooks)
	}
	if config.Plans.OnDone != "remove" {
		t.Fatalf("plans.on_done = %q, want remove", config.Plans.OnDone)
	}
}

func TestParseMissingOptionalUsesDefaults(t *testing.T) {
	config, err := Parse([]byte(`
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"
`))
	if err != nil {
		t.Fatalf("Parse minimal config: %v", err)
	}

	if config.Prioritize.Run != "after-each" {
		t.Fatalf("expected default prioritize.run, got %q", config.Prioritize.Run)
	}
	if config.Prioritize.Skill != "prioritize" {
		t.Fatalf("expected default prioritize.skill, got %q", config.Prioritize.Skill)
	}
	if config.Autonomy != "auto" {
		t.Fatalf("expected default autonomy=auto, got %q", config.Autonomy)
	}
	if config.Plans.OnDone != "keep" {
		t.Fatalf("plans.on_done default = %q, want keep", config.Plans.OnDone)
	}
	if config.Adapters != nil {
		t.Fatal("adapters should remain unset when omitted from an existing config file")
	}
}

func TestParseInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr string
	}{
		{
			name: "missing provider",
			payload: `
[routing.defaults]
provider = ""
model = "x"
`,
			wantErr: "routing.defaults.provider",
		},
		{
			name: "invalid run frequency",
			payload: `
[routing.defaults]
provider = "claude"
model = "x"

[prioritize]
run = "sometimes"
`,
			wantErr: "prioritize.run",
		},
		{
			name: "empty prioritize skill",
			payload: `
[routing.defaults]
provider = "claude"
model = "x"

[prioritize]
skill = ""
`,
			wantErr: "prioritize.skill",
		},
		{
			name: "invalid duration",
			payload: `
[routing.defaults]
provider = "claude"
model = "x"

[monitor]
stuck_threshold = "not-a-duration"
`,
			wantErr: "monitor.stuck_threshold",
		},
		{
			name: "invalid max cooks",
			payload: `
[routing.defaults]
provider = "claude"
model = "x"

[concurrency]
max_cooks = 0
`,
			wantErr: "concurrency.max_cooks",
		},
		{
			name: "missing tag provider",
			payload: `
[routing.defaults]
provider = "claude"
model = "x"

[routing.tags.frontend]
provider = ""
model = "y"
`,
			wantErr: "routing.tags.frontend.provider",
		},
		{
			name: "invalid on_done value",
			payload: `
[routing.defaults]
provider = "claude"
model = "x"

[plans]
on_done = "bad"
`,
			wantErr: "plans.on_done",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.payload))
			if err == nil {
				t.Fatal("expected parse error")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error %q missing %q", err, test.wantErr)
			}
		})
	}
}

func TestValidationClassification(t *testing.T) {
	oldLookPath := lookPath
	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	oldStatPath := statPath
	statPath = func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	t.Cleanup(func() { lookPath = oldLookPath })
	t.Cleanup(func() { statPath = oldStatPath })

	config := DefaultConfig()
	config.Adapters["backlog"] = AdapterConfig{
		Skill: "backlog",
		Scripts: map[string]string{
			"sync": "/definitely/missing/backlog-sync",
		},
	}
	config.Agents.Claude.Path = "/definitely/missing/claude"

	result := Validate(config)
	if result.CanSpawn() {
		t.Fatal("CanSpawn should be false with fatal diagnostics")
	}
	if len(result.Repairables()) == 0 {
		t.Fatal("expected at least one repairable diagnostic")
	}
	if len(result.Fatals()) == 0 {
		t.Fatal("expected at least one fatal diagnostic")
	}

	foundRepairable := false
	for _, diagnostic := range result.Repairables() {
		if diagnostic.FieldPath == "adapters.backlog.scripts.sync" {
			foundRepairable = true
			break
		}
	}
	if !foundRepairable {
		t.Fatal("missing expected repairable script diagnostic")
	}

	foundFatal := false
	for _, diagnostic := range result.Fatals() {
		if diagnostic.FieldPath == "agents.claude.path" {
			foundFatal = true
			if diagnostic.Fix == "" {
				t.Fatal("fatal diagnostic should include fix instructions")
			}
			if diagnostic.Message == "" {
				t.Fatal("fatal diagnostic should include a message")
			}
		}
	}
	if !foundFatal {
		t.Fatal("missing expected fatal agents.claude.path diagnostic")
	}
}

func TestValidationRepairablesOnlyCanSpawn(t *testing.T) {
	oldLookPath := lookPath
	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	oldStatPath := statPath
	statPath = func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	t.Cleanup(func() { lookPath = oldLookPath })
	t.Cleanup(func() { statPath = oldStatPath })

	config := DefaultConfig()
	config.Agents.Claude.Path = ""
	config.Agents.Codex.Path = ""

	result := Validate(config)
	if !result.CanSpawn() {
		t.Fatal("CanSpawn should be true when only repairable diagnostics are present")
	}
	if len(result.Repairables()) == 0 {
		t.Fatal("expected repairable diagnostics from missing adapter scripts")
	}
	if len(result.Fatals()) != 0 {
		t.Fatalf("expected no fatal diagnostics, got %d", len(result.Fatals()))
	}
}

func TestParseAdapterDefaultsForPartialAdapter(t *testing.T) {
	config, err := Parse([]byte(`
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[adapters.backlog]
skill = "custom-backlog"
`))
	if err != nil {
		t.Fatalf("Parse config: %v", err)
	}

	backlog, ok := config.Adapters["backlog"]
	if !ok {
		t.Fatal("expected backlog adapter to exist")
	}
	if backlog.Skill != "custom-backlog" {
		t.Fatalf("backlog skill = %q", backlog.Skill)
	}
	if backlog.Scripts["sync"] != ".noodle/adapters/backlog-sync" {
		t.Fatalf("backlog sync default = %q", backlog.Scripts["sync"])
	}
	if backlog.Scripts["edit"] != ".noodle/adapters/backlog-edit" {
		t.Fatalf("backlog edit default = %q", backlog.Scripts["edit"])
	}
}

func TestValidateAdapterScriptCommandVsPathChecks(t *testing.T) {
	oldLookPath := lookPath
	lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	oldStatPath := statPath
	statPath = func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	t.Cleanup(func() { lookPath = oldLookPath })
	t.Cleanup(func() { statPath = oldStatPath })

	config := DefaultConfig()
	config.Adapters = map[string]AdapterConfig{
		"backlog": {
			Skill: "backlog",
			Scripts: map[string]string{
				"sync": "gh issue list --json number,title",
				"done": "./missing-script",
			},
		},
	}

	result := Validate(config)
	if len(result.Fatals()) != 0 {
		t.Fatalf("expected no fatal diagnostics, got %d", len(result.Fatals()))
	}

	foundPathDiagnostic := false
	foundCommandDiagnostic := false
	for _, diagnostic := range result.Repairables() {
		if diagnostic.FieldPath == "adapters.backlog.scripts.done" {
			foundPathDiagnostic = true
		}
		if diagnostic.FieldPath == "adapters.backlog.scripts.sync" {
			foundCommandDiagnostic = true
		}
	}

	if !foundPathDiagnostic {
		t.Fatal("expected missing path diagnostic for adapters.backlog.scripts.done")
	}
	if foundCommandDiagnostic {
		t.Fatal("did not expect diagnostic for non-path command in adapters.backlog.scripts.sync")
	}
}

func TestAutonomyFieldParsesDirectly(t *testing.T) {
	for _, mode := range []string{"auto", "approve"} {
		t.Run(mode, func(t *testing.T) {
			config, err := Parse([]byte(`
autonomy = "` + mode + `"

[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"
`))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if config.Autonomy != mode {
				t.Fatalf("autonomy = %q, want %q", config.Autonomy, mode)
			}
		})
	}
}

func TestAutonomyExplicitPersistsWhenSet(t *testing.T) {
	config, err := Parse([]byte(`
autonomy = "approve"

[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if config.Autonomy != "approve" {
		t.Fatalf("autonomy = %q, want approve", config.Autonomy)
	}
}

func TestAutonomyInvalidValueReturnsError(t *testing.T) {
	_, err := Parse([]byte(`
autonomy = "yolo"

[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"
`))
	if err == nil {
		t.Fatal("expected parse error for invalid autonomy")
	}
	if !strings.Contains(err.Error(), "autonomy") {
		t.Fatalf("error %q missing autonomy field reference", err)
	}
}

func TestPendingApprovalHelper(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Autonomy = "approve"
	if !cfg.PendingApproval() {
		t.Fatal("approve autonomy should require pending approval")
	}
	cfg.Autonomy = "auto"
	if cfg.PendingApproval() {
		t.Fatal("auto autonomy should not require pending approval")
	}
}

func TestLoadParsesFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noodle.toml")
	if err := os.WriteFile(path, []byte(`
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config, _, err := Load(path)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if config.Routing.Defaults.Provider != "claude" {
		t.Fatalf("provider = %q", config.Routing.Defaults.Provider)
	}
}

func TestAvailableRuntimesDefaultsToTmux(t *testing.T) {
	cfg := DefaultConfig()
	got := strings.Join(cfg.AvailableRuntimes(), ",")
	if got != "tmux" {
		t.Fatalf("available runtimes = %q, want tmux", got)
	}
}

func TestAvailableRuntimesIncludesSpritesWhenConfiguredAndTokenSet(t *testing.T) {
	old := os.Getenv("SPRITES_TOKEN")
	if err := os.Setenv("SPRITES_TOKEN", "token-value"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("SPRITES_TOKEN", old)
	})

	cfg, err := Parse([]byte(`
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[runtime.sprites]
sprite_name = "noodle-dev"
`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	got := strings.Join(cfg.AvailableRuntimes(), ",")
	if got != "tmux,sprites" {
		t.Fatalf("available runtimes = %q, want tmux,sprites", got)
	}
}

func TestAvailableRuntimesSkipsSpritesWhenTokenMissing(t *testing.T) {
	old := os.Getenv("SPRITES_TOKEN")
	if err := os.Setenv("SPRITES_TOKEN", ""); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("SPRITES_TOKEN", old)
	})

	cfg, err := Parse([]byte(`
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[runtime.sprites]
sprite_name = "noodle-dev"
`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	got := strings.Join(cfg.AvailableRuntimes(), ",")
	if got != "tmux" {
		t.Fatalf("available runtimes = %q, want tmux", got)
	}
}

func TestSpritesTokenReadsCustomEnvVar(t *testing.T) {
	oldA := os.Getenv("SPRITES_TOKEN")
	oldB := os.Getenv("NOODLE_SPRITES_TOKEN")
	if err := os.Setenv("SPRITES_TOKEN", "ignored"); err != nil {
		t.Fatalf("set SPRITES_TOKEN: %v", err)
	}
	if err := os.Setenv("NOODLE_SPRITES_TOKEN", "chosen"); err != nil {
		t.Fatalf("set NOODLE_SPRITES_TOKEN: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("SPRITES_TOKEN", oldA)
		_ = os.Setenv("NOODLE_SPRITES_TOKEN", oldB)
	})

	cfg := SpritesConfig{TokenEnv: "NOODLE_SPRITES_TOKEN"}
	if got := cfg.Token(); got != "chosen" {
		t.Fatalf("token = %q, want chosen", got)
	}
}
