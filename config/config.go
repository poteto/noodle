package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
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
	Plans       PlansConfig              `toml:"plans"`
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
	MaxCooks int `toml:"max_cooks"`
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
	Default string        `toml:"default"` // runtime kind, defaults to tmux
	Sprites SpritesConfig `toml:"sprites"`
	Cursor  CursorConfig  `toml:"cursor"`

	spritesDefined bool
	cursorDefined  bool
}

type SpritesConfig struct {
	TokenEnv    string `toml:"token_env"`
	BaseURL     string `toml:"base_url"`
	SpriteName  string `toml:"sprite_name"`
	GitTokenEnv string `toml:"git_token_env"`
}

type CursorConfig struct {
	APIKeyEnv  string `toml:"api_key_env"`
	BaseURL    string `toml:"base_url"`
	Repository string `toml:"repository"`
}

// PlansConfig controls plan lifecycle behavior.
type PlansConfig struct {
	OnDone string `toml:"on_done"` // "keep" | "remove"
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
	DiagnosticCodeRuntimeTmuxMissing     = "runtime_tmux_missing"
	DiagnosticCodeAgentDirMissing        = "agent_dir_missing"
	DiagnosticCodeAdapterSkillMissing    = "adapter_skill_missing"
	DiagnosticCodeAdapterScriptEmpty     = "adapter_script_empty"
	DiagnosticCodeAdapterScriptMissing   = "adapter_script_missing"
	DiagnosticCodeProviderUnknown        = "provider_unknown"
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
			MaxCooks: 4,
		},
		Agents: AgentsConfig{},
		Runtime: RuntimeConfig{
			Default: "tmux",
		},
		Plans: PlansConfig{
			OnDone: "keep",
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

// Load reads and validates config from disk.
func Load(path string) (Config, ValidationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			config := DefaultConfig()
			return config, Validate(config), nil
		}
		return Config{}, ValidationResult{}, fmt.Errorf("read %s: %w", path, err)
	}

	config, err := Parse(data)
	if err != nil {
		return Config{}, ValidationResult{}, err
	}
	return config, Validate(config), nil
}

// Parse parses a .noodle.toml payload and applies omitted-field defaults.
func Parse(data []byte) (Config, error) {
	var config Config
	metadata, err := toml.Decode(string(data), &config)
	if err != nil {
		return Config{}, fmt.Errorf("parse .noodle.toml: %w", err)
	}

	applyDefaultsFromMetadata(&config, metadata)
	if err := validateParsedValues(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func applyDefaultsFromMetadata(config *Config, metadata toml.MetaData) {
	if !metadata.IsDefined("skills", "paths") || len(config.Skills.Paths) == 0 {
		config.Skills.Paths = defaultSkillPaths()
	}

	if !metadata.IsDefined("schedule", "run") {
		config.Schedule.Run = "after-each"
	}
	if !metadata.IsDefined("schedule", "model") {
		config.Schedule.Model = "claude-sonnet"
	}

	if !metadata.IsDefined("routing", "defaults", "provider") {
		config.Routing.Defaults.Provider = "claude"
	}
	if !metadata.IsDefined("routing", "defaults", "model") {
		config.Routing.Defaults.Model = "claude-opus-4-6"
	}
	if config.Routing.Tags == nil {
		config.Routing.Tags = map[string]ModelPolicy{}
	}

	if !metadata.IsDefined("autonomy") {
		config.Autonomy = AutonomyAuto
	}

	if !metadata.IsDefined("recovery", "max_retries") {
		config.Recovery.MaxRetries = 3
	}
	if !metadata.IsDefined("monitor", "stuck_threshold") {
		config.Monitor.StuckThreshold = "120s"
	}
	if !metadata.IsDefined("monitor", "ticket_stale") {
		config.Monitor.TicketStale = "30m"
	}
	if !metadata.IsDefined("monitor", "poll_interval") {
		config.Monitor.PollInterval = "5s"
	}

	if !metadata.IsDefined("concurrency", "max_cooks") {
		config.Concurrency.MaxCooks = 4
	}

	config.Runtime.spritesDefined = metadata.IsDefined("runtime", "sprites")
	config.Runtime.cursorDefined = metadata.IsDefined("runtime", "cursor")
	if !metadata.IsDefined("runtime", "default") {
		config.Runtime.Default = "tmux"
	}

	if !metadata.IsDefined("plans", "on_done") {
		config.Plans.OnDone = "keep"
	}

	if !metadata.IsDefined("server", "port") {
		config.Server.Port = 3000
	}

	if metadata.IsDefined("adapters") {
		if config.Adapters == nil {
			config.Adapters = map[string]AdapterConfig{}
		}
		applyAdapterDefaults(config.Adapters)
	}
}

func applyAdapterDefaults(adapters map[string]AdapterConfig) {
	defaults := defaultAdapters()
	for name, defaultAdapter := range defaults {
		adapter, ok := adapters[name]
		if !ok {
			continue
		}
		if adapter.Skill == "" {
			adapter.Skill = defaultAdapter.Skill
		}
		if adapter.Scripts == nil {
			adapter.Scripts = map[string]string{}
		}
		for action, command := range defaultAdapter.Scripts {
			if _, has := adapter.Scripts[action]; !has {
				adapter.Scripts[action] = command
			}
		}
		adapters[name] = adapter
	}
}

func defaultSkillPaths() []string {
	return []string{".agents/skills"}
}

func validateParsedValues(config Config) error {
	switch config.Autonomy {
	case AutonomyAuto, AutonomyApprove:
	case "":
		// treated as default in applyDefaults
	default:
		return fmt.Errorf(
			"autonomy: unsupported value %q (expected auto, approve)",
			config.Autonomy,
		)
	}
	if err := validateProvider("routing.defaults.provider", config.Routing.Defaults.Provider); err != nil {
		return err
	}
	if config.Routing.Defaults.Model == "" {
		return fmt.Errorf("routing.defaults.model: model is required")
	}
	for tag, policy := range config.Routing.Tags {
		field := fmt.Sprintf("routing.tags.%s.provider", tag)
		if err := validateProvider(field, policy.Provider); err != nil {
			return err
		}
	}

	switch config.Schedule.Run {
	case "after-each", "after-n", "manual":
	default:
		return fmt.Errorf(
			"schedule.run: unsupported value %q (expected after-each, after-n, manual)",
			config.Schedule.Run,
		)
	}

	if config.Recovery.MaxRetries < 0 {
		return fmt.Errorf("recovery.max_retries: must be greater than or equal to 0")
	}
	if config.Concurrency.MaxCooks <= 0 {
		return fmt.Errorf("concurrency.max_cooks: must be greater than 0")
	}

	switch config.Plans.OnDone {
	case "keep", "remove":
	default:
		return fmt.Errorf(
			"plans.on_done: unsupported value %q (expected keep, remove)",
			config.Plans.OnDone,
		)
	}

	if config.Server.Port < 0 || config.Server.Port > 65535 {
		return fmt.Errorf("server.port: must be between 0 and 65535")
	}

	if err := validatePositiveDuration("monitor.stuck_threshold", config.Monitor.StuckThreshold); err != nil {
		return err
	}
	if err := validatePositiveDuration("monitor.ticket_stale", config.Monitor.TicketStale); err != nil {
		return err
	}
	if err := validatePositiveDuration("monitor.poll_interval", config.Monitor.PollInterval); err != nil {
		return err
	}
	return nil
}

func validateProvider(fieldPath, provider string) error {
	if strings.TrimSpace(provider) == "" {
		return fmt.Errorf("%s: provider is required", fieldPath)
	}
	return nil
}

func validatePositiveDuration(fieldPath, raw string) error {
	duration, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("%s: invalid duration %q", fieldPath, raw)
	}
	if duration <= 0 {
		return fmt.Errorf("%s: duration must be greater than 0", fieldPath)
	}
	return nil
}

var (
	lookPath = exec.LookPath
	statPath = os.Stat
)

func expandHomePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return homeDir, nil
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		return filepath.Join(homeDir, strings.TrimPrefix(strings.TrimPrefix(path, "~/"), "~\\")), nil
	}
	// "~user" style paths are not supported; keep as-is.
	return path, nil
}

// Validate classifies repairable and fatal runtime diagnostics.
func Validate(config Config) ValidationResult {
	result := ValidationResult{
		Diagnostics: make([]ConfigDiagnostic, 0),
	}

	if config.Routing.Defaults.Provider == "" || config.Routing.Defaults.Model == "" {
		result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
			FieldPath: "routing.defaults",
			Message:   "provider and model must both be configured",
			Severity:  DiagnosticSeverityFatal,
			Fix:       "Set routing.defaults.provider and routing.defaults.model in .noodle.toml.",
			Code:      DiagnosticCodeRoutingDefaultsMissing,
		})
	} else if !isKnownProvider(config.Routing.Defaults.Provider) {
		result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
			FieldPath: "routing.defaults.provider",
			Message:   fmt.Sprintf("unknown provider %q (valid: claude, codex)", config.Routing.Defaults.Provider),
			Severity:  DiagnosticSeverityFatal,
			Fix:       "Set routing.defaults.provider to \"claude\" or \"codex\" in .noodle.toml.",
			Code:      DiagnosticCodeProviderUnknown,
		})
	}

	if _, err := lookPath("tmux"); err != nil {
		result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
			FieldPath: "runtime.tmux",
			Message:   "tmux is not available on PATH",
			Severity:  DiagnosticSeverityFatal,
			Fix:       "Install tmux and ensure it is available on PATH.",
			Code:      DiagnosticCodeRuntimeTmuxMissing,
		})
	}

	appendAgentDirDiagnostic := func(fieldPath, value string) {
		if value == "" {
			return
		}
		resolvedPath := value
		if expanded, err := expandHomePath(value); err == nil {
			resolvedPath = expanded
		}
		info, err := statPath(resolvedPath)
		if err == nil && info.IsDir() {
			return
		}
		result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
			FieldPath: fieldPath,
			Message:   fmt.Sprintf("directory %q not found", value),
			Severity:  DiagnosticSeverityFatal,
			Fix:       fmt.Sprintf("Create %s or update %s to the correct directory.", value, fieldPath),
			Code:      DiagnosticCodeAgentDirMissing,
			Meta: map[string]string{
				"path": value,
			},
		})
	}
	appendAgentDirDiagnostic("agents.claude.path", config.Agents.Claude.Path)
	appendAgentDirDiagnostic("agents.codex.path", config.Agents.Codex.Path)

	for adapterName, adapter := range config.Adapters {
		if adapter.Skill == "" {
			result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
				FieldPath: fmt.Sprintf("adapters.%s.skill", adapterName),
				Message:   "skill is not configured",
				Severity:  DiagnosticSeverityRepairable,
				Fix:       fmt.Sprintf("Set adapters.%s.skill in .noodle.toml.", adapterName),
				Code:      DiagnosticCodeAdapterSkillMissing,
				Meta: map[string]string{
					"adapter": adapterName,
				},
			})
		}
		for action, command := range adapter.Scripts {
			if strings.TrimSpace(command) == "" {
				result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
					FieldPath: fmt.Sprintf("adapters.%s.scripts.%s", adapterName, action),
					Message:   "script command is empty",
					Severity:  DiagnosticSeverityRepairable,
					Fix:       fmt.Sprintf("Set adapters.%s.scripts.%s to an executable command.", adapterName, action),
					Code:      DiagnosticCodeAdapterScriptEmpty,
					Meta: map[string]string{
						"adapter": adapterName,
						"action":  action,
					},
				})
				continue
			}
			token := firstToken(command)
			if !commandLooksPath(token) {
				continue
			}
			if _, err := statPath(token); err == nil {
				continue
			}
			result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
				FieldPath: fmt.Sprintf("adapters.%s.scripts.%s", adapterName, action),
				Message:   fmt.Sprintf("script path %q not found", token),
				Severity:  DiagnosticSeverityRepairable,
				Fix:       fmt.Sprintf("Create %s or update adapters.%s.scripts.%s.", token, adapterName, action),
				Code:      DiagnosticCodeAdapterScriptMissing,
				Meta: map[string]string{
					"adapter": adapterName,
					"action":  action,
					"path":    token,
				},
			})
		}
	}

	return result
}

func firstToken(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func commandLooksPath(token string) bool {
	if token == "" {
		return false
	}
	return strings.HasPrefix(token, ".") ||
		strings.HasPrefix(token, "/") ||
		strings.Contains(token, string(filepath.Separator))
}

// PendingApproval returns whether successful cooks need human approval to merge.
func (c Config) PendingApproval() bool {
	return c.Autonomy == AutonomyApprove
}

func (c Config) AvailableRuntimes() []string {
	available := []string{"tmux"}
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

func isKnownProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "claude", "codex":
		return true
	}
	return false
}

func (c CursorConfig) APIKey() string {
	key := strings.TrimSpace(c.APIKeyEnv)
	if key == "" {
		key = "CURSOR_API_KEY"
	}
	return strings.TrimSpace(os.Getenv(key))
}
