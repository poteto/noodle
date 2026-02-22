package main

import "testing"

func TestCommandCatalogRegistersStatusAndWorktree(t *testing.T) {
	catalog := CommandCatalog()

	start, ok := FindCommand(catalog, "start")
	if !ok {
		t.Fatal("start command not registered")
	}
	if start.Category != "core" {
		t.Fatalf("start category = %q", start.Category)
	}

	status, ok := FindCommand(catalog, "status")
	if !ok {
		t.Fatal("status command not registered")
	}
	if status.Category != "core" {
		t.Fatalf("status category = %q", status.Category)
	}

	worktree, ok := FindCommand(catalog, "worktree")
	if !ok {
		t.Fatal("worktree command not registered")
	}
	if worktree.Category != "core" {
		t.Fatalf("worktree category = %q", worktree.Category)
	}
}
