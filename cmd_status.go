package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/statusfile"
	"github.com/spf13/cobra"
)

type statusSummary struct {
	ActiveCooks int
	QueueDepth  int
	TotalCost   float64
	LoopState   string
}

func newStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: cmdmeta.Short("status"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStatus(app)
		},
	}
}

func runStatus(app *App) error {
	runtimeDir, err := app.RuntimeDir()
	if err != nil {
		return err
	}
	summary, err := readStatusSummary(runtimeDir)
	if err != nil {
		return err
	}

	if summary.ActiveCooks == 0 {
		fmt.Fprintf(
			os.Stdout,
			"no active cooks | queue=%d | cost=$%.2f | loop=%s\n",
			summary.QueueDepth,
			summary.TotalCost,
			summary.LoopState,
		)
		return nil
	}

	fmt.Fprintf(
		os.Stdout,
		"active cooks=%d | queue=%d | cost=$%.2f | loop=%s\n",
		summary.ActiveCooks,
		summary.QueueDepth,
		summary.TotalCost,
		summary.LoopState,
	)
	return nil
}

func readStatusSummary(runtimeDir string) (statusSummary, error) {
	active, cost, loopState, err := readSessionSummary(runtimeDir)
	if err != nil {
		return statusSummary{}, err
	}
	queueDepth, err := readOrdersDepth(filepath.Join(runtimeDir, "orders.json"))
	if err != nil {
		return statusSummary{}, err
	}
	return statusSummary{
		ActiveCooks: active,
		QueueDepth:  queueDepth,
		TotalCost:   cost,
		LoopState:   loopState,
	}, nil
}

func readSessionSummary(runtimeDir string) (active int, totalCost float64, loopState string, _ error) {
	status, err := statusfile.Read(filepath.Join(runtimeDir, "status.json"))
	if err != nil {
		return 0, 0, "", err
	}
	active = len(status.Active)
	loopState = normalizeLoopState(status.LoopState)
	if loopState == "" {
		loopState = "running"
	}
	return active, totalCost, loopState, nil
}

func normalizeLoopState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "running":
		return "running"
	case "paused":
		return "paused"
	case "draining":
		return "draining"
	default:
		return ""
	}
}

func pickLoopState(current, candidate string) string {
	current = normalizeLoopState(current)
	candidate = normalizeLoopState(candidate)
	if candidate == "" {
		return current
	}
	if loopStateRank(candidate) > loopStateRank(current) {
		return candidate
	}
	return current
}

func loopStateRank(state string) int {
	switch state {
	case "draining":
		return 3
	case "paused":
		return 2
	case "running":
		return 1
	default:
		return 0
	}
}

func readOrdersDepth(path string) (int, error) {
	orders, err := orderx.ReadOrders(path)
	if err != nil {
		return 0, err
	}
	return len(orders.Orders), nil
}
