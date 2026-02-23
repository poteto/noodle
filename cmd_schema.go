package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/internal/schemadoc"
	"github.com/spf13/cobra"
)

func newSchemaCmd(_ *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema [target]",
		Short: "Print generated schema docs for Noodle runtime contracts",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runSchemaList()
			}
			return runSchema(args[0])
		},
	}
	cmd.AddCommand(newSchemaListCmd())
	return cmd
}

func newSchemaListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available schema targets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSchemaList()
		},
	}
}

func runSchemaList() error {
	targets := schemadoc.ListTargets()
	for _, target := range targets {
		fmt.Fprintf(os.Stdout, "%s\t%s\n", target.Name, target.Description)
	}
	return nil
}

func runSchema(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("schema target is empty")
	}
	out, err := schemadoc.RenderMarkdown(target)
	if err != nil {
		return fmt.Errorf("%w. Run `noodle schema list` to see supported targets", err)
	}
	fmt.Fprint(os.Stdout, out)
	return nil
}
