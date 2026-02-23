package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/skill"
)

type dispatchCommandDispatcher interface {
	Dispatch(ctx context.Context, req dispatcher.DispatchRequest) (dispatcher.Session, error)
}

var newDispatchCommandDispatcher = func(config dispatcher.TmuxDispatcherConfig) dispatchCommandDispatcher {
	return dispatcher.NewTmuxDispatcher(config)
}

type envFlag map[string]string

func (e *envFlag) String() string {
	if e == nil || len(*e) == 0 {
		return ""
	}
	parts := make([]string, 0, len(*e))
	for key, value := range *e {
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ",")
}

func (e *envFlag) Set(value string) error {
	key, rawValue, ok := strings.Cut(value, "=")
	if !ok {
		return fmt.Errorf("env must be KEY=VALUE")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("env key cannot be empty")
	}
	if *e == nil {
		*e = map[string]string{}
	}
	(*e)[key] = rawValue
	return nil
}

func runDispatchCommand(ctx context.Context, app *App, _ []Command, args []string) error {
	if app != nil && !app.Validation.CanSpawn() {
		return fmt.Errorf("fatal config diagnostics prevent dispatch")
	}

	flags := flag.NewFlagSet("dispatch", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	name := flags.String("name", "cook", "Session name")
	prompt := flags.String("prompt", "", "Prompt text for the dispatched session")
	provider := flags.String("provider", "", "Provider (claude or codex)")
	model := flags.String("model", "", "Model name")
	skillName := flags.String("skill", "", "Optional skill name to inject")
	reasoningLevel := flags.String("reasoning-level", "", "Optional reasoning level")
	worktreePath := flags.String("worktree", "", "Linked worktree path")
	maxTurns := flags.Int("max-turns", 0, "Optional max turns")
	budgetCap := flags.Float64("budget-cap", 0, "Optional budget cap")

	var envVars envFlag
	flags.Var(&envVars, "env", "Extra env var (repeatable, KEY=VALUE)")

	if err := flags.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if strings.TrimSpace(*worktreePath) == "" {
		return fmt.Errorf("worktree is required")
	}

	defaultProvider := ""
	defaultModel := ""
	if app != nil {
		defaultProvider = app.Config.Routing.Defaults.Provider
		defaultModel = app.Config.Routing.Defaults.Model
	}
	if strings.TrimSpace(*provider) == "" {
		*provider = defaultProvider
	}
	if strings.TrimSpace(*model) == "" {
		*model = defaultModel
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}
	runtimeDir := filepath.Join(cwd, ".noodle")

	noodleBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	var resolver skill.Resolver
	var agentDirs dispatcher.AgentDirs
	if app != nil {
		resolver = skill.Resolver{SearchPaths: app.Config.Skills.Paths}
		agentDirs = dispatcher.AgentDirs{
			ClaudeDir: app.Config.Agents.ClaudeDir,
			CodexDir:  app.Config.Agents.CodexDir,
		}
	}

	s := newDispatchCommandDispatcher(dispatcher.TmuxDispatcherConfig{
		ProjectDir:    cwd,
		RuntimeDir:    runtimeDir,
		NoodleBin:     noodleBin,
		SkillResolver: resolver,
		AgentDirs:     agentDirs,
	})
	session, err := s.Dispatch(ctx, dispatcher.DispatchRequest{
		Name:           *name,
		Prompt:         *prompt,
		Provider:       *provider,
		Model:          *model,
		Skill:          *skillName,
		ReasoningLevel: *reasoningLevel,
		WorktreePath:   *worktreePath,
		MaxTurns:       *maxTurns,
		EnvVars:        envVars,
		BudgetCap:      *budgetCap,
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, session.ID())
	return nil
}
