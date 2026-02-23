package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/poteto/noodle/plan"
)

func runPlanCommand(_ context.Context, _ *App, _ []Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("plan subcommand is required (create, done, phase-add, list)")
	}

	subcommand := strings.TrimSpace(args[0])
	subArgs := args[1:]

	switch subcommand {
	case "create":
		return runPlanCreate(subArgs)
	case "done":
		return runPlanDone(subArgs)
	case "phase-add":
		return runPlanPhaseAdd(subArgs)
	case "list":
		return runPlanList(subArgs)
	default:
		return fmt.Errorf("unknown plan subcommand %q", subcommand)
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

func runPlanDone(args []string) error {
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

	return plan.Done(plansDir, planID)
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

func runPlanList(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("plan list does not accept arguments")
	}

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
