package main

import "testing"

func TestCommandCatalogRegistersStatusTuiWorktreeAndDebug(t *testing.T) {
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

	debug, ok := FindCommand(catalog, "debug")
	if !ok {
		t.Fatal("debug command not registered")
	}
	if debug.Category != "core" {
		t.Fatalf("debug category = %q", debug.Category)
	}

	tui, ok := FindCommand(catalog, "tui")
	if !ok {
		t.Fatal("tui command not registered")
	}
	if tui.Category != "core" {
		t.Fatalf("tui category = %q", tui.Category)
	}

	worktree, ok := FindCommand(catalog, "worktree")
	if !ok {
		t.Fatal("worktree command not registered")
	}
	if worktree.Category != "core" {
		t.Fatalf("worktree category = %q", worktree.Category)
	}
}
