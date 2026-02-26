package dispatcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	sprites "github.com/superfly/sprites-go"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/internal/shellx"
	"github.com/poteto/noodle/skill"
)

const spriteWorkDir = "/work/repo"

// SpritesDispatcherConfig configures a sprites dispatcher.
type SpritesDispatcherConfig struct {
	ProjectDir    string
	RuntimeDir    string
	NoodleBin     string
	SkillResolver skill.Resolver
	SpriteName    string
	Token         string
	GitToken      string // GitHub token for repo access on the sprite
}

// SpritesDispatcher dispatches sessions on remote Sprites VMs via the sprites-go SDK.
type SpritesDispatcher struct {
	projectDir    string
	runtimeDir    string
	noodleBin     string
	skillResolver skill.Resolver
	spriteName    string
	token         string
	gitToken      string

	// newSprite creates a Sprite handle. Injected for testing.
	newSprite func(name string) spriteHandle
}

// spriteHandle is the subset of sprites.Sprite we use, for testability.
type spriteHandle interface {
	CommandContext(ctx context.Context, name string, args ...string) *sprites.Cmd
}

// NewSpritesDispatcher constructs a sprites dispatcher.
func NewSpritesDispatcher(config SpritesDispatcherConfig) *SpritesDispatcher {
	client := sprites.New(config.Token)
	return &SpritesDispatcher{
		projectDir:    strings.TrimSpace(config.ProjectDir),
		runtimeDir:    strings.TrimSpace(config.RuntimeDir),
		noodleBin:     strings.TrimSpace(config.NoodleBin),
		skillResolver: config.SkillResolver,
		spriteName:    strings.TrimSpace(config.SpriteName),
		token:         config.Token,
		gitToken:      config.GitToken,
		newSprite: func(name string) spriteHandle {
			return client.Sprite(name)
		},
	}
}

func (d *SpritesDispatcher) Dispatch(ctx context.Context, req DispatchRequest) (Session, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	reqRuntime := normalizeRuntime(req.Runtime)
	if reqRuntime != "sprites" {
		return nil, fmt.Errorf("runtime %q not supported by sprites dispatcher", reqRuntime)
	}
	req.Runtime = reqRuntime

	if d.runtimeDir == "" {
		return nil, fmt.Errorf("runtime directory not set")
	}
	if d.spriteName == "" {
		return nil, fmt.Errorf("sprite name not set")
	}

	sessionID, err := generateSessionID(req.Name)
	if err != nil {
		return nil, fmt.Errorf("generate session ID: %w", err)
	}
	sessionDir, promptPath, stampedPath, canonicalPath, _ := sessionPaths(d.runtimeDir, sessionID)

	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}
	eventWriter, err := event.NewEventWriter(d.runtimeDir, sessionID)
	if err != nil {
		return nil, fmt.Errorf("create event writer: %w", err)
	}

	var skillBundle loadedSkill
	if sp := strings.TrimSpace(req.SystemPrompt); sp != "" {
		skillBundle = loadedSkill{SystemPrompt: sp}
	} else if req.TaskKey == "execute" && req.DomainSkill != "" {
		skillBundle, err = loadExecuteBundle(d.skillResolver, req.Provider, req.Skill, req.DomainSkill)
	} else {
		skillBundle, err = loadSkillBundle(d.skillResolver, req.Provider, req.Skill)
	}
	if err != nil {
		return nil, err
	}

	preamble := buildSessionPreamble()
	systemPrompt, composedPrompt := composePrompts(req.Provider, req.Prompt, preamble, skillBundle.SystemPrompt)

	if err := os.WriteFile(promptPath, []byte(req.Prompt), 0o644); err != nil {
		return nil, fmt.Errorf("write prompt file: %w", err)
	}

	// input.txt: full composed prompt piped to agent stdin (includes
	// inlined skill content for providers without system prompt support).
	inputFile := promptPath
	if composedPrompt != req.Prompt {
		inputFile = filepath.Join(sessionDir, "input.txt")
		if err := os.WriteFile(inputFile, []byte(composedPrompt), 0o644); err != nil {
			return nil, fmt.Errorf("write input file: %w", err)
		}
	}

	sprite := d.newSprite(d.spriteName)

	// Push worktree branch to GitHub, then clone on the sprite.
	remoteURL := gitRemoteURL(req.WorktreePath)
	if remoteURL == "" {
		return nil, fmt.Errorf("git remote URL not found for %s", req.WorktreePath)
	}
	branch := gitCurrentBranch(req.WorktreePath)
	if branch == "" {
		return nil, fmt.Errorf("git branch not found for %s", req.WorktreePath)
	}
	if err := pushWorktreeBranch(ctx, req.WorktreePath, branch); err != nil {
		return nil, fmt.Errorf("push worktree branch: %w", err)
	}
	cloneURL := remoteURL
	if d.gitToken != "" {
		cloneURL = authenticatedRemoteURL(remoteURL, d.gitToken)
	}
	if err := cloneOnSprite(ctx, sprite, cloneURL, branch); err != nil {
		msg := err.Error()
		if d.gitToken != "" {
			msg = strings.ReplaceAll(msg, d.gitToken, "REDACTED")
		}
		return nil, fmt.Errorf("clone on sprite: %s", msg)
	}

	// Upload prompt file to sprite.
	if err := uploadFileToSprite(ctx, sprite, inputFile, "/work/prompt.txt"); err != nil {
		return nil, fmt.Errorf("upload prompt to sprite: %w", err)
	}

	// Build the provider command to run on the sprite.
	var agentArgs []string
	if strings.EqualFold(strings.TrimSpace(req.Provider), "codex") {
		agentArgs = buildSpriteCodexArgs(req)
	} else {
		agentArgs = buildSpriteClaudeArgs(req, systemPrompt)
	}

	// Start agent on the sprite, reading prompt from the uploaded file.
	// Sprites stdin doesn't reliably forward data, so we use shell redirect.
	quotedArgs := make([]string, len(agentArgs))
	for i, arg := range agentArgs {
		quotedArgs[i] = shellx.Quote(arg)
	}
	shellCmd := fmt.Sprintf("cat /work/prompt.txt | %s", strings.Join(quotedArgs, " "))
	cmd := sprite.CommandContext(ctx, "sh", "-c", shellCmd)
	cmd.Dir = spriteWorkDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start agent on sprite: %w", err)
	}

	if err := writeDispatchMetadata(d.runtimeDir, sessionID, req, nowUTC()); err != nil {
		_ = cmd.Signal("SIGKILL")
		return nil, fmt.Errorf("write spawn metadata: %w", err)
	}

	session := newSpritesSession(spritesSessionConfig{
		id:            sessionID,
		sprite:        sprite,
		spriteName:    d.spriteName,
		cmd:           cmd,
		stdout:        stdout,
		runtimeDir:    d.runtimeDir,
		stampedPath:   stampedPath,
		canonicalPath: canonicalPath,
		eventWriter:   eventWriter,
		prompt:        req.Prompt,
		warnings:      skillBundle.Warnings,
		remoteURL:     remoteURL,
	})
	session.start(ctx)
	return session, nil
}

// gitCurrentBranch returns the current branch name for a git repo.
func gitCurrentBranch(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// pushWorktreeBranch pushes the worktree's current branch to origin.
func pushWorktreeBranch(ctx context.Context, worktreePath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "push", "origin", "HEAD:refs/heads/"+branch, "--force")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// authenticatedRemoteURL rewrites a GitHub remote URL to include token auth.
// Handles both SSH (git@github.com:org/repo.git) and HTTPS formats.
// Fine-grained PATs (github_pat_*) use "oauth2" as the username;
// classic PATs and GitHub App installation tokens use "x-access-token".
func authenticatedRemoteURL(remoteURL, token string) string {
	user := "x-access-token"
	if strings.HasPrefix(token, "github_pat_") {
		user = "oauth2"
	}
	// git@github.com:org/repo.git → https://user:token@github.com/org/repo.git
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		path := strings.TrimPrefix(remoteURL, "git@github.com:")
		return fmt.Sprintf("https://%s:%s@github.com/%s", user, token, path)
	}
	// https://github.com/org/repo.git → https://user:token@github.com/org/repo.git
	if strings.HasPrefix(remoteURL, "https://github.com/") {
		path := strings.TrimPrefix(remoteURL, "https://github.com/")
		return fmt.Sprintf("https://%s:%s@github.com/%s", user, token, path)
	}
	return remoteURL
}

// cloneOnSprite clones a repo from a remote URL onto the sprite.
func cloneOnSprite(ctx context.Context, sprite spriteHandle, remoteURL, branch string) error {
	// Clean stale repo from previous runs.
	rmCmd := sprite.CommandContext(ctx, "rm", "-rf", spriteWorkDir)
	_ = rmCmd.Run()

	cloneCmd := sprite.CommandContext(ctx, "git", "clone", remoteURL, "--branch", branch, "--single-branch", "--quiet", spriteWorkDir)
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// uploadFileToSprite writes a local file to a path on the sprite using base64
// encoding to avoid stdin forwarding issues.
func uploadFileToSprite(ctx context.Context, sprite spriteHandle, localPath, remotePath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read local file: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	script := fmt.Sprintf("echo '%s' | base64 -d > %s", encoded, shellx.Quote(remotePath))
	cmd := sprite.CommandContext(ctx, "sh", "-c", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("write remote file: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func buildSpriteCodexArgs(req DispatchRequest) []string {
	return append([]string{"codex"}, codexBaseArgs(req)...)
}

func buildSpriteClaudeArgs(req DispatchRequest, systemPrompt string) []string {
	return append([]string{"claude"}, claudeBaseArgs(req, systemPrompt)...)
}

// gitRemoteURL reads the origin remote URL from a git repo.
func gitRemoteURL(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// pushChangesFromSprite commits and pushes any changes on the sprite to a noodle branch.
func pushChangesFromSprite(ctx context.Context, sprite spriteHandle, sessionID string) (SyncResult, error) {
	branch := "noodle/" + sessionID

	// Commit all changes (if any).
	commitScript := fmt.Sprintf(
		"cd %s && git add -A && (git diff --cached --quiet || git commit -m %s)",
		shellx.Quote(spriteWorkDir),
		shellx.Quote("noodle: "+sessionID),
	)
	commitCmd := sprite.CommandContext(ctx, "sh", "-c", commitScript)
	_ = commitCmd.Run() // OK if nothing to commit.

	// Check if HEAD has moved beyond origin (any new commits from the agent).
	logCmd := sprite.CommandContext(ctx, "sh", "-c",
		fmt.Sprintf("cd %s && git log --oneline @{upstream}..HEAD 2>/dev/null | head -1", shellx.Quote(spriteWorkDir)))
	out, _ := logCmd.Output()
	if strings.TrimSpace(string(out)) == "" {
		return SyncResult{Type: SyncResultTypeNone}, nil
	}

	// Push to remote branch.
	pushScript := fmt.Sprintf(
		"cd %s && git push origin HEAD:refs/heads/%s",
		shellx.Quote(spriteWorkDir),
		branch,
	)
	pushCmd := sprite.CommandContext(ctx, "sh", "-c", pushScript)
	if pushOut, err := pushCmd.CombinedOutput(); err != nil {
		return SyncResult{}, fmt.Errorf("push to %s: %s: %w", branch, strings.TrimSpace(string(pushOut)), err)
	}

	return SyncResult{Type: SyncResultTypeBranch, Branch: branch}, nil
}

// writeSyncResult updates spawn.json with the sync result after session completion.
func writeSyncResult(runtimeDir, sessionID string, result SyncResult) error {
	path := filepath.Join(runtimeDir, "sessions", sessionID, "spawn.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read spawn metadata: %w", err)
	}

	// Parse, add sync field, re-write.
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parse spawn metadata: %w", err)
	}
	syncData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode sync result: %w", err)
	}
	payload["sync"] = syncData

	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode spawn metadata: %w", err)
	}
	return os.WriteFile(path, encoded, 0o644)
}
