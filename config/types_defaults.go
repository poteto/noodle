package config

import (
	"os"
	"strings"

	"github.com/poteto/noodle/internal/stringx"
)

const (
	DefaultConfigPath = ".noodle.toml"
)

// Config is the top-level .noodle.toml contract for runtime wiring.
type Config struct {
	Adapters    map[string]AdapterConfig `toml:"adapters"`
	Routing     RoutingConfig            `toml:"routing"`
	Skills      SkillsConfig             `toml:"skills"`
	Mode        string                   `toml:"mode"`
	Concurrency ConcurrencyConfig        `toml:"concurrency"`
	Agents      AgentsConfig             `toml:"agents"`
	Runtime     RuntimeConfig            `toml:"runtime"`
	Server      ServerConfig             `toml:"server"`
}

type AdapterConfig struct {
	Skill   string            `toml:"skill"`
	Scripts map[string]string `toml:"scripts"`
}

type RoutingConfig struct {
	Defaults ModelPolicy `toml:"defaults"`
}

type ModelPolicy struct {
	Provider string `toml:"provider"`
	Model    string `toml:"model"`
}

type SkillsConfig struct {
	Paths []string `toml:"paths"`
}

type ConcurrencyConfig struct {
	MaxConcurrency int `toml:"max_concurrency"`
}

type ProviderConfig struct {
	Path string   `toml:"path"`
	Args []string `toml:"args"`
}

type AgentsConfig struct {
	Claude ProviderConfig `toml:"claude"`
	Codex  ProviderConfig `toml:"codex"`
}

// RuntimeConfig controls the default runtime for spawned cook sessions.
type RuntimeConfig struct {
	Default string        `toml:"default"` // runtime kind, defaults to process
	Process ProcessConfig `toml:"process"`
	Sprites SpritesConfig `toml:"sprites"`
	Cursor  CursorConfig  `toml:"cursor"`

	spritesDefined bool
	cursorDefined  bool
}

// MaxConcurrentFor returns the per-runtime concurrency cap for a given runtime name.
// Returns 0 when unlimited (no per-runtime cap; the global MaxConcurrency ceiling applies).
func (c RuntimeConfig) MaxConcurrentFor(name string) int {
	switch stringx.Normalize(name) {
	case "process":
		return c.Process.MaxConcurrent
	case "sprites":
		return c.Sprites.MaxConcurrent
	case "cursor":
		return c.Cursor.MaxConcurrent
	default:
		return 0
	}
}

type ProcessConfig struct {
	MaxConcurrent int `toml:"max_concurrent"`
}

type SpritesConfig struct {
	TokenEnv      string `toml:"token_env"`
	BaseURL       string `toml:"base_url"`
	SpriteName    string `toml:"sprite_name"`
	GitTokenEnv   string `toml:"git_token_env"`
	MaxConcurrent int    `toml:"max_concurrent"`
}

type CursorConfig struct {
	APIKeyEnv     string `toml:"api_key_env"`
	BaseURL       string `toml:"base_url"`
	Repository    string `toml:"repository"`
	MaxConcurrent int    `toml:"max_concurrent"`
}

// ServerConfig controls the web UI server.
type ServerConfig struct {
	Port    int   `toml:"port"`
	Enabled *bool `toml:"enabled"` // nil = auto (enabled in interactive terminals)
}

type DiagnosticSeverity string

const (
	DiagnosticSeverityRepairable DiagnosticSeverity = "repairable"
	DiagnosticSeverityFatal      DiagnosticSeverity = "fatal"
	DiagnosticSeverityWarning    DiagnosticSeverity = "warning"
)

type ConfigDiagnostic struct {
	FieldPath string
	Message   string
	Severity  DiagnosticSeverity
	Fix       string
	Code      string
	Meta      map[string]string
}

const (
	DiagnosticCodeRoutingDefaultsMissing = "routing_defaults_missing"
	DiagnosticCodeAgentDirMissing        = "agent_dir_missing"
	DiagnosticCodeAdapterSkillMissing    = "adapter_skill_missing"
	DiagnosticCodeAdapterScriptEmpty     = "adapter_script_empty"
	DiagnosticCodeAdapterScriptMissing   = "adapter_script_missing"
	DiagnosticCodeProviderUnknown        = "provider_unknown"
	DiagnosticCodeRuntimeDefaultUnknown  = "runtime_default_unknown"
	DiagnosticCodeNoBacklogAdapter       = "no_backlog_adapter"
	DiagnosticCodeConfigFieldRemoved     = "config_field_removed"
	DiagnosticCodeConfigFieldUnknown     = "config_field_unknown"
	DiagnosticCodeConfigValueNormalized  = "config_value_normalized"
)

type ValidationResult struct {
	Diagnostics []ConfigDiagnostic
}

func (r ValidationResult) CanSpawn() bool {
	return len(r.Fatals()) == 0
}

func (r ValidationResult) Repairables() []ConfigDiagnostic {
	return filterDiagnostics(r.Diagnostics, DiagnosticSeverityRepairable)
}

func (r ValidationResult) Fatals() []ConfigDiagnostic {
	return filterDiagnostics(r.Diagnostics, DiagnosticSeverityFatal)
}

func (r ValidationResult) Warnings() []ConfigDiagnostic {
	return filterDiagnostics(r.Diagnostics, DiagnosticSeverityWarning)
}

func filterDiagnostics(in []ConfigDiagnostic, severity DiagnosticSeverity) []ConfigDiagnostic {
	out := make([]ConfigDiagnostic, 0, len(in))
	for _, diagnostic := range in {
		if diagnostic.Severity == severity {
			out = append(out, diagnostic)
		}
	}
	return out
}

// DefaultConfig returns the full no-config-file defaults.
func DefaultConfig() Config {
	return Config{
		Adapters: defaultAdapters(),
		Routing: RoutingConfig{
			Defaults: ModelPolicy{
				Provider: "claude",
				Model:    "claude-opus-4-6",
			},
		},
		Skills: SkillsConfig{
			Paths: defaultSkillPaths(),
		},
		Mode: defaultMode,
		Concurrency: ConcurrencyConfig{
			MaxConcurrency: 4,
		},
		Agents: AgentsConfig{},
		Runtime: RuntimeConfig{
			Default: "process",
			Process: ProcessConfig{MaxConcurrent: 4},
			Sprites: SpritesConfig{MaxConcurrent: 50},
			Cursor:  CursorConfig{MaxConcurrent: 10},
		},
		Server: ServerConfig{
			Port: 3000,
		},
	}
}

func defaultAdapters() map[string]AdapterConfig {
	return map[string]AdapterConfig{
		"backlog": {
			Skill: "backlog",
			Scripts: map[string]string{
				"sync": "adapters/backlog-sync",
				"add":  "adapters/backlog-add",
				"done": "adapters/backlog-done",
				"edit": "adapters/backlog-edit",
			},
		},
	}
}

func defaultSkillPaths() []string {
	return []string{".agents/skills"}
}

func (c Config) AvailableRuntimes() []string {
	available := []string{"process"}
	if c.Runtime.spritesDefined && c.Runtime.Sprites.Token() != "" {
		available = append(available, "sprites")
	}
	if c.Runtime.cursorDefined && c.Runtime.Cursor.APIKey() != "" {
		available = append(available, "cursor")
	}
	return available
}

func (c SpritesConfig) Token() string {
	key := strings.TrimSpace(c.TokenEnv)
	if key == "" {
		key = "SPRITES_TOKEN"
	}
	return strings.TrimSpace(os.Getenv(key))
}

func (c SpritesConfig) GitToken() string {
	key := strings.TrimSpace(c.GitTokenEnv)
	if key == "" {
		key = "GITHUB_TOKEN"
	}
	return strings.TrimSpace(os.Getenv(key))
}

func (c CursorConfig) APIKey() string {
	key := strings.TrimSpace(c.APIKeyEnv)
	if key == "" {
		key = "CURSOR_API_KEY"
	}
	return strings.TrimSpace(os.Getenv(key))
}
