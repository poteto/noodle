package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/poteto/noodle/internal/state"
)

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

	if !metadata.IsDefined("routing", "defaults", "provider") {
		config.Routing.Defaults.Provider = "claude"
	}
	if !metadata.IsDefined("routing", "defaults", "model") {
		config.Routing.Defaults.Model = "claude-opus-4-6"
	}
	if config.Routing.Tags == nil {
		config.Routing.Tags = map[string]ModelPolicy{}
	}

	if !metadata.IsDefined("mode") {
		config.Mode = string(state.RunModeAuto)
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
	if !metadata.IsDefined("concurrency", "max_completion_overflow") {
		config.Concurrency.MaxCompletionOverflow = 1024
	}
	if !metadata.IsDefined("concurrency", "merge_backpressure_threshold") {
		config.Concurrency.MergeBackpressureThreshold = 128
	}
	if !metadata.IsDefined("concurrency", "shutdown_timeout") {
		config.Concurrency.ShutdownTimeout = "30s"
	}

	config.Runtime.spritesDefined = metadata.IsDefined("runtime", "sprites")
	config.Runtime.cursorDefined = metadata.IsDefined("runtime", "cursor")
	if !metadata.IsDefined("runtime", "default") {
		config.Runtime.Default = "process"
	}
	if !metadata.IsDefined("runtime", "process", "max_concurrent") {
		config.Runtime.Process.MaxConcurrent = 4
	}
	if !metadata.IsDefined("runtime", "sprites", "max_concurrent") {
		config.Runtime.Sprites.MaxConcurrent = 50
	}
	if !metadata.IsDefined("runtime", "cursor", "max_concurrent") {
		config.Runtime.Cursor.MaxConcurrent = 10
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

func validateParsedValues(config Config) error {
	switch config.Mode {
	case string(state.RunModeAuto), string(state.RunModeSupervised), string(state.RunModeManual):
	case "":
		// treated as default in applyDefaults
	default:
		return fmt.Errorf(
			"mode: unsupported value %q (supported: auto, supervised, manual)",
			config.Mode,
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

	if config.Recovery.MaxRetries < 0 {
		return fmt.Errorf("recovery.max_retries: must be greater than or equal to 0")
	}
	if config.Concurrency.MaxCooks <= 0 {
		return fmt.Errorf("concurrency.max_cooks: must be greater than 0")
	}
	if config.Concurrency.MaxCompletionOverflow <= 0 {
		return fmt.Errorf("concurrency.max_completion_overflow: must be greater than 0")
	}
	if config.Concurrency.MergeBackpressureThreshold <= 0 {
		return fmt.Errorf("concurrency.merge_backpressure_threshold: must be greater than 0")
	}
	if err := validatePositiveDuration("concurrency.shutdown_timeout", config.Concurrency.ShutdownTimeout); err != nil {
		return err
	}

	if config.Runtime.Process.MaxConcurrent < 0 {
		return fmt.Errorf("runtime.process.max_concurrent: must be greater than or equal to 0")
	}
	if config.Runtime.Sprites.MaxConcurrent < 0 {
		return fmt.Errorf("runtime.sprites.max_concurrent: must be greater than or equal to 0")
	}
	if config.Runtime.Cursor.MaxConcurrent < 0 {
		return fmt.Errorf("runtime.cursor.max_concurrent: must be greater than or equal to 0")
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
