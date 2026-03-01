package stringx

import "testing"

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "whitespace", in: " \t\n ", want: ""},
		{name: "mixed case", in: "  HeLLo WoRLD  ", want: "hello world"},
		{name: "already normalized", in: "already-normalized", want: "already-normalized"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Normalize(tt.in); got != tt.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMiddleTruncateFitsNoOp(t *testing.T) {
	got := MiddleTruncate("hello", 10)
	if got != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
}

func TestMiddleTruncatePlain(t *testing.T) {
	got := MiddleTruncate("abcdefghij", 7)
	// 3 head + … + 3 tail = 7
	if got != "abc…hij" {
		t.Fatalf("got %q, want %q", got, "abc…hij")
	}
}

func TestMiddleTruncatePathPreservesEnds(t *testing.T) {
	// /Users/lauren/code/noodle/.noodle/orders.json -> fits more end segments
	got := MiddleTruncate("/Users/lauren/code/noodle/.noodle/orders.json", 30)
	if got != "/…/noodle/.noodle/orders.json" {
		t.Fatalf("got %q, want %q", got, "/…/noodle/.noodle/orders.json")
	}
}

func TestMiddleTruncatePathTight(t *testing.T) {
	got := MiddleTruncate("/Users/lauren/code/noodle/.noodle/orders.json", 22)
	if got != "/…/.noodle/orders.json" {
		t.Fatalf("got %q, want %q", got, "/…/.noodle/orders.json")
	}
}

func TestMiddleTruncateShortPath(t *testing.T) {
	got := MiddleTruncate("a/b/c/d/e", 9)
	// Fits exactly.
	if got != "a/b/c/d/e" {
		t.Fatalf("got %q, want %q", got, "a/b/c/d/e")
	}
}

func TestMiddleTruncatePathCollapse(t *testing.T) {
	got := MiddleTruncate("brain/plans/25-tui-revamp/phase-08.md", 25)
	// Should preserve first + last, maybe fit more.
	if len(got) > 25 {
		t.Fatalf("result too long: %d > 25: %q", len(got), got)
	}
	if got == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestMiddleTruncateWidthOne(t *testing.T) {
	got := MiddleTruncate("abcdef", 1)
	if got != "…" {
		t.Fatalf("got %q, want %q", got, "…")
	}
}

func TestMiddleTruncateWidthZero(t *testing.T) {
	got := MiddleTruncate("abcdef", 0)
	if got != "" {
		t.Fatalf("got %q, want %q", got, "")
	}
}
