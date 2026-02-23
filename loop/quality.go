package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/debate"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/mise"
)

func (l *Loop) runQuality(ctx context.Context, cook *activeCook) (bool, string) {
	reviewReq := dispatcher.DispatchRequest{
		Name:         cook.worktreeName + "-quality",
		Prompt:       "Review completed cook work for item " + cook.queueItem.ID,
		Provider:     l.config.Routing.Defaults.Provider,
		Model:        l.config.Routing.Defaults.Model,
		Skill:        taskSkill(l.registry, "quality", "quality"),
		WorktreePath: cook.worktreePath,
	}
	session, err := l.deps.Dispatcher.Dispatch(ctx, reviewReq)
	if err != nil {
		return false, "unable to spawn quality review: " + err.Error()
	}
	select {
	case <-ctx.Done():
		_ = session.Kill()
		return false, ctx.Err().Error()
	case <-session.Done():
	}

	// The quality skill writes .noodle/quality/<session-id>.json relative to
	// its CWD (the cook's worktree). That's the single source of truth.
	// No verdict file → reject. No fallback, no fail-open.
	verdictName := session.ID() + ".json"
	verdict, err := readQualityVerdictFile(filepath.Join(cook.worktreePath, ".noodle", "quality", verdictName))
	if err != nil {
		feedback := "quality review produced no verdict"
		_ = l.writeDebateVerdict(cook, false, feedback)
		return false, feedback
	}

	// Copy verdict to project-level .noodle/quality/ so mise can include
	// historical quality signals in the brief for prioritization.
	if err := copyVerdictToRuntime(
		filepath.Join(cook.worktreePath, ".noodle", "quality", verdictName),
		filepath.Join(l.runtimeDir, "quality", verdictName),
	); err != nil {
		fmt.Fprintf(os.Stderr, "warning: verdict not mirrored to runtime: %v\n", err)
	}

	_ = l.writeDebateVerdict(cook, verdict.Accept, verdict.Feedback)
	return verdict.Accept, verdict.Feedback
}

// readQualityVerdictFile reads the structured JSON verdict the quality skill
// writes to .noodle/quality/<session-id>.json. Uses mise.QualityVerdict as
// the single canonical type — no separate loop-local struct needed.
func readQualityVerdictFile(path string) (mise.QualityVerdict, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return mise.QualityVerdict{}, err
	}
	var v mise.QualityVerdict
	if err := json.Unmarshal(data, &v); err != nil {
		return mise.QualityVerdict{}, err
	}
	return v, nil
}

// copyVerdictToRuntime copies a verdict file from a worktree's .noodle/quality/
// to the project-level .noodle/quality/ so mise can read historical verdicts.
func copyVerdictToRuntime(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func (l *Loop) writeDebateVerdict(cook *activeCook, accept bool, feedback string) error {
	store, err := debate.NewStore(filepath.Join(l.projectDir, "brain", "debates"))
	if err != nil {
		return err
	}
	d, err := store.Create("cook-"+cook.queueItem.ID, 6)
	if err != nil {
		return err
	}
	if _, err := store.AddRound(d, "reviewer", "Quality review for item "+cook.queueItem.ID); err != nil {
		return err
	}
	return store.WriteVerdict(d, debate.Verdict{Consensus: accept, Summary: strings.TrimSpace(feedback)})
}
