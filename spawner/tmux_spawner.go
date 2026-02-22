package spawner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/skill"
	wt "github.com/poteto/noodle/worktree"
)

type commandRunner func(
	ctx context.Context,
	dir string,
	env []string,
	name string,
	args ...string,
) ([]byte, error)

// AgentDirs holds optional CLI binary directories by provider.
type AgentDirs struct {
	ClaudeDir string
	CodexDir  string
}

// TmuxSpawnerConfig configures a tmux spawner.
type TmuxSpawnerConfig struct {
	ProjectDir    string
	RuntimeDir    string
	NoodleBin     string
	SkillResolver skill.Resolver
	AgentDirs     AgentDirs
}

// TmuxSpawner spawns provider sessions in detached tmux sessions.
type TmuxSpawner struct {
	projectDir    string
	runtimeDir    string
	noodleBin     string
	skillResolver skill.Resolver
	agentDirs     AgentDirs
	run           commandRunner
}

// NewTmuxSpawner constructs a spawner from config.
func NewTmuxSpawner(config TmuxSpawnerConfig) *TmuxSpawner {
	return &TmuxSpawner{
		projectDir:    strings.TrimSpace(config.ProjectDir),
		runtimeDir:    strings.TrimSpace(config.RuntimeDir),
		noodleBin:     strings.TrimSpace(config.NoodleBin),
		skillResolver: config.SkillResolver,
		agentDirs:     config.AgentDirs,
		run:           defaultRunner,
	}
}

// Spawn validates a request and starts a detached tmux-backed session.
func (s *TmuxSpawner) Spawn(ctx context.Context, req SpawnRequest) (Session, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	validWorktree, err := wt.ValidateLinkedCheckout(req.WorktreePath)
	if err != nil {
		return nil, fmt.Errorf("worktree enforcement: %w", err)
	}
	req.WorktreePath = validWorktree

	if s.runtimeDir == "" {
		return nil, fmt.Errorf("runtime directory is required")
	}
	if s.noodleBin == "" {
		return nil, fmt.Errorf("noodle binary path is required")
	}

	sessionID, err := generateSessionID(req.Name)
	if err != nil {
		return nil, fmt.Errorf("generate session ID: %w", err)
	}
	sessionDir, promptPath, stampedPath, canonicalPath := sessionPaths(s.runtimeDir, sessionID)

	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}
	eventWriter, err := event.NewEventWriter(s.runtimeDir, sessionID)
	if err != nil {
		return nil, fmt.Errorf("create event writer: %w", err)
	}

	skillBundle, err := loadSkillBundle(s.skillResolver, req.Provider, req.Skill)
	if err != nil {
		return nil, err
	}

	systemPrompt := ""
	finalPrompt := req.Prompt
	if strings.EqualFold(req.Provider, "claude") {
		systemPrompt = skillBundle.SystemPrompt
	} else if strings.EqualFold(req.Provider, "codex") && strings.TrimSpace(skillBundle.SystemPrompt) != "" {
		finalPrompt = skillBundle.SystemPrompt + "\n\n---\n\n" + req.Prompt
	}
	if err := os.WriteFile(promptPath, []byte(finalPrompt), 0o644); err != nil {
		return nil, fmt.Errorf("write prompt file: %w", err)
	}

	agentBinary := s.resolveAgentBinary(req.Provider)
	providerCommand := buildProviderCommand(req, promptPath, agentBinary, systemPrompt)
	pipeline := buildPipelineCommand(providerCommand, s.noodleBin, stampedPath, canonicalPath)

	tmuxName := tmuxSessionName(sessionID, req.Name)
	output, err := s.run(
		ctx,
		req.WorktreePath,
		buildSpawnEnv(req),
		"tmux",
		"new-session",
		"-d",
		"-s",
		tmuxName,
		pipeline,
	)
	if err != nil {
		return nil, fmt.Errorf("tmux new-session: %s: %w", strings.TrimSpace(string(output)), err)
	}
	if err := writeSpawnMetadata(s.runtimeDir, sessionID, req, nowUTC()); err != nil {
		_, _ = s.run(ctx, req.WorktreePath, buildSpawnEnv(req), "tmux", "kill-session", "-t", tmuxName)
		return nil, fmt.Errorf("write spawn metadata: %w", err)
	}

	session := newTmuxSession(
		sessionID,
		tmuxName,
		req.WorktreePath,
		buildSpawnEnv(req),
		canonicalPath,
		eventWriter,
		skillBundle.Warnings,
		s.run,
	)
	session.start(ctx)
	return session, nil
}

func (s *TmuxSpawner) resolveAgentBinary(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "codex":
		if path := strings.TrimSpace(s.agentDirs.CodexDir); path != "" {
			candidate := filepath.Join(path, "codex")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		return "codex"
	default:
		if path := strings.TrimSpace(s.agentDirs.ClaudeDir); path != "" {
			candidate := filepath.Join(path, "claude")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		return "claude"
	}
}

func buildSpawnEnv(req SpawnRequest) []string {
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

func defaultRunner(
	ctx context.Context,
	dir string,
	env []string,
	name string,
	args ...string,
) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = env
	}
	return cmd.CombinedOutput()
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
