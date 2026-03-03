package main

import (
	"fmt"
	"os"

	"github.com/poteto/noodle/cmdmeta"
	"github.com/spf13/cobra"
)

func newVersionCmd(_ *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: cmdmeta.Short("version"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runVersion()
		},
	}
}

func runVersion() error {
	fmt.Fprintln(os.Stdout, currentVersion())
	return nil
}
