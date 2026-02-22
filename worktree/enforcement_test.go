package worktree

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateLinkedCheckout_RejectsPrimaryCheckout(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	repo := setupTestRepo(t)

	_, err := ValidateLinkedCheckout(repo)
	if err == nil {
		t.Fatal("expected primary checkout rejection error")
	}
	if !strings.Contains(err.Error(), "primary checkout") {
		t.Fatalf("expected primary checkout error, got: %v", err)
	}
}

func TestValidateLinkedCheckout_AcceptsLinkedWorktree(t *testing.T) {
	t.Parallel()
	skipWorktreeIntegrationShort(t)

	repo := setupTestRepo(t)
	app := &App{Root: repo}
	if err := app.Create("enforce-test"); err != nil {
		t.Fatalf("Create worktree: %v", err)
	}
	t.Cleanup(func() { _ = app.Cleanup("enforce-test", true) })

	wtPath := WorktreePath(repo, "enforce-test")
	got, err := ValidateLinkedCheckout(wtPath)
	if err != nil {
		t.Fatalf("ValidateLinkedCheckout error: %v", err)
	}
	abs, _ := filepath.Abs(wtPath)
	if got != abs {
		t.Fatalf("resolved path = %q, want %q", got, abs)
	}
}
