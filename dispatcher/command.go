package dispatcher

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/shellx"
)

const codexSkillRefsLimitBytes = 50 * 1024

// claudeBaseArgs returns the canonical Claude CLI flags for a dispatch request.
// Callers prepend the binary path and append site-specific extras.
func claudeBaseArgs(req DispatchRequest, systemPrompt string) []string {
	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--permission-mode", "bypassPermissions",
	}
	if req.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(req.MaxTurns))
	}
	if req.BudgetCap > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", req.BudgetCap))
	}
	if strings.TrimSpace(req.Model) != "" {
		args = append(args, "--model", req.Model)
	}
	if strings.TrimSpace(req.ReasoningLevel) != "" {
		args = append(args, "--effort", req.ReasoningLevel)
	}
	if strings.TrimSpace(systemPrompt) != "" {
		args = append(args, "--append-system-prompt", systemPrompt)
	}
	return args
}

// codexBaseArgs returns the canonical Codex CLI flags for a dispatch request.
// Callers prepend the binary path and append site-specific extras.
func codexBaseArgs(req DispatchRequest) []string {
	args := []string{
		"exec",
		"--dangerously-bypass-approvals-and-sandbox",
		"--skip-git-repo-check",
		"--json",
	}
	if strings.TrimSpace(req.Model) != "" {
		args = append(args, "--model", req.Model)
	}
	return args
}

func generateSessionID(name string) (string, error) {
	randBytes := make([]byte, 3)
	if _, err := rand.Read(randBytes); err != nil {
		return "", err
	}
	base := shellx.SanitizeToken(name, "cook")
	timestamp := time.Now().UTC().Format("20060102-150405")
	return fmt.Sprintf("%s-%s-%x", base, timestamp, randBytes), nil
}

func sessionPaths(runtimeDir, sessionID string) (sessionDir, promptPath, stampedPath, canonicalPath, stderrPath string) {
	sessionDir = filepath.Join(runtimeDir, "sessions", sessionID)
	promptPath = filepath.Join(sessionDir, "prompt.txt")
	stampedPath = filepath.Join(sessionDir, "raw.ndjson")
	canonicalPath = filepath.Join(sessionDir, "canonical.ndjson")
	stderrPath = filepath.Join(sessionDir, "stderr.log")
	return
}

func inputPath(sessionDir string) string {
	return filepath.Join(sessionDir, "input.txt")
}

// resolveTemplateVars replaces {{key}} placeholders with verbatim values.
// No shell quoting — the template author controls quoting in their template.
func resolveTemplateVars(tmpl string, vars map[string]string) string {
	result := tmpl
	for key, value := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}

func expandHomePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "~") {
		return path
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		return filepath.Join(homeDir, strings.TrimPrefix(strings.TrimPrefix(path, "~/"), "~\\"))
	}
	return path
}

func buildDispatchEnv(req DispatchRequest) []string {
	env := make([]string, 0, len(os.Environ())+len(req.EnvVars)+4)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.EqualFold(key, "CLAUDECODE") {
			continue
		}
		env = append(env, entry)
	}

	env = append(env, "NOODLE_WORKTREE="+req.WorktreePath)
	env = append(env, "NOODLE_PROVIDER="+req.Provider)
	env = append(env, "NOODLE_MODEL="+req.Model)
	if req.ReasoningLevel != "" {
		env = append(env, "NOODLE_REASONING_LEVEL="+req.ReasoningLevel)
	}
	for key, value := range req.EnvVars {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		env = append(env, key+"="+value)
	}
	return env
}

func composePrompts(provider, requestPrompt, preamble, skillSystemPrompt string) (systemPrompt, finalPrompt string) {
	finalPrompt = requestPrompt
	fullSystemPrompt := joinNonEmpty(preamble, skillSystemPrompt)
	if strings.EqualFold(provider, "claude") {
		systemPrompt = fullSystemPrompt
		return systemPrompt, finalPrompt
	}
	// Codex loads skills natively via Skill(name) — only inline the preamble.
	if strings.EqualFold(provider, "codex") && strings.TrimSpace(preamble) != "" {
		finalPrompt = requestPrompt + "\n\n---\n\n" + preamble
	}
	return systemPrompt, finalPrompt
}

func joinNonEmpty(parts ...string) string {
	var out []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n\n")
}
