package main

import (
	"os"

	"github.com/poteto/noodle/config"
	"github.com/spf13/cobra"
)

// App holds loaded configuration and validation state, shared across all subcommands.
type App struct {
	Config     config.Config
	Validation config.ValidationResult
}

// NewRootCmd builds the cobra command tree with a shared *App populated in PersistentPreRunE.
func NewRootCmd() *cobra.Command {
	var app App

	root := &cobra.Command{
		Use:           "noodle",
		Short:         "Open-source AI coding framework",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
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

	root.AddCommand(
		newStartCmd(&app),
		newSkillsCmd(&app),
		newSchemaCmd(&app),
		newStatusCmd(&app),
		newDebugCmd(&app),
		newWorktreeCmd(&app),
		newStampCmd(&app),
		newDispatchCmd(&app),
		newMiseCmd(&app),
		newPlanCmd(&app),
	)

	return root
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
