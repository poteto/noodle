package server

import (
	"slices"
	"testing"
)

func TestMergeWarningsEmpty(t *testing.T) {
	got := mergeWarnings(nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}

	got = mergeWarnings([]string{}, []string{})
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestMergeWarningsDedup(t *testing.T) {
	got := mergeWarnings(
		[]string{"alpha", "beta"},
		[]string{"beta", "gamma"},
	)
	want := []string{"alpha", "beta", "gamma"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeWarningsDisjointSorted(t *testing.T) {
	got := mergeWarnings(
		[]string{"zoo"},
		[]string{"apple"},
	)
	want := []string{"apple", "zoo"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeWarningsDoesNotMutateInputs(t *testing.T) {
	static := []string{"b", "a"}
	dynamic := []string{"c", "b"}

	staticCopy := slices.Clone(static)
	dynamicCopy := slices.Clone(dynamic)

	mergeWarnings(static, dynamic)

	if !slices.Equal(static, staticCopy) {
		t.Fatalf("static mutated: got %v, want %v", static, staticCopy)
	}
	if !slices.Equal(dynamic, dynamicCopy) {
		t.Fatalf("dynamic mutated: got %v, want %v", dynamic, dynamicCopy)
	}
}
