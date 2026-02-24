package dispatcher

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

// ProviderConfig holds optional CLI binary directory and extra flags by provider.
type ProviderConfig struct {
	Path string
	Args []string
}

// ProviderConfigs holds per-provider configuration.
type ProviderConfigs struct {
	Claude ProviderConfig
	Codex  ProviderConfig
}

// TmuxDispatcherConfig configures a tmux dispatcher.
type TmuxDispatcherConfig struct {
	ProjectDir      string
	RuntimeDir      string
	NoodleBin       string
	SkillResolver   skill.Resolver
	ProviderConfigs ProviderConfigs
	RuntimeDefault  string // command template from config, empty = built-in
	RuntimeKind     string // runtime kind this dispatcher instance services
}

// TmuxDispatcher dispatches provider sessions in detached tmux sessions.
type TmuxDispatcher struct {
	projectDir      string
	runtimeDir      string
	noodleBin       string
	skillResolver   skill.Resolver
	providerConfigs ProviderConfigs
	runtimeDefault  string
	runtimeKind     string
	run             commandRunner
}

// NewTmuxDispatcher constructs a dispatcher from config.
func NewTmuxDispatcher(config TmuxDispatcherConfig) *TmuxDispatcher {
	return &TmuxDispatcher{
		projectDir:      strings.TrimSpace(config.ProjectDir),
		runtimeDir:      strings.TrimSpace(config.RuntimeDir),
		noodleBin:       strings.TrimSpace(config.NoodleBin),
		skillResolver:   config.SkillResolver,
		providerConfigs: config.ProviderConfigs,
		runtimeDefault:  strings.TrimSpace(config.RuntimeDefault),
		runtimeKind:     normalizeRuntimeKind(config.RuntimeKind),
		run:             defaultRunner,
	}
}

// Dispatch validates a request and starts a detached tmux-backed session.
func (s *TmuxDispatcher) Dispatch(ctx context.Context, req DispatchRequest) (Session, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	reqRuntime := strings.ToLower(strings.TrimSpace(req.Runtime))
	if reqRuntime == "" {
		reqRuntime = s.runtimeKind
	}
	if reqRuntime != s.runtimeKind {
		return nil, fmt.Errorf("runtime %q not configured", reqRuntime)
	}
	req.Runtime = reqRuntime

	if req.AllowPrimaryCheckout {
		req.WorktreePath = strings.TrimSpace(req.WorktreePath)
		if req.WorktreePath == "" {
			req.WorktreePath = s.projectDir
		}
		if strings.TrimSpace(req.WorktreePath) == "" {
			return nil, fmt.Errorf("project directory not set")
		}
	} else {
		validWorktree, err := wt.ValidateLinkedCheckout(req.WorktreePath)
		if err != nil {
			return nil, fmt.Errorf("worktree enforcement: %w", err)
		}
		req.WorktreePath = validWorktree
	}

	if s.runtimeDir == "" {
		return nil, fmt.Errorf("runtime directory not set")
	}
	if s.noodleBin == "" {
		return nil, fmt.Errorf("noodle binary path not set")
	}

	sessionID, err := generateSessionID(req.Name)
	if err != nil {
		return nil, fmt.Errorf("generate session ID: %w", err)
	}
	sessionDir, promptPath, stampedPath, canonicalPath, stderrPath := sessionPaths(s.runtimeDir, sessionID)

	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}
	eventWriter, err := event.NewEventWriter(s.runtimeDir, sessionID)
	if err != nil {
		return nil, fmt.Errorf("create event writer: %w", err)
	}

	var skillBundle loadedSkill
	if req.TaskKey == "execute" && req.DomainSkill != "" {
		skillBundle, err = loadExecuteBundle(s.skillResolver, req.Provider, req.Skill, req.DomainSkill)
	} else {
		skillBundle, err = loadSkillBundle(s.skillResolver, req.Provider, req.Skill)
	}
	if err != nil {
		return nil, err
	}

	fullSystemPrompt := buildSessionPreamble() + "\n\n" + skillBundle.SystemPrompt
	systemPrompt, finalPrompt := composePrompts(req.Provider, req.Prompt, fullSystemPrompt)
	runtimeCmd := strings.TrimSpace(s.runtimeDefault)
	if runtimeCmd != "" &&
		strings.EqualFold(req.Provider, "claude") &&
		strings.TrimSpace(systemPrompt) != "" {
		finalPrompt = finalPrompt + "\n\n---\n\n" + systemPrompt
		systemPrompt = ""
	}
	if err := os.WriteFile(promptPath, []byte(finalPrompt), 0o644); err != nil {
		return nil, fmt.Errorf("write prompt file: %w", err)
	}

	var pipeline string
	if runtimeCmd != "" {
		vars := map[string]string{
			"session": sessionID,
			"repo":    req.WorktreePath,
			"prompt":  promptPath,
			"skill":   req.Skill,
			"brief":   filepath.Join(s.runtimeDir, "mise.json"),
		}
		resolved := resolveTemplateVars(runtimeCmd, vars)
		pipeline = buildPipelineCommand(resolved, s.noodleBin, stampedPath, canonicalPath)
	} else {
		agentBinary := s.resolveAgentBinary(req.Provider)
		extraArgs := s.resolveExtraArgs(req.Provider)
		providerCommand := buildProviderCommand(req, promptPath, agentBinary, systemPrompt, stderrPath, extraArgs)
		pipeline = buildPipelineCommand(providerCommand, s.noodleBin, stampedPath, canonicalPath)
	}

	tmuxName := tmuxSessionName(sessionID, req.Name)
	output, err := s.run(
		ctx,
		req.WorktreePath,
		buildDispatchEnv(req),
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
	if err := writeDispatchMetadata(s.runtimeDir, sessionID, req, nowUTC()); err != nil {
		_, _ = s.run(ctx, req.WorktreePath, buildDispatchEnv(req), "tmux", "kill-session", "-t", tmuxName)
		return nil, fmt.Errorf("write spawn metadata: %w", err)
	}

	session := newTmuxSession(
		sessionID,
		tmuxName,
		req.WorktreePath,
		buildDispatchEnv(req),
		canonicalPath,
		finalPrompt,
		eventWriter,
		skillBundle.Warnings,
		s.run,
	)
	session.start(ctx)
	return session, nil
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

func (s *TmuxDispatcher) resolveAgentBinary(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "codex":
		if path := strings.TrimSpace(s.providerConfigs.Codex.Path); path != "" {
			candidate := filepath.Join(path, "codex")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		return "codex"
	default:
		if path := strings.TrimSpace(s.providerConfigs.Claude.Path); path != "" {
			candidate := filepath.Join(path, "claude")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		return "claude"
	}
}

func (s *TmuxDispatcher) resolveExtraArgs(provider string) []string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "codex":
		return s.providerConfigs.Codex.Args
	default:
		return s.providerConfigs.Claude.Args
	}
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

func normalizeRuntimeKind(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "tmux"
	}
	return value
}

func composePrompts(provider, requestPrompt, skillSystemPrompt string) (systemPrompt, finalPrompt string) {
	finalPrompt = requestPrompt
	if strings.EqualFold(provider, "claude") {
		systemPrompt = skillSystemPrompt
		return systemPrompt, finalPrompt
	}
	if strings.EqualFold(provider, "codex") && strings.TrimSpace(skillSystemPrompt) != "" {
		finalPrompt = requestPrompt + "\n\n---\n\n" + skillSystemPrompt
	}
	return systemPrompt, finalPrompt
}
