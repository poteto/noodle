package spawner

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const codexSkillRefsLimitBytes = 50 * 1024

func buildProviderCommand(
	req SpawnRequest,
	promptFile string,
	agentBinary string,
	systemPrompt string,
	stderrFile string,
) string {
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "codex" {
		return buildCodexCommand(req, promptFile, agentBinary, stderrFile)
	}
	return buildClaudeCommand(req, promptFile, agentBinary, systemPrompt, stderrFile)
}

func buildClaudeCommand(req SpawnRequest, promptFile, agentBinary, systemPrompt, stderrFile string) string {
	args := []string{
		agentBinary,
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--permission-mode", "dontAsk",
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
	command := shellCommandWithInput(args, promptFile, true)
	if strings.TrimSpace(stderrFile) != "" {
		command += " 2> " + shellQuote(stderrFile)
	}
	return command
}

func buildCodexCommand(req SpawnRequest, promptFile, agentBinary, stderrFile string) string {
	args := []string{
		agentBinary,
		"--ask-for-approval", "never",
		"exec",
		"--skip-git-repo-check",
		"--sandbox", "workspace-write",
		"--json",
	}
	if strings.TrimSpace(req.Model) != "" {
		args = append(args, "--model", req.Model)
	}
	// Codex writes non-JSON progress banners to stderr even with --json.
	// Keep stderr out of the stamp pipeline so parser input stays valid NDJSON.
	command := shellCommandWithInput(args, promptFile, false)
	if strings.TrimSpace(stderrFile) != "" {
		command += " 2> " + shellQuote(stderrFile)
	}
	return command
}

func buildPipelineCommand(providerCmd, noodleBin, stampedPath, canonicalPath string) string {
	return fmt.Sprintf(
		"%s | %s stamp --output %s --events %s",
		providerCmd,
		shellQuote(noodleBin),
		shellQuote(stampedPath),
		shellQuote(canonicalPath),
	)
}

func shellCommandWithInput(args []string, inputPath string, includeStderr bool) string {
	var builder strings.Builder
	for i, arg := range args {
		if i > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(shellQuote(arg))
	}
	builder.WriteString(" < ")
	builder.WriteString(shellQuote(inputPath))
	if includeStderr {
		builder.WriteString(" 2>&1")
	}
	return builder.String()
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func tmuxSessionName(sessionID, fallbackName string) string {
	name := strings.TrimSpace(sessionID)
	if name == "" {
		name = sanitizeToken(fallbackName, "cook")
	}
	return "noodle-" + sanitizeToken(name, "cook")
}

func sanitizeToken(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = fallback
	}
	var out strings.Builder
	lastHyphen := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			out.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			out.WriteByte('-')
			lastHyphen = true
		}
	}
	result := strings.Trim(out.String(), "-")
	if result == "" {
		result = fallback
	}
	if len(result) > 48 {
		result = strings.Trim(result[:48], "-")
	}
	if result == "" {
		return fallback
	}
	return result
}

func generateSessionID(name string) (string, error) {
	randBytes := make([]byte, 3)
	if _, err := rand.Read(randBytes); err != nil {
		return "", err
	}
	base := sanitizeToken(name, "cook")
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
