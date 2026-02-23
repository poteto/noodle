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
	DefaultConfigPath = "noodle.toml"
)

// Config is the top-level noodle.toml contract for runtime wiring.
type Config struct {
	Phases      map[string]string        `toml:"phases"`
	Adapters    map[string]AdapterConfig `toml:"adapters"`
	Prioritize  PrioritizeConfig         `toml:"prioritize"`
	Routing     RoutingConfig            `toml:"routing"`
	Skills      SkillsConfig             `toml:"skills"`
	Review      ReviewConfig             `toml:"review"`
	Recovery    RecoveryConfig           `toml:"recovery"`
	Monitor     MonitorConfig            `toml:"monitor"`
	Concurrency ConcurrencyConfig        `toml:"concurrency"`
	Agents      AgentsConfig             `toml:"agents"`
}

type AdapterConfig struct {
	Skill   string            `toml:"skill"`
	Scripts map[string]string `toml:"scripts"`
}

type PrioritizeConfig struct {
	Skill string `toml:"skill"`
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

type ReviewConfig struct {
	Enabled bool `toml:"enabled"`
}

type RecoveryConfig struct {
	MaxRetries         int    `toml:"max_retries"`
	RetrySuffixPattern string `toml:"retry_suffix_pattern"`
}

type MonitorConfig struct {
	StuckThreshold string `toml:"stuck_threshold"`
	TicketStale    string `toml:"ticket_stale"`
	PollInterval   string `toml:"poll_interval"`
}

type ConcurrencyConfig struct {
	MaxCooks int `toml:"max_cooks"`
}

type AgentsConfig struct {
	ClaudeDir string `toml:"claude_dir"`
	CodexDir  string `toml:"codex_dir"`
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
}

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
		Phases: map[string]string{
			"oops":      "oops",
			"debugging": "debugging",
		},
		Adapters: defaultAdapters(),
		Prioritize: PrioritizeConfig{
			Skill: "prioritize",
			Run:   "after-each",
			Model: "claude-sonnet",
		},
		Routing: RoutingConfig{
			Defaults: ModelPolicy{
				Provider: "claude",
				Model:    "claude-sonnet-4-6",
			},
			Tags: map[string]ModelPolicy{},
		},
		Skills: SkillsConfig{
			Paths: []string{"skills", "~/.noodle/skills"},
		},
		Review: ReviewConfig{
			Enabled: true,
		},
		Recovery: RecoveryConfig{
			MaxRetries:         3,
			RetrySuffixPattern: "-recover-%d",
		},
		Monitor: MonitorConfig{
			StuckThreshold: "120s",
			TicketStale:    "30m",
			PollInterval:   "5s",
		},
		Concurrency: ConcurrencyConfig{
			MaxCooks: 4,
		},
		Agents: AgentsConfig{
			ClaudeDir: "",
			CodexDir:  "",
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
		"plans": {
			Skill: "plans",
			Scripts: map[string]string{
				"sync":      ".noodle/adapters/plans-sync",
				"create":    ".noodle/adapters/plan-create",
				"done":      ".noodle/adapters/plan-done",
				"phase-add": ".noodle/adapters/plan-phase-add",
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

// Parse parses a noodle.toml payload and applies omitted-field defaults.
func Parse(data []byte) (Config, error) {
	var config Config
	metadata, err := toml.Decode(string(data), &config)
	if err != nil {
		return Config{}, fmt.Errorf("parse noodle.toml: %w", err)
	}

	applyDefaultsFromMetadata(&config, metadata)
	if err := validateParsedValues(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func applyDefaultsFromMetadata(config *Config, metadata toml.MetaData) {
	if config.Phases == nil {
		config.Phases = map[string]string{}
	}
	if _, ok := config.Phases["oops"]; !ok {
		config.Phases["oops"] = "oops"
	}
	if _, ok := config.Phases["debugging"]; !ok {
		config.Phases["debugging"] = "debugging"
	}

	if !metadata.IsDefined("skills", "paths") || len(config.Skills.Paths) == 0 {
		config.Skills.Paths = []string{"skills", "~/.noodle/skills"}
	}

	if !metadata.IsDefined("prioritize", "run") {
		config.Prioritize.Run = "after-each"
	}
	if !metadata.IsDefined("prioritize", "skill") {
		config.Prioritize.Skill = "prioritize"
	}
	if !metadata.IsDefined("prioritize", "model") {
		config.Prioritize.Model = "claude-sonnet"
	}

	if !metadata.IsDefined("routing", "defaults", "provider") {
		config.Routing.Defaults.Provider = "claude"
	}
	if !metadata.IsDefined("routing", "defaults", "model") {
		config.Routing.Defaults.Model = "claude-sonnet-4-6"
	}
	if config.Routing.Tags == nil {
		config.Routing.Tags = map[string]ModelPolicy{}
	}

	if !metadata.IsDefined("review", "enabled") {
		config.Review.Enabled = true
	}

	if !metadata.IsDefined("recovery", "max_retries") {
		config.Recovery.MaxRetries = 3
	}
	if !metadata.IsDefined("recovery", "retry_suffix_pattern") {
		config.Recovery.RetrySuffixPattern = "-recover-%d"
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

func validateParsedValues(config Config) error {
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

	switch config.Prioritize.Run {
	case "after-each", "after-n", "manual":
	default:
		return fmt.Errorf(
			"prioritize.run: unsupported value %q (expected after-each, after-n, manual)",
			config.Prioritize.Run,
		)
	}
	if strings.TrimSpace(config.Prioritize.Skill) == "" {
		return fmt.Errorf("prioritize.skill: skill is required")
	}

	if config.Recovery.MaxRetries < 0 {
		return fmt.Errorf("recovery.max_retries: must be greater than or equal to 0")
	}
	if !strings.Contains(config.Recovery.RetrySuffixPattern, "%d") {
		return fmt.Errorf("recovery.retry_suffix_pattern: must include %%d placeholder")
	}
	if config.Concurrency.MaxCooks <= 0 {
		return fmt.Errorf("concurrency.max_cooks: must be greater than 0")
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
	switch provider {
	case "claude", "codex":
		return nil
	default:
		return fmt.Errorf("%s: unsupported provider %q", fieldPath, provider)
	}
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
			Fix:       "Set routing.defaults.provider and routing.defaults.model in noodle.toml.",
		})
	}

	if _, err := lookPath("tmux"); err != nil {
		result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
			FieldPath: "runtime.tmux",
			Message:   "tmux is not available on PATH",
			Severity:  DiagnosticSeverityFatal,
			Fix:       "Install tmux and ensure it is available on PATH.",
		})
	}

	appendAgentDirDiagnostic := func(fieldPath, value string) {
		if value == "" {
			return
		}
		info, err := statPath(value)
		if err == nil && info.IsDir() {
			return
		}
		result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
			FieldPath: fieldPath,
			Message:   fmt.Sprintf("directory %q not found", value),
			Severity:  DiagnosticSeverityFatal,
			Fix:       fmt.Sprintf("Create %s or update %s to the correct directory.", value, fieldPath),
		})
	}
	appendAgentDirDiagnostic("agents.claude_dir", config.Agents.ClaudeDir)
	appendAgentDirDiagnostic("agents.codex_dir", config.Agents.CodexDir)

	for adapterName, adapter := range config.Adapters {
		if adapter.Skill == "" {
			result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
				FieldPath: fmt.Sprintf("adapters.%s.skill", adapterName),
				Message:   "skill is not configured",
				Severity:  DiagnosticSeverityRepairable,
				Fix:       fmt.Sprintf("Set adapters.%s.skill in noodle.toml.", adapterName),
			})
		}
		for action, command := range adapter.Scripts {
			if strings.TrimSpace(command) == "" {
				result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
					FieldPath: fmt.Sprintf("adapters.%s.scripts.%s", adapterName, action),
					Message:   "script command is empty",
					Severity:  DiagnosticSeverityRepairable,
					Fix:       fmt.Sprintf("Set adapters.%s.scripts.%s to an executable command.", adapterName, action),
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
