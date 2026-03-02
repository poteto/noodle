package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/stringx"
)

const (
	defaultRoutingProvider = "claude"
	defaultRoutingModel    = "claude-opus-4-6"
	defaultMode            = string(state.RunModeAuto)
	defaultMaxConcurrency  = 4
	defaultRuntimeName     = "process"
	defaultServerPort      = 3000
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

	config, parseDiagnostics, err := parseWithDiagnostics(data)
	if err != nil {
		return Config{}, ValidationResult{}, err
	}
	validation := Validate(config)
	validation.Diagnostics = append(parseDiagnostics, validation.Diagnostics...)
	return config, validation, nil
}

// Parse parses a .noodle.toml payload and applies omitted-field defaults.
func Parse(data []byte) (Config, error) {
	config, _, err := parseWithDiagnostics(data)
	return config, err
}

func parseWithDiagnostics(data []byte) (Config, []ConfigDiagnostic, error) {
	var config Config
	metadata, err := toml.Decode(string(data), &config)
	if err != nil {
		return Config{}, nil, fmt.Errorf("parse .noodle.toml: %w", err)
	}

	diagnostics := make([]ConfigDiagnostic, 0)
	applyDefaultsFromMetadata(&config, metadata, &diagnostics)
	appendUndecodedDiagnostics(metadata, &diagnostics)
	normalizeParsedValues(&config, metadata, &diagnostics)
	return config, diagnostics, nil
}

func applyDefaultsFromMetadata(config *Config, metadata toml.MetaData, diagnostics *[]ConfigDiagnostic) {
	if !metadata.IsDefined("skills", "paths") || len(config.Skills.Paths) == 0 {
		if metadata.IsDefined("skills", "paths") && len(config.Skills.Paths) == 0 {
			appendNormalizedValueDiagnostic(diagnostics,
				"skills.paths",
				"skills.paths is empty; using default skill paths",
			)
		}
		config.Skills.Paths = defaultSkillPaths()
	}

	if !metadata.IsDefined("routing", "defaults", "provider") {
		config.Routing.Defaults.Provider = defaultRoutingProvider
	}
	if !metadata.IsDefined("routing", "defaults", "model") {
		config.Routing.Defaults.Model = defaultRoutingModel
	}

	if !metadata.IsDefined("mode") {
		config.Mode = defaultMode
	}

	if !metadata.IsDefined("concurrency", "max_concurrency") {
		config.Concurrency.MaxConcurrency = defaultMaxConcurrency
	}

	config.Runtime.spritesDefined = metadata.IsDefined("runtime", "sprites")
	config.Runtime.cursorDefined = metadata.IsDefined("runtime", "cursor")
	if !metadata.IsDefined("runtime", "default") {
		config.Runtime.Default = defaultRuntimeName
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
		config.Server.Port = defaultServerPort
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

func appendUndecodedDiagnostics(metadata toml.MetaData, diagnostics *[]ConfigDiagnostic) {
	for _, key := range metadata.Undecoded() {
		fieldPath := keyPath(key)
		if fieldPath == "" {
			continue
		}

		if message, ok := removedFieldWarning(fieldPath); ok {
			*diagnostics = append(*diagnostics, ConfigDiagnostic{
				FieldPath: fieldPath,
				Message:   message,
				Severity:  DiagnosticSeverityWarning,
				Code:      DiagnosticCodeConfigFieldRemoved,
			})
			continue
		}

		*diagnostics = append(*diagnostics, ConfigDiagnostic{
			FieldPath: fieldPath,
			Message:   fmt.Sprintf("unknown config field %q ignored", fieldPath),
			Severity:  DiagnosticSeverityWarning,
			Code:      DiagnosticCodeConfigFieldUnknown,
		})
	}
}

func keyPath(key toml.Key) string {
	if len(key) == 0 {
		return ""
	}
	parts := make([]string, 0, len(key))
	for _, raw := range key {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return ""
		}
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, ".")
}

func removedFieldWarning(fieldPath string) (string, bool) {
	switch {
	case hasFieldPrefix(fieldPath, "recovery"):
		return "recovery config was removed; field ignored and defaults applied", true
	case hasFieldPrefix(fieldPath, "monitor"):
		return "monitor config was removed; field ignored and defaults applied", true
	case hasFieldPrefix(fieldPath, "routing.tags"):
		return "routing.tags was removed; scheduler controls per-stage routing", true
	case hasFieldPrefix(fieldPath, "concurrency.max_cooks"):
		return "concurrency.max_cooks was removed; use concurrency.max_concurrency", true
	case hasFieldPrefix(fieldPath, "concurrency.max_completion_overflow"):
		return "concurrency.max_completion_overflow was removed; runtime uses a fixed internal default", true
	case hasFieldPrefix(fieldPath, "concurrency.merge_backpressure_threshold"):
		return "concurrency.merge_backpressure_threshold was removed; runtime uses a fixed internal default", true
	case hasFieldPrefix(fieldPath, "concurrency.shutdown_timeout"):
		return "concurrency.shutdown_timeout was removed; shutdown uses a fixed 2s deadline", true
	default:
		return "", false
	}
}

func hasFieldPrefix(fieldPath, prefix string) bool {
	return fieldPath == prefix || strings.HasPrefix(fieldPath, prefix+".")
}

func appendNormalizedValueDiagnostic(diagnostics *[]ConfigDiagnostic, fieldPath, message string) {
	*diagnostics = append(*diagnostics, ConfigDiagnostic{
		FieldPath: fieldPath,
		Message:   message,
		Severity:  DiagnosticSeverityWarning,
		Code:      DiagnosticCodeConfigValueNormalized,
	})
}

func normalizeParsedValues(config *Config, metadata toml.MetaData, diagnostics *[]ConfigDiagnostic) {
	mode := stringx.Normalize(config.Mode)
	switch mode {
	case "":
		config.Mode = defaultMode
	case string(state.RunModeAuto), string(state.RunModeSupervised), string(state.RunModeManual):
		config.Mode = mode
	default:
		appendNormalizedValueDiagnostic(diagnostics, "mode",
			fmt.Sprintf("mode %q unsupported; using %q", config.Mode, defaultMode),
		)
		config.Mode = defaultMode
	}

	provider := stringx.Normalize(config.Routing.Defaults.Provider)
	switch provider {
	case "":
		if metadata.IsDefined("routing", "defaults", "provider") {
			appendNormalizedValueDiagnostic(diagnostics, "routing.defaults.provider",
				fmt.Sprintf("provider not set; using %q", defaultRoutingProvider),
			)
		}
		config.Routing.Defaults.Provider = defaultRoutingProvider
	case "claude", "codex":
		config.Routing.Defaults.Provider = provider
	default:
		appendNormalizedValueDiagnostic(diagnostics, "routing.defaults.provider",
			fmt.Sprintf("provider %q unsupported; using %q", config.Routing.Defaults.Provider, defaultRoutingProvider),
		)
		config.Routing.Defaults.Provider = defaultRoutingProvider
	}

	if strings.TrimSpace(config.Routing.Defaults.Model) == "" {
		if metadata.IsDefined("routing", "defaults", "model") {
			appendNormalizedValueDiagnostic(diagnostics, "routing.defaults.model",
				fmt.Sprintf("model not set; using %q", defaultRoutingModel),
			)
		}
		config.Routing.Defaults.Model = defaultRoutingModel
	}

	if config.Concurrency.MaxConcurrency <= 0 {
		if metadata.IsDefined("concurrency", "max_concurrency") {
			appendNormalizedValueDiagnostic(diagnostics, "concurrency.max_concurrency",
				fmt.Sprintf("max_concurrency=%d is invalid; using %d", config.Concurrency.MaxConcurrency, defaultMaxConcurrency),
			)
		}
		config.Concurrency.MaxConcurrency = defaultMaxConcurrency
	}

	if strings.TrimSpace(config.Runtime.Default) == "" {
		config.Runtime.Default = defaultRuntimeName
	} else {
		config.Runtime.Default = stringx.Normalize(config.Runtime.Default)
		switch config.Runtime.Default {
		case "process", "sprites", "cursor":
		default:
			appendNormalizedValueDiagnostic(diagnostics, "runtime.default",
				fmt.Sprintf("runtime.default %q unsupported; using %q", config.Runtime.Default, defaultRuntimeName),
			)
			config.Runtime.Default = defaultRuntimeName
		}
	}

	normalizeRuntimeMaxConcurrent(
		"runtime.process.max_concurrent",
		metadata.IsDefined("runtime", "process", "max_concurrent"),
		&config.Runtime.Process.MaxConcurrent,
		4,
		diagnostics,
	)
	normalizeRuntimeMaxConcurrent(
		"runtime.sprites.max_concurrent",
		metadata.IsDefined("runtime", "sprites", "max_concurrent"),
		&config.Runtime.Sprites.MaxConcurrent,
		50,
		diagnostics,
	)
	normalizeRuntimeMaxConcurrent(
		"runtime.cursor.max_concurrent",
		metadata.IsDefined("runtime", "cursor", "max_concurrent"),
		&config.Runtime.Cursor.MaxConcurrent,
		10,
		diagnostics,
	)

	if config.Server.Port < 0 || config.Server.Port > 65535 {
		appendNormalizedValueDiagnostic(diagnostics, "server.port",
			fmt.Sprintf("server.port=%d is invalid; using %d", config.Server.Port, defaultServerPort),
		)
		config.Server.Port = defaultServerPort
	}
}

func normalizeRuntimeMaxConcurrent(
	fieldPath string,
	isDefined bool,
	value *int,
	fallback int,
	diagnostics *[]ConfigDiagnostic,
) {
	if value == nil || *value >= 0 {
		return
	}
	if isDefined {
		appendNormalizedValueDiagnostic(diagnostics, fieldPath,
			fmt.Sprintf("%s=%d is invalid; using %d", fieldPath, *value, fallback),
		)
	}
	*value = fallback
}
