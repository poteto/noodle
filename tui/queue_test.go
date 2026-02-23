package tui

import (
	"strings"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func TestDeriveQueueStatusCooking(t *testing.T) {
	item := QueueItem{ID: "task-1", TaskKey: "execute"}
	active := map[string]struct{}{"task-1": {}}
	action := map[string]struct{}{}
	got := deriveQueueStatus(item, active, action)
	if got != QueueStatusCooking {
		t.Fatalf("status = %q, want %q", got, QueueStatusCooking)
	}
}

func TestDeriveQueueStatusReviewing(t *testing.T) {
	item := QueueItem{ID: "task-2", TaskKey: "execute"}
	active := map[string]struct{}{}
	action := map[string]struct{}{"task-2": {}}
	got := deriveQueueStatus(item, active, action)
	if got != QueueStatusReviewing {
		t.Fatalf("status = %q, want %q", got, QueueStatusReviewing)
	}
}

func TestDeriveQueueStatusPlanned(t *testing.T) {
	item := QueueItem{ID: "task-3", TaskKey: "execute", Review: boolPtr(true)}
	active := map[string]struct{}{}
	action := map[string]struct{}{}
	got := deriveQueueStatus(item, active, action)
	if got != QueueStatusPlanned {
		t.Fatalf("status = %q, want %q", got, QueueStatusPlanned)
	}
}

func TestDeriveQueueStatusNoPlan(t *testing.T) {
	item := QueueItem{ID: "task-4", TaskKey: "execute"}
	active := map[string]struct{}{}
	action := map[string]struct{}{}
	got := deriveQueueStatus(item, active, action)
	if got != QueueStatusNoPlan {
		t.Fatalf("status = %q, want %q", got, QueueStatusNoPlan)
	}
}

func TestDeriveQueueStatusReady(t *testing.T) {
	for _, taskKey := range []string{"reflect", "prioritize", "quality"} {
		item := QueueItem{ID: "task-5", TaskKey: taskKey}
		active := map[string]struct{}{}
		action := map[string]struct{}{}
		got := deriveQueueStatus(item, active, action)
		if got != QueueStatusReady {
			t.Fatalf("status(%s) = %q, want %q", taskKey, got, QueueStatusReady)
		}
	}
}

func TestQueueTabRenderAt80And120(t *testing.T) {
	qt := NewQueueTab()
	items := []QueueItem{
		{ID: "task-1", TaskKey: "execute", Title: "Implement login flow"},
		{ID: "task-2", TaskKey: "plan", Title: "Design API schema"},
		{ID: "task-3", TaskKey: "quality", Title: "Review PR #42"},
	}
	qt.SetQueue(items, []string{"task-1"}, nil)

	for _, width := range []int{80, 120} {
		out := qt.Render(width, 20)
		if out == "" {
			t.Fatalf("empty render at width %d", width)
		}
		if !strings.Contains(out, "cooked") {
			t.Fatalf("missing progress bar at width %d", width)
		}
	}
}

func TestQueueTabRendersEmptyState(t *testing.T) {
	qt := NewQueueTab()
	qt.SetQueue(nil, nil, nil)
	out := qt.Render(80, 20)
	if !strings.Contains(out, "queue empty") {
		t.Fatalf("expected empty state message, got %q", out)
	}
}

func TestQueueTabProgressBarCounts(t *testing.T) {
	qt := NewQueueTab()
	items := []QueueItem{
		{ID: "active-1", TaskKey: "execute", Title: "Task A"},
		{ID: "review-1", TaskKey: "execute", Title: "Task B"},
		{ID: "pending-1", TaskKey: "execute", Title: "Task C"},
	}
	qt.SetQueue(items, []string{"active-1"}, []string{"review-1"})
	out := qt.Render(80, 20)
	if !strings.Contains(out, "2/3 cooked") {
		t.Fatalf("expected '2/3 cooked' in output, got: %s", out)
	}
}

func TestQueueStatusAllLifecycleStates(t *testing.T) {
	cases := []struct {
		name   string
		item   QueueItem
		active map[string]struct{}
		action map[string]struct{}
		want   QueueStatus
	}{
		{
			name:   "cooking — active session",
			item:   QueueItem{ID: "a", TaskKey: "execute"},
			active: map[string]struct{}{"a": {}},
			want:   QueueStatusCooking,
		},
		{
			name:   "reviewing — action needed",
			item:   QueueItem{ID: "b", TaskKey: "execute"},
			action: map[string]struct{}{"b": {}},
			want:   QueueStatusReviewing,
		},
		{
			name: "planned — has review",
			item: QueueItem{ID: "c", TaskKey: "execute", Review: boolPtr(false)},
			want: QueueStatusPlanned,
		},
		{
			name: "no plan — execute without review",
			item: QueueItem{ID: "d", TaskKey: "execute"},
			want: QueueStatusNoPlan,
		},
		{
			name: "ready — reflect",
			item: QueueItem{ID: "e", TaskKey: "reflect"},
			want: QueueStatusReady,
		},
		{
			name: "ready — prioritize",
			item: QueueItem{ID: "f", TaskKey: "prioritize"},
			want: QueueStatusReady,
		},
		{
			name: "ready — quality",
			item: QueueItem{ID: "g", TaskKey: "quality"},
			want: QueueStatusReady,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			active := tc.active
			if active == nil {
				active = map[string]struct{}{}
			}
			action := tc.action
			if action == nil {
				action = map[string]struct{}{}
			}
			got := deriveQueueStatus(tc.item, active, action)
			if got != tc.want {
				t.Fatalf("deriveQueueStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNoPlanStatusUsesWarningColor(t *testing.T) {
	// Verify the style for "no plan" uses the warning color from the theme.
	style := statusStyleForQueue(QueueStatusNoPlan)
	rendered := style.Render("no plan")
	if rendered == "" {
		t.Fatal("expected non-empty styled output for no plan status")
	}
	// In test environments lipgloss may not emit ANSI codes, so we verify
	// the function returns the correct style object by comparing against
	// other status styles (cooking uses success, not warning).
	cookingStyle := statusStyleForQueue(QueueStatusCooking)
	if style.GetForeground() == cookingStyle.GetForeground() {
		t.Fatal("no plan style should differ from cooking style (warning vs success)")
	}
}

func TestTruncTitle(t *testing.T) {
	if got := truncTitle("My Task", "id-1"); got != "My Task" {
		t.Fatalf("truncTitle with title = %q", got)
	}
	if got := truncTitle("", "id-1"); got != "id-1" {
		t.Fatalf("truncTitle without title = %q", got)
	}
	if got := truncTitle("  ", "id-2"); got != "id-2" {
		t.Fatalf("truncTitle whitespace title = %q", got)
	}
}

func TestQueueTabSetQueueRowCount(t *testing.T) {
	qt := NewQueueTab()
	items := []QueueItem{
		{ID: "1", TaskKey: "execute", Title: "A"},
		{ID: "2", TaskKey: "plan", Title: "B"},
	}
	qt.SetQueue(items, nil, nil)
	if len(qt.table.Rows()) != 2 {
		t.Fatalf("row count = %d, want 2", len(qt.table.Rows()))
	}
}
