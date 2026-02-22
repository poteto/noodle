package worktree

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCheckCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		ctx     HookContext
		denied  bool
		snippet string // substring expected in denial reason
	}{
		// git worktree subcommands — all denied
		{"git worktree add", "git worktree add foo", HookContext{}, true, "worktree skill"},
		{"git worktree list", "git worktree list", HookContext{}, true, "worktree skill"},
		{"git worktree remove", "git worktree remove foo", HookContext{}, true, "worktree skill"},
		{"git worktree prune", "git worktree prune", HookContext{}, true, "worktree skill"},
		{"git worktree lock", "git worktree lock foo", HookContext{}, true, "worktree skill"},
		{"git worktree unlock", "git worktree unlock foo", HookContext{}, true, "worktree skill"},
		{"git worktree move", "git worktree move foo bar", HookContext{}, true, "worktree skill"},
		{"git worktree repair", "git worktree repair", HookContext{}, true, "worktree skill"},
		{"git worktree add in pipeline", "git worktree add foo && echo done", HookContext{}, true, "worktree skill"},
		{"git worktree with extra spaces", "git  worktree  add foo", HookContext{}, true, "worktree skill"},

		// cd into .worktrees/ — denied
		{"cd .worktrees/foo", "cd .worktrees/foo", HookContext{}, true, "NEVER cd"},
		{"cd with path", "cd /repo/.worktrees/branch", HookContext{}, true, "NEVER cd"},
		{"cd with relative path", "cd ../.worktrees/branch", HookContext{}, true, "NEVER cd"},

		// allowed commands
		{"plain git status", "git status", HookContext{}, false, ""},
		{"git checkout", "git checkout main", HookContext{}, false, ""},
		{"git branch", "git branch -a", HookContext{}, false, ""},
		{"go run worktree tool", "go run -C .agents/skills/worktree/scripts . create foo", HookContext{}, false, ""},
		{"echo git worktree", "echo git worktree add", HookContext{}, true, "worktree skill"}, // \b matches after space
		{"cd elsewhere", "cd /tmp/something", HookContext{}, false, ""},
		{"cd home", "cd ~", HookContext{}, false, ""},
		{"subshell cd into worktrees", "(cd .worktrees/foo && go test ./...)", HookContext{}, false, ""},
		{"git -C worktrees", "git -C .worktrees/foo status", HookContext{}, false, ""},
		{"grep worktree", "grep -r worktree .", HookContext{}, false, ""},
		{"empty command", "", HookContext{}, false, ""},

		// contextual policy: cook + primary checkout
		{"cook primary denies write", "git commit -m test", HookContext{Cook: true, InPrimaryCheckout: true}, true, "Skill(worktree)"},
		{"cook primary allows read", "git status", HookContext{Cook: true, InPrimaryCheckout: true}, false, ""},
		{"cook linked allows write", "git commit -m test", HookContext{Cook: true, InPrimaryCheckout: false}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			denial := CheckCommandWithContext(tt.cmd, tt.ctx)
			if tt.denied && denial == nil {
				t.Errorf("expected denial for %q, got nil", tt.cmd)
			}
			if !tt.denied && denial != nil {
				t.Errorf("expected allow for %q, got denial: %s", tt.cmd, denial.Reason)
			}
			if tt.denied && denial != nil && !strings.Contains(denial.Reason, tt.snippet) {
				t.Errorf("denial reason %q missing expected snippet %q", denial.Reason, tt.snippet)
			}
		})
	}
}

func TestIsPrimaryCheckout(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"primary repo .git", "/repo/.git", true},
		{"linked checkout gitdir", "/repo/.git/worktrees/feature-x", false},
		{"trailing slash", "/repo/.git/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPrimaryCheckout(tt.path); got != tt.want {
				t.Fatalf("IsPrimaryCheckout(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRunHook(t *testing.T) {
	t.Run("denies git worktree add", func(t *testing.T) {
		input := `{"tool_name":"Bash","tool_input":{"command":"git worktree add foo"}}`
		var buf bytes.Buffer
		err := RunHook(strings.NewReader(input), &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.Len() == 0 {
			t.Fatal("expected deny output, got nothing")
		}

		var out hookOutput
		if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
			t.Fatalf("invalid JSON output: %v", err)
		}
		if out.HookSpecificOutput.PermissionDecision != "deny" {
			t.Errorf("expected deny, got %q", out.HookSpecificOutput.PermissionDecision)
		}
		if out.HookSpecificOutput.HookEventName != "PreToolUse" {
			t.Errorf("expected PreToolUse, got %q", out.HookSpecificOutput.HookEventName)
		}
	})

	t.Run("denies cd into worktrees", func(t *testing.T) {
		input := `{"tool_name":"Bash","tool_input":{"command":"cd .worktrees/branch"}}`
		var buf bytes.Buffer
		err := RunHook(strings.NewReader(input), &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.Len() == 0 {
			t.Fatal("expected deny output, got nothing")
		}

		var out hookOutput
		if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
			t.Fatalf("invalid JSON output: %v", err)
		}
		if out.HookSpecificOutput.PermissionDecision != "deny" {
			t.Errorf("expected deny, got %q", out.HookSpecificOutput.PermissionDecision)
		}
	})

	t.Run("allows safe command", func(t *testing.T) {
		input := `{"tool_name":"Bash","tool_input":{"command":"git status"}}`
		var buf bytes.Buffer
		err := RunHook(strings.NewReader(input), &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.Len() != 0 {
			t.Errorf("expected no output for allowed command, got: %s", buf.String())
		}
	})

	t.Run("denies cook writes from primary checkout", func(t *testing.T) {
		t.Setenv("COOK", "1")

		prevDetect := DetectPrimaryCheckoutFunc
		t.Cleanup(func() {
			DetectPrimaryCheckoutFunc = prevDetect
		})
		DetectPrimaryCheckoutFunc = func() (bool, error) {
			return true, nil
		}

		input := `{"tool_name":"Bash","tool_input":{"command":"git commit -m test"}}`
		var buf bytes.Buffer
		err := RunHook(strings.NewReader(input), &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.Len() == 0 {
			t.Fatal("expected deny output, got nothing")
		}

		var out hookOutput
		if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
			t.Fatalf("invalid JSON output: %v", err)
		}
		if out.HookSpecificOutput.PermissionDecision != "deny" {
			t.Errorf("expected deny, got %q", out.HookSpecificOutput.PermissionDecision)
		}
		if !strings.Contains(out.HookSpecificOutput.PermissionDecisionReason, "Skill(worktree)") {
			t.Errorf("missing Skill(worktree) guidance in reason: %q", out.HookSpecificOutput.PermissionDecisionReason)
		}
	})

	t.Run("ignores non-Bash tools", func(t *testing.T) {
		input := `{"tool_name":"Read","tool_input":{"file_path":"/foo"}}`
		var buf bytes.Buffer
		err := RunHook(strings.NewReader(input), &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.Len() != 0 {
			t.Errorf("expected no output for non-Bash tool, got: %s", buf.String())
		}
	})
}
