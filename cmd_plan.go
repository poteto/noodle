package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/plan"
	"github.com/spf13/cobra"
)

func newPlanCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: cmdmeta.Short("plan"),
	}
	cmd.AddCommand(
		newPlanCreateCmd(),
		newPlanActivateCmd(),
		newPlanDoneCmd(app),
		newPlanPhaseAddCmd(),
		newPlanListCmd(),
	)
	return cmd
}

func newPlanCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <todo-id> <slug>",
		Short: cmdmeta.Short("plan", "create"),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanCreate(args)
		},
	}
}

func newPlanActivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "activate <plan-id>",
		Short: cmdmeta.Short("plan", "activate"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanActivate(args)
		},
	}
}

func newPlanDoneCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "done <plan-id>",
		Short: cmdmeta.Short("plan", "done"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanDone(args, app.Config.Plans.OnDone)
		},
	}
}

func newPlanPhaseAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "phase-add <plan-id> <phase-name>",
		Short: cmdmeta.Short("plan", "phase-add"),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanPhaseAdd(args)
		},
	}
}

func newPlanListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: cmdmeta.Short("plan", "list"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPlanList()
		},
	}
}

func runPlanCreate(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("plan create requires <todo-id> <slug>")
	}

	todoID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("todo-id not a valid integer: %q", args[0])
	}

	slug := strings.TrimSpace(args[1])
	if slug == "" {
		return fmt.Errorf("plan create slug is empty")
	}

	plansDir, err := resolvePlansDir()
	if err != nil {
		return err
	}

	created, err := plan.Create(plansDir, todoID, slug)
	if err != nil {
		return err
	}

	fmt.Println(created)
	return nil
}

func runPlanActivate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("plan activate requires <plan-id>")
	}

	planID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("plan-id not a valid integer: %q", args[0])
	}

	plansDir, err := resolvePlansDir()
	if err != nil {
		return err
	}

	return plan.Activate(plansDir, planID)
}

func runPlanDone(args []string, onDone string) error {
	if len(args) != 1 {
		return fmt.Errorf("plan done requires <plan-id>")
	}

	planID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("plan-id not a valid integer: %q", args[0])
	}

	plansDir, err := resolvePlansDir()
	if err != nil {
		return err
	}

	return plan.Done(plansDir, planID, onDone)
}

func runPlanPhaseAdd(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("plan phase-add requires <plan-id> <phase-name>")
	}

	planID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("plan-id not a valid integer: %q", args[0])
	}

	phaseName := strings.TrimSpace(args[1])
	if phaseName == "" {
		return fmt.Errorf("plan phase-add phase name is empty")
	}

	plansDir, err := resolvePlansDir()
	if err != nil {
		return err
	}

	created, err := plan.PhaseAdd(plansDir, planID, phaseName)
	if err != nil {
		return err
	}

	fmt.Println(created)
	return nil
}

func runPlanList() error {
	plansDir, err := resolvePlansDir()
	if err != nil {
		return err
	}

	plans, err := plan.ReadAll(plansDir)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(plans)
}

// resolvePlansDir returns the absolute path to brain/plans/ relative to the
// current working directory.
func resolvePlansDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("working directory not resolved: %w", err)
	}
	return filepath.Join(wd, "brain", "plans"), nil
}
