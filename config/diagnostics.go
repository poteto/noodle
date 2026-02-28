package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var statPath = os.Stat

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

	if rd := strings.ToLower(strings.TrimSpace(config.Runtime.Default)); rd != "" {
		switch rd {
		case "process", "sprites", "cursor":
			// Known runtimes — ok.
		default:
			result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
				FieldPath: "runtime.default",
				Message:   fmt.Sprintf("unknown runtime %q (valid: process, sprites, cursor)", config.Runtime.Default),
				Severity:  DiagnosticSeverityWarning,
				Fix:       "Set runtime.default to \"process\", \"sprites\", or \"cursor\" in .noodle.toml.",
				Code:      DiagnosticCodeRuntimeDefaultUnknown,
			})
		}
	}

	// No PATH check needed for process runtime — it uses direct child processes.

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

	if _, hasBacklog := config.Adapters["backlog"]; !hasBacklog {
		result.Diagnostics = append(result.Diagnostics, ConfigDiagnostic{
			FieldPath: "adapters.backlog",
			Message:   "no backlog adapter configured",
			Severity:  DiagnosticSeverityRepairable,
			Fix:       "Add [adapters.backlog] to .noodle.toml with adapter scripts.",
			Code:      DiagnosticCodeNoBacklogAdapter,
		})
	}

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

func isKnownProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "claude", "codex":
		return true
	}
	return false
}
