package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigValues(t *testing.T) {
	config := DefaultConfig()

	if got := strings.Join(config.Skills.Paths, ","); got != ".agents/skills" {
		t.Fatalf("skills.paths default = %q", got)
	}
	if config.Routing.Defaults.Provider != "claude" {
		t.Fatalf("routing.defaults.provider default = %q", config.Routing.Defaults.Provider)
	}
	if config.Routing.Defaults.Model != "claude-opus-4-6" {
		t.Fatalf("routing.defaults.model default = %q", config.Routing.Defaults.Model)
	}
	if config.Mode != "auto" {
		t.Fatalf("mode default = %q, want auto", config.Mode)
	}
	if config.Concurrency.MaxConcurrency != 4 {
		t.Fatalf("concurrency.max_concurrency default = %d", config.Concurrency.MaxConcurrency)
	}
	if config.Agents.Claude.Path != "" || config.Agents.Codex.Path != "" {
		t.Fatalf("agent path defaults should be empty: %#v", config.Agents)
	}
	if config.Runtime.Default != "process" {
		t.Fatalf("runtime.default = %q, want process", config.Runtime.Default)
	}
	if config.Runtime.Process.MaxConcurrent != 4 {
		t.Fatalf("runtime.process.max_concurrent default = %d, want 4", config.Runtime.Process.MaxConcurrent)
	}
	if config.Runtime.Sprites.MaxConcurrent != 50 {
		t.Fatalf("runtime.sprites.max_concurrent default = %d, want 50", config.Runtime.Sprites.MaxConcurrent)
	}
	if config.Runtime.Cursor.MaxConcurrent != 10 {
		t.Fatalf("runtime.cursor.max_concurrent default = %d, want 10", config.Runtime.Cursor.MaxConcurrent)
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
	if _, ok := config.Adapters["backlog"]; !ok {
		t.Fatal("expected default backlog adapter when config file is missing")
	}
}

func TestParseConfigRoundTrip(t *testing.T) {
	tomlPayload := `
[routing.defaults]
provider = "codex"
model = "gpt-5.3-codex"

[skills]
paths = [".agents/skills"]

[concurrency]
max_concurrency = 2

[adapters.backlog]
skill = "my-backlog"

[adapters.backlog.scripts]
sync = "gh issue list"
add = "gh issue create"
done = "gh issue close"
edit = "gh issue edit"

`

	config, err := Parse([]byte(tomlPayload))
	if err != nil {
		t.Fatalf("Parse config: %v", err)
	}

	if config.Routing.Defaults.Provider != "codex" {
		t.Fatalf("routing.defaults.provider = %q", config.Routing.Defaults.Provider)
	}
	if config.Mode != "auto" {
		t.Fatalf("expected default mode=auto, got %q", config.Mode)
	}
	if config.Concurrency.MaxConcurrency != 2 {
		t.Fatalf("concurrency.max_concurrency = %d", config.Concurrency.MaxConcurrency)
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

	if config.Mode != "auto" {
		t.Fatalf("expected default mode=auto, got %q", config.Mode)
	}
	if config.Adapters != nil {
		t.Fatal("adapters should remain unset when omitted from an existing config file")
	}
}

func TestParseInvalidValues(t *testing.T) {
	tests := []struct {
		name               string
		payload            string
		wantMode           string
		wantProvider       string
		wantMaxConcurrency int
	}{
		{
			name: "missing provider",
			payload: `
[routing.defaults]
provider = ""
model = "x"
`,
			wantProvider:       "claude",
			wantMode:           "auto",
			wantMaxConcurrency: 4,
		},
		{
			name: "invalid mode",
			payload: `
mode = "yolo"

[routing.defaults]
provider = "claude"
model = "x"
`,
			wantMode:           "auto",
			wantProvider:       "claude",
			wantMaxConcurrency: 4,
		},
		{
			name: "invalid max concurrency",
			payload: `
[routing.defaults]
provider = "claude"
model = "x"

[concurrency]
max_concurrency = 0
`,
			wantMode:           "auto",
			wantProvider:       "claude",
			wantMaxConcurrency: 4,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := Parse([]byte(test.payload))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if cfg.Mode != test.wantMode {
				t.Fatalf("mode = %q, want %q", cfg.Mode, test.wantMode)
			}
			if cfg.Routing.Defaults.Provider != test.wantProvider {
				t.Fatalf("provider = %q, want %q", cfg.Routing.Defaults.Provider, test.wantProvider)
			}
			if cfg.Concurrency.MaxConcurrency != test.wantMaxConcurrency {
				t.Fatalf("max_concurrency = %d, want %d", cfg.Concurrency.MaxConcurrency, test.wantMaxConcurrency)
			}
		})
	}
}

func TestLoadWarnsOnRemovedFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noodle.toml")
	if err := os.WriteFile(path, []byte(`
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[recovery]
max_retries = 9

[monitor]
poll_interval = "9s"

[concurrency]
max_cooks = 99
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, validation, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Concurrency.MaxConcurrency != 4 {
		t.Fatalf("max_concurrency = %d, want default 4", cfg.Concurrency.MaxConcurrency)
	}

	foundRemoved := false
	for _, diagnostic := range validation.Warnings() {
		if diagnostic.Code == DiagnosticCodeConfigFieldRemoved {
			foundRemoved = true
			break
		}
	}
	if !foundRemoved {
		t.Fatal("expected removed-field warning diagnostics")
	}
}

func TestValidationClassification(t *testing.T) {
	oldStatPath := statPath
	statPath = func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }
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
	oldStatPath := statPath
	statPath = func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }
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

func TestValidateExpandsHomeInAgentPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	if err := os.MkdirAll(filepath.Join(homeDir, ".claude"), 0o755); err != nil {
		t.Fatalf("mkdir ~/.claude: %v", err)
	}

	config := DefaultConfig()
	config.Agents.Claude.Path = "~/.claude"

	result := Validate(config)
	for _, diagnostic := range result.Fatals() {
		if diagnostic.FieldPath == "agents.claude.path" {
			t.Fatalf("unexpected fatal diagnostic for expanded home path: %+v", diagnostic)
		}
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
	oldStatPath := statPath
	statPath = func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }
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

func TestModeFieldParsesDirectly(t *testing.T) {
	for _, mode := range []string{"auto", "supervised", "manual"} {
		t.Run(mode, func(t *testing.T) {
			config, err := Parse([]byte(`
mode = "` + mode + `"

[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"
`))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if config.Mode != mode {
				t.Fatalf("mode = %q, want %q", config.Mode, mode)
			}
		})
	}
}

func TestModeExplicitPersistsWhenSet(t *testing.T) {
	config, err := Parse([]byte(`
mode = "supervised"

[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if config.Mode != "supervised" {
		t.Fatalf("mode = %q, want supervised", config.Mode)
	}
}

func TestModeInvalidValueFallsBackToDefault(t *testing.T) {
	cfg, err := Parse([]byte(`
mode = "yolo"

[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Mode != "auto" {
		t.Fatalf("mode = %q, want auto", cfg.Mode)
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

func TestAvailableRuntimesDefaultsToProcess(t *testing.T) {
	cfg := DefaultConfig()
	got := strings.Join(cfg.AvailableRuntimes(), ",")
	if got != "process" {
		t.Fatalf("available runtimes = %q, want process", got)
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
	if got != "process,sprites" {
		t.Fatalf("available runtimes = %q, want process,sprites", got)
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
	if got != "process" {
		t.Fatalf("available runtimes = %q, want process", got)
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

func TestParsePerRuntimeMaxConcurrent(t *testing.T) {
	cfg, err := Parse([]byte(`
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[runtime.process]
max_concurrent = 8

[runtime.sprites]
max_concurrent = 100
sprite_name = "test"
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Runtime.Process.MaxConcurrent != 8 {
		t.Fatalf("runtime.process.max_concurrent = %d, want 8", cfg.Runtime.Process.MaxConcurrent)
	}
	if cfg.Runtime.Sprites.MaxConcurrent != 100 {
		t.Fatalf("runtime.sprites.max_concurrent = %d, want 100", cfg.Runtime.Sprites.MaxConcurrent)
	}
	// Cursor was not set — should get default.
	if cfg.Runtime.Cursor.MaxConcurrent != 10 {
		t.Fatalf("runtime.cursor.max_concurrent = %d, want 10", cfg.Runtime.Cursor.MaxConcurrent)
	}
}

func TestLoadWarnsOnNormalizedRuntimeDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noodle.toml")
	if err := os.WriteFile(path, []byte(`
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[runtime]
default = "tmux"
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, validation, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Runtime.Default != "process" {
		t.Fatalf("runtime.default = %q, want process", cfg.Runtime.Default)
	}
	found := false
	for _, w := range validation.Warnings() {
		if w.FieldPath == "runtime.default" && w.Code == DiagnosticCodeConfigValueNormalized {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected normalized runtime.default warning")
	}
}

func TestValidateNoBacklogAdapterDiagnostic(t *testing.T) {
	cfg, err := Parse([]byte(`
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// Parsed config with no adapters section — adapters is nil.
	result := Validate(cfg)
	found := false
	for _, d := range result.Repairables() {
		if d.Code == DiagnosticCodeNoBacklogAdapter {
			found = true
			if d.FieldPath != "adapters.backlog" {
				t.Fatalf("field path = %q, want adapters.backlog", d.FieldPath)
			}
		}
	}
	if !found {
		t.Fatal("expected no_backlog_adapter diagnostic when adapters is nil")
	}
	// Should still be spawnable (repairable, not fatal).
	if !result.CanSpawn() {
		t.Fatal("no backlog adapter should be repairable, not fatal")
	}
}

func TestValidateNoBacklogAdapterWithOtherAdapters(t *testing.T) {
	cfg, err := Parse([]byte(`
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[adapters.custom]
skill = "custom"

[adapters.custom.scripts]
sync = "echo hello"
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	result := Validate(cfg)
	found := false
	for _, d := range result.Repairables() {
		if d.Code == DiagnosticCodeNoBacklogAdapter {
			found = true
		}
	}
	if !found {
		t.Fatal("expected no_backlog_adapter diagnostic when backlog adapter missing from map")
	}
}

func TestValidateBacklogAdapterPresentNoNoDiagnostic(t *testing.T) {
	cfg := DefaultConfig()
	result := Validate(cfg)
	for _, d := range result.Diagnostics {
		if d.Code == DiagnosticCodeNoBacklogAdapter {
			t.Fatal("no_backlog_adapter diagnostic should not fire when backlog adapter is present")
		}
	}
}

func TestMaxConcurrentForLookup(t *testing.T) {
	rc := RuntimeConfig{
		Process: ProcessConfig{MaxConcurrent: 4},
		Sprites: SpritesConfig{MaxConcurrent: 50},
		Cursor:  CursorConfig{MaxConcurrent: 10},
	}
	tests := []struct {
		name string
		want int
	}{
		{"process", 4},
		{"Process", 4},
		{"sprites", 50},
		{"SPRITES", 50},
		{"cursor", 10},
		{"unknown", 0},
	}
	for _, tt := range tests {
		if got := rc.MaxConcurrentFor(tt.name); got != tt.want {
			t.Errorf("MaxConcurrentFor(%q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}
