package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/dispatcher"
	"github.com/spf13/cobra"
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

func (e *envFlag) Type() string { return "KEY=VALUE" }

type dispatchArgs struct {
	name           string
	prompt         string
	provider       string
	model          string
	skill          string
	reasoningLevel string
	worktree       string
	maxTurns       int
	budgetCap      float64
	envVars        map[string]string
}

func newDispatchCmd(app *App) *cobra.Command {
	var (
		args    dispatchArgs
		envVars envFlag
	)
	cmd := &cobra.Command{
		Use:   "dispatch",
		Short: cmdmeta.Short("dispatch"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			args.envVars = envVars
			return runDispatch(cmd.Context(), app, args)
		},
	}
	cmd.Flags().StringVar(&args.name, "name", "cook", "Session name")
	cmd.Flags().StringVar(&args.prompt, "prompt", "", "Prompt text for the dispatched session")
	cmd.Flags().StringVar(&args.provider, "provider", "", "Provider (claude or codex)")
	cmd.Flags().StringVar(&args.model, "model", "", "Model name")
	cmd.Flags().StringVar(&args.skill, "skill", "", "Optional skill name to inject")
	cmd.Flags().StringVar(&args.reasoningLevel, "reasoning-level", "", "Optional reasoning level")
	cmd.Flags().StringVar(&args.worktree, "worktree", "", "Linked worktree path")
	cmd.Flags().IntVar(&args.maxTurns, "max-turns", 0, "Optional max turns")
	cmd.Flags().Float64Var(&args.budgetCap, "budget-cap", 0, "Optional budget cap")
	cmd.Flags().Var(&envVars, "env", "Extra env var (repeatable, KEY=VALUE)")
	return cmd
}

func runDispatch(ctx context.Context, app *App, args dispatchArgs) error {
	if app != nil && !app.Validation.CanSpawn() {
		return fmt.Errorf("fatal config diagnostics prevent dispatch")
	}

	if strings.TrimSpace(args.prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if strings.TrimSpace(args.worktree) == "" {
		return fmt.Errorf("worktree is required")
	}

	defaultProvider := ""
	defaultModel := ""
	if app != nil {
		defaultProvider = app.Config.Routing.Defaults.Provider
		defaultModel = app.Config.Routing.Defaults.Model
	}
	if strings.TrimSpace(args.provider) == "" {
		args.provider = defaultProvider
	}
	if strings.TrimSpace(args.model) == "" {
		args.model = defaultModel
	}

	cwd, err := app.ProjectDir()
	if err != nil {
		return err
	}
	runtimeDir, err := app.RuntimeDir()
	if err != nil {
		return err
	}

	noodleBin, err := app.NoodleBinaryPath()
	if err != nil {
		return err
	}

	s := newDispatchCommandDispatcher(dispatcher.TmuxDispatcherConfig{
		ProjectDir:      cwd,
		RuntimeDir:      runtimeDir,
		NoodleBin:       noodleBin,
		SkillResolver:   app.SkillResolver(),
		ProviderConfigs: app.ProviderConfigs(),
	})
	session, err := s.Dispatch(ctx, dispatcher.DispatchRequest{
		Name:           args.name,
		Prompt:         args.prompt,
		Provider:       args.provider,
		Model:          args.model,
		Skill:          args.skill,
		ReasoningLevel: args.reasoningLevel,
		WorktreePath:   args.worktree,
		MaxTurns:       args.maxTurns,
		EnvVars:        args.envVars,
		BudgetCap:      args.budgetCap,
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, session.ID())
	return nil
}
