package main

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestRunWorktreeCommandRequiresSubcommand(t *testing.T) {
	err := runWorktreeCommand(context.Background(), nil, nil, nil)
	if err == nil {
		t.Fatal("expected missing subcommand error")
	}
	if !strings.Contains(err.Error(), "subcommand is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunWorktreeCommandDispatchesCreate(t *testing.T) {
	originalFactory := newWorktreeCommandApp
	t.Cleanup(func() {
		newWorktreeCommandApp = originalFactory
	})

	fake := &fakeWorktreeCommandApp{}
	newWorktreeCommandApp = func() (worktreeCommandApp, error) {
		return fake, nil
	}

	if err := runWorktreeCommand(context.Background(), nil, nil, []string{"create", "feat-a"}); err != nil {
		t.Fatalf("runWorktreeCommand: %v", err)
	}
	if fake.createName != "feat-a" {
		t.Fatalf("create name = %q", fake.createName)
	}
}

func TestRunWorktreeCommandDispatchesCleanupWithForce(t *testing.T) {
	originalFactory := newWorktreeCommandApp
	t.Cleanup(func() {
		newWorktreeCommandApp = originalFactory
	})

	fake := &fakeWorktreeCommandApp{}
	newWorktreeCommandApp = func() (worktreeCommandApp, error) {
		return fake, nil
	}

	if err := runWorktreeCommand(context.Background(), nil, nil, []string{"cleanup", "--force", "feat-a"}); err != nil {
		t.Fatalf("runWorktreeCommand: %v", err)
	}
	if fake.cleanupName != "feat-a" {
		t.Fatalf("cleanup name = %q", fake.cleanupName)
	}
	if !fake.cleanupForce {
		t.Fatal("expected cleanup force=true")
	}
}

func TestRunWorktreeCommandDispatchesHook(t *testing.T) {
	originalFactory := newWorktreeCommandApp
	originalHook := runWorktreeHook
	t.Cleanup(func() {
		newWorktreeCommandApp = originalFactory
		runWorktreeHook = originalHook
	})

	factoryCalled := false
	newWorktreeCommandApp = func() (worktreeCommandApp, error) {
		factoryCalled = true
		return &fakeWorktreeCommandApp{}, nil
	}

	hookCalled := false
	runWorktreeHook = func(_ io.Reader, _ io.Writer) error {
		hookCalled = true
		return nil
	}

	if err := runWorktreeCommand(context.Background(), nil, nil, []string{"hook"}); err != nil {
		t.Fatalf("runWorktreeCommand: %v", err)
	}
	if factoryCalled {
		t.Fatal("expected hook to bypass worktree app factory")
	}
	if !hookCalled {
		t.Fatal("expected hook handler call")
	}
}

type fakeWorktreeCommandApp struct {
	createName   string
	mergeName    string
	cleanupName  string
	cleanupForce bool
	listCalled   bool
	pruneCalled  bool
}

func (f *fakeWorktreeCommandApp) Create(name string) error {
	f.createName = name
	return nil
}

func (f *fakeWorktreeCommandApp) Merge(name string) error {
	f.mergeName = name
	return nil
}

func (f *fakeWorktreeCommandApp) Cleanup(name string, force bool) error {
	f.cleanupName = name
	f.cleanupForce = force
	return nil
}

func (f *fakeWorktreeCommandApp) List() error {
	f.listCalled = true
	return nil
}

func (f *fakeWorktreeCommandApp) Prune() error {
	f.pruneCalled = true
	return nil
}
