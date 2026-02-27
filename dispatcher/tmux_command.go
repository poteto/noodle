package dispatcher

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/shellx"
)

const codexSkillRefsLimitBytes = 50 * 1024

func buildProviderCommand(
	req DispatchRequest,
	promptFile string,
	agentBinary string,
	systemPrompt string,
	stderrFile string,
	extraArgs []string,
) string {
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "codex" {
		return buildCodexCommand(req, promptFile, agentBinary, stderrFile, extraArgs)
	}
	return buildClaudeCommand(req, promptFile, agentBinary, systemPrompt, stderrFile, extraArgs)
}

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

func buildClaudeCommand(req DispatchRequest, promptFile, agentBinary, systemPrompt, stderrFile string, extraArgs []string) string {
	args := append([]string{agentBinary}, claudeBaseArgs(req, systemPrompt)...)
	args = append(args, extraArgs...)
	command := shellCommandWithInput(args, promptFile, true)
	if strings.TrimSpace(stderrFile) != "" {
		command += " 2> " + shellx.Quote(stderrFile)
	}
	return command
}

func buildCodexCommand(req DispatchRequest, promptFile, agentBinary, stderrFile string, extraArgs []string) string {
	args := append([]string{agentBinary}, codexBaseArgs(req)...)
	args = append(args, extraArgs...)
	// Codex writes non-JSON progress banners to stderr even with --json.
	// Keep stderr out of the stamp pipeline so parser input stays valid NDJSON.
	command := shellCommandWithInput(args, promptFile, false)
	if strings.TrimSpace(stderrFile) != "" {
		command += " 2> " + shellx.Quote(stderrFile)
	}
	return command
}

func buildPipelineCommand(providerCmd, noodleBin, stampedPath, canonicalPath string) string {
	return fmt.Sprintf(
		"%s | %s stamp --output %s --events %s",
		providerCmd,
		shellx.Quote(noodleBin),
		shellx.Quote(stampedPath),
		shellx.Quote(canonicalPath),
	)
}

func shellCommandWithInput(args []string, inputPath string, includeStderr bool) string {
	var builder strings.Builder
	for i, arg := range args {
		if i > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(shellx.Quote(arg))
	}
	builder.WriteString(" < ")
	builder.WriteString(shellx.Quote(inputPath))
	if includeStderr {
		builder.WriteString(" 2>&1")
	}
	return builder.String()
}

func tmuxSessionName(sessionID, fallbackName string) string {
	name := strings.TrimSpace(sessionID)
	if name == "" {
		name = shellx.SanitizeToken(fallbackName, "cook")
	}
	return "noodle-" + shellx.SanitizeToken(name, "cook")
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
