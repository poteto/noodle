package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/startup"
	"github.com/spf13/cobra"
)

// App holds loaded configuration and validation state, shared across all subcommands.
type App struct {
	Config     config.Config
	Validation config.ValidationResult
	projectDir string // resolved once in PersistentPreRunE
}

// NewRootCmd builds the cobra command tree with a shared *App populated in PersistentPreRunE.
func NewRootCmd() *cobra.Command {
	var app App
	var projectDirFlag string

	root := &cobra.Command{
		Use:           "noodle",
		Short:         "Open-source AI coding framework",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := resolveProjectDir(projectDirFlag)
			if err != nil {
				return err
			}
			app.projectDir = dir
			if err := os.Chdir(dir); err != nil {
				return fmt.Errorf("change to project directory %s: %w", dir, err)
			}
			if rootSubcommandName(cmd) == "start" {
				if err := startup.EnsureProjectStructure(".", os.Stderr); err != nil {
					return err
				}
			}
			loaded, validation, err := config.Load(config.DefaultConfigPath)
			if err != nil {
				return err
			}
			app.Config = loaded
			app.Validation = validation
			return reportConfigDiagnostics(
				cmd.Context(), os.Stderr, os.Stdin,
				rootSubcommandName(cmd), &app, validation,
			)
		},
	}
	root.PersistentFlags().StringVar(&projectDirFlag, "project-dir", "", "project directory (default: current directory, env: NOODLE_PROJECT_DIR)")

	root.AddCommand(
		newStartCmd(&app),
		newSkillsCmd(&app),
		newSchemaCmd(&app),
		newStatusCmd(&app),
		newWorktreeCmd(&app),
		newEventCmd(&app),
		newResetCmd(&app),
	)

	return root
}

// resolveProjectDir returns an absolute project directory from flag > env > cwd.
func resolveProjectDir(flag string) (string, error) {
	dir := flag
	if dir == "" {
		dir = os.Getenv("NOODLE_PROJECT_DIR")
	}
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get current directory: %w", err)
		}
		return cwd, nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve project directory %s: %w", dir, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("project directory %s: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project directory %s: not a directory", abs)
	}
	return abs, nil
}

// rootSubcommandName walks up to the command directly below root and returns
// its name.  For top-level commands this is just cmd.Name(); for nested
// subcommands (e.g. "worktree create") it returns the first-level parent.
func rootSubcommandName(cmd *cobra.Command) string {
	for cmd.HasParent() && cmd.Parent().HasParent() {
		cmd = cmd.Parent()
	}
	return cmd.Name()
}
