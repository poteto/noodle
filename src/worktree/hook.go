package worktree

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// HookDenial holds the reason a command was denied.
type HookDenial struct {
	Reason string
}

var (
	reGitWorktree = regexp.MustCompile(`\bgit\s+worktree\s+(add|list|remove|prune|lock|unlock|move|repair)\b`)
	reCdWorktrees = regexp.MustCompile(`^cd\s+.*\.worktrees/`)
	reLikelyWrite = regexp.MustCompile(
		`(?i)^\s*(git\s+(add|am|apply|checkout|cherry-pick|clean|commit|merge|mv|rebase|reset|restore|revert|rm|stash|switch)\b|pnpm\s+(add|install|remove|up|update)\b|npm\s+(ci|install|uninstall|update)\b|yarn\s+(add|install|remove|up)\b|cargo\s+(add|rm|remove)\b|go\s+(generate|mod)\b|mkdir\b|mv\b|rm\b|touch\b|cp\b)`,
	)
)

// HookContext captures session and checkout mode for command validation.
type HookContext struct {
	Cook              bool
	InPrimaryCheckout bool
}

// DetectPrimaryCheckoutFunc is the function used to detect primary checkout.
// It is a variable to allow test injection.
var DetectPrimaryCheckoutFunc = DetectPrimaryCheckout

// IsPrimaryCheckout reports whether absGitDir points at the repo's primary checkout.
func IsPrimaryCheckout(absGitDir string) bool {
	clean := filepath.Clean(strings.TrimSpace(absGitDir))
	return filepath.Base(clean) == ".git"
}

// DetectPrimaryCheckout resolves the current git dir and classifies checkout type.
func DetectPrimaryCheckout() (bool, error) {
	out, err := exec.Command("git", "rev-parse", "--absolute-git-dir").Output()
	if err != nil {
		return false, err
	}
	return IsPrimaryCheckout(string(out)), nil
}

func isLikelyWriteCommand(cmd string) bool {
	return reLikelyWrite.MatchString(cmd)
}

// CheckCommand inspects a Bash command string and returns a denial if it
// matches a dangerous worktree pattern, or nil if the command is allowed.
func CheckCommand(cmd string) *HookDenial {
	ctx := HookContext{}
	if os.Getenv("COOK") == "1" {
		ctx.Cook = true
		inPrimaryCheckout, err := DetectPrimaryCheckoutFunc()
		if err == nil {
			ctx.InPrimaryCheckout = inPrimaryCheckout
		}
	}
	return CheckCommandWithContext(cmd, ctx)
}

// CheckCommandWithContext applies static + contextual worktree safety checks.
func CheckCommandWithContext(cmd string, ctx HookContext) *HookDenial {
	if reGitWorktree.MatchString(cmd) {
		return &HookDenial{
			Reason: "Raw `git worktree` commands are forbidden. Use the worktree skill instead — it enforces CWD safety, correct sequencing, stash/rebase handling, and dep reinstall. Invoke via Skill(worktree).",
		}
	}
	if reCdWorktrees.MatchString(cmd) {
		return &HookDenial{
			Reason: "NEVER cd into .worktrees/ — when the worktree is removed, the CWD is deleted and the shell dies permanently. Use subshells (cd .worktrees/... && cmd), git -C, go -C, or the worktree skill exec command instead.",
		}
	}
	if ctx.Cook && ctx.InPrimaryCheckout && isLikelyWriteCommand(cmd) {
		return &HookDenial{
			Reason: "Cook write commands from the primary checkout are blocked. Use Skill(worktree) to create/use a linked worktree (usually `.worktrees/<name>`) for autonomous/significant edits. Primary checkout is only for interactive single-agent small changes.",
		}
	}
	return nil
}

// hookInput mirrors the PreToolUse JSON that Claude Code sends to hooks.
type hookInput struct {
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

// hookOutput is the deny response format expected by Claude Code hooks.
type hookOutput struct {
	HookSpecificOutput struct {
		HookEventName            string `json:"hookEventName"`
		PermissionDecision       string `json:"permissionDecision"`
		PermissionDecisionReason string `json:"permissionDecisionReason"`
	} `json:"hookSpecificOutput"`
}

// RunHook reads a PreToolUse JSON payload from r, checks the command, and
// writes a deny decision to w if the command is blocked. Returns nil on
// success (including when the command is allowed and nothing is written).
func RunHook(r io.Reader, w io.Writer) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	var input hookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	if input.ToolName != "Bash" {
		return nil
	}

	denial := CheckCommand(input.ToolInput.Command)
	if denial == nil {
		return nil
	}

	var out hookOutput
	out.HookSpecificOutput.HookEventName = "PreToolUse"
	out.HookSpecificOutput.PermissionDecision = "deny"
	out.HookSpecificOutput.PermissionDecisionReason = denial.Reason

	return json.NewEncoder(w).Encode(out)
}
