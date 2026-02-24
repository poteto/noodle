package components

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

func TestCardRenderAtMultipleWidths(t *testing.T) {
	c := &Card{
		Title: "Test Card",
		Body:  "Some content here",
	}
	for _, w := range []int{30, 50, 80, 120} {
		out := c.Render(w)
		if out == "" {
			t.Fatalf("empty card at width %d", w)
		}
		c.ClearCache()
	}
}

func TestCardCache(t *testing.T) {
	c := &Card{Title: "Cached", Body: "body"}
	first := c.Render(40)
	second := c.Render(40)
	if first != second {
		t.Fatal("cached render mismatch")
	}
	// Different width should produce different output
	third := c.Render(80)
	if first == third {
		t.Fatal("expected different render at different width")
	}
}

func TestPillRender(t *testing.T) {
	p := Pill{Label: "Merge", Icon: "✓", Focused: true, Color: DefaultTheme.Success}
	out := p.Render(20)
	if out == "" {
		t.Fatal("empty pill render")
	}
	if !strings.Contains(lipgloss.NewStyle().Render(out), "") {
		// Just verify it renders without panic
	}
}

func TestPillBlurred(t *testing.T) {
	p := Pill{Label: "Reject", Focused: false}
	out := p.Render(20)
	if out == "" {
		t.Fatal("empty blurred pill")
	}
}

func TestSectionHeader(t *testing.T) {
	out := Section("Autonomy", 40)
	if !strings.Contains(out, "Autonomy") {
		t.Fatalf("section missing title: %q", out)
	}
	if !strings.Contains(out, "╌") {
		t.Fatalf("section missing underline: %q", out)
	}
}

func TestSectionNarrow(t *testing.T) {
	out := Section("Very Long Section Title", 10)
	if out == "" {
		t.Fatal("empty narrow section")
	}
}

func TestBadgeTaskType(t *testing.T) {
	for _, tt := range []string{"Execute", "Plan", "Quality", "Reflect", "Prioritize"} {
		out := TaskTypeBadge(tt)
		if out == "" {
			t.Fatalf("empty badge for %s", tt)
		}
	}
}

func TestVerdictBadge(t *testing.T) {
	for _, v := range []string{"APPROVE", "FLAG", "REJECT", "unknown"} {
		out := VerdictBadge(v)
		if out == "" {
			t.Fatalf("empty verdict badge for %s", v)
		}
	}
}

func TestProgressBar(t *testing.T) {
	out := ProgressBar(2.5, 50, 40)
	if !strings.Contains(out, "█") {
		t.Fatalf("progress bar missing fill: %q", out)
	}
	if !strings.Contains(out, "░") {
		t.Fatalf("progress bar missing empty: %q", out)
	}
	if !strings.Contains(out, "5%") {
		t.Fatalf("progress bar missing percentage: %q", out)
	}
}

func TestProgressBarHighUsage(t *testing.T) {
	out := ProgressBar(48, 50, 40)
	if !strings.Contains(out, "96%") {
		t.Fatalf("expected 96%% in high usage bar: %q", out)
	}
}

func TestHealthDot(t *testing.T) {
	for _, h := range []string{"green", "yellow", "red"} {
		out := HealthDot(h)
		if !strings.Contains(out, "●") {
			t.Fatalf("health dot missing for %s: %q", h, out)
		}
	}
}

func TestStatusIcon(t *testing.T) {
	cases := map[string]string{
		"failed":    "x",
		"completed": "✓",
		"running":   "~",
	}
	for status, want := range cases {
		out := StatusIcon(status)
		if !strings.Contains(out, want) {
			t.Fatalf("StatusIcon(%q) = %q, want %q", status, out, want)
		}
	}
}

func TestAgeLabel(t *testing.T) {
	now := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		at   time.Time
		want string
	}{
		{now.Add(-30 * time.Second), "30s"},
		{now.Add(-5 * time.Minute), "5m"},
		{now.Add(-2 * time.Hour), "2h"},
		{time.Time{}, "-"},
	}
	for _, tc := range cases {
		got := AgeLabel(now, tc.at)
		if got != tc.want {
			t.Fatalf("AgeLabel(%v) = %q, want %q", tc.at, got, tc.want)
		}
	}
}

func TestDurationLabel(t *testing.T) {
	cases := []struct {
		secs int64
		want string
	}{
		{0, "0s"},
		{45, "45s"},
		{120, "2m 00s"},
		{3661, "1h 01m"},
	}
	for _, tc := range cases {
		got := DurationLabel(tc.secs)
		if got != tc.want {
			t.Fatalf("DurationLabel(%d) = %q, want %q", tc.secs, got, tc.want)
		}
	}
}

func TestCostLabel(t *testing.T) {
	out := CostLabel(4.82)
	if !strings.Contains(out, "$4.82") {
		t.Fatalf("CostLabel = %q, want $4.82", out)
	}
}

func TestTrimTo(t *testing.T) {
	if got := TrimTo("hello world", 5); got != "he…ld" {
		t.Fatalf("TrimTo = %q", got)
	}
	if got := TrimTo("hi", 10); got != "hi" {
		t.Fatalf("TrimTo short = %q", got)
	}
}
