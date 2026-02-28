package config

import (
	"os"
	"strings"
)

const (
	DefaultConfigPath = ".noodle.toml"
)

// Autonomy modes control how much human oversight the loop requires.
const (
	AutonomyAuto    = "auto"    // auto-merge on success, unless task type disallows it
	AutonomyApprove = "approve" // human must confirm merge
)

// Config is the top-level .noodle.toml contract for runtime wiring.
type Config struct {
	Adapters    map[string]AdapterConfig `toml:"adapters"`
	Schedule    ScheduleConfig           `toml:"schedule"`
	Routing     RoutingConfig            `toml:"routing"`
	Skills      SkillsConfig             `toml:"skills"`
	Autonomy    string                   `toml:"autonomy"`
	Recovery    RecoveryConfig           `toml:"recovery"`
	Monitor     MonitorConfig            `toml:"monitor"`
	Concurrency ConcurrencyConfig        `toml:"concurrency"`
	Agents      AgentsConfig             `toml:"agents"`
	Runtime     RuntimeConfig            `toml:"runtime"`
	Server      ServerConfig             `toml:"server"`
}

type AdapterConfig struct {
	Skill   string            `toml:"skill"`
	Scripts map[string]string `toml:"scripts"`
}

type ScheduleConfig struct {
	Run   string `toml:"run"`
	Model string `toml:"model"`
}

type RoutingConfig struct {
	Defaults ModelPolicy            `toml:"defaults"`
	Tags     map[string]ModelPolicy `toml:"tags"`
}

type ModelPolicy struct {
	Provider string `toml:"provider"`
	Model    string `toml:"model"`
}

type SkillsConfig struct {
	Paths []string `toml:"paths"`
}

type RecoveryConfig struct {
	MaxRetries int `toml:"max_retries"`
}

type MonitorConfig struct {
	StuckThreshold string `toml:"stuck_threshold"`
	TicketStale    string `toml:"ticket_stale"`
	PollInterval   string `toml:"poll_interval"`
}

type ConcurrencyConfig struct {
	MaxCooks                   int    `toml:"max_cooks"`
	MaxCompletionOverflow      int    `toml:"max_completion_overflow"`
	MergeBackpressureThreshold int    `toml:"merge_backpressure_threshold"`
	ShutdownTimeout            string `toml:"shutdown_timeout"`
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
// Returns 0 when unlimited (no per-runtime cap; the global MaxCooks ceiling applies).
func (c RuntimeConfig) MaxConcurrentFor(name string) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
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
	DiagnosticCodeRuntimeProcessMissing  = "runtime_process_missing"
	DiagnosticCodeAgentDirMissing        = "agent_dir_missing"
	DiagnosticCodeAdapterSkillMissing    = "adapter_skill_missing"
	DiagnosticCodeAdapterScriptEmpty     = "adapter_script_empty"
	DiagnosticCodeAdapterScriptMissing   = "adapter_script_missing"
	DiagnosticCodeProviderUnknown        = "provider_unknown"
	DiagnosticCodeRuntimeDefaultUnknown  = "runtime_default_unknown"
	DiagnosticCodeNoBacklogAdapter       = "no_backlog_adapter"
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
		Schedule: ScheduleConfig{
			Run:   "after-each",
			Model: "claude-sonnet",
		},
		Routing: RoutingConfig{
			Defaults: ModelPolicy{
				Provider: "claude",
				Model:    "claude-opus-4-6",
			},
			Tags: map[string]ModelPolicy{},
		},
		Skills: SkillsConfig{
			Paths: defaultSkillPaths(),
		},
		Autonomy: AutonomyAuto,
		Recovery: RecoveryConfig{
			MaxRetries: 3,
		},
		Monitor: MonitorConfig{
			StuckThreshold: "120s",
			TicketStale:    "30m",
			PollInterval:   "5s",
		},
		Concurrency: ConcurrencyConfig{
			MaxCooks:                   4,
			MaxCompletionOverflow:      1024,
			MergeBackpressureThreshold: 128,
			ShutdownTimeout:            "30s",
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
				"sync": ".noodle/adapters/backlog-sync",
				"add":  ".noodle/adapters/backlog-add",
				"done": ".noodle/adapters/backlog-done",
				"edit": ".noodle/adapters/backlog-edit",
			},
		},
	}
}

func defaultSkillPaths() []string {
	return []string{".agents/skills"}
}

// PendingApproval returns whether successful cooks need human approval to merge.
func (c Config) PendingApproval() bool {
	return c.Autonomy == AutonomyApprove
}

func (c Config) AvailableRuntimes() []string {
	available := []string{"process"}
	if c.Runtime.spritesDefined && c.Runtime.Sprites.Token() != "" {
		available = append(available, "sprites")
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
