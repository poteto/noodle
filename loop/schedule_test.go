package loop

import "testing"

func TestHasNonScheduleItems(t *testing.T) {
	testCases := []struct {
		name  string
		queue Queue
		want  bool
	}{
		{
			name:  "empty queue",
			queue: Queue{},
			want:  false,
		},
		{
			name: "only schedule item",
			queue: Queue{
				Items: []QueueItem{
					{ID: "schedule"},
				},
			},
			want: false,
		},
		{
			name: "one non-schedule item",
			queue: Queue{
				Items: []QueueItem{
					{ID: "42"},
				},
			},
			want: true,
		},
		{
			name: "mixed schedule and non-schedule items",
			queue: Queue{
				Items: []QueueItem{
					{ID: "schedule"},
					{ID: "42"},
				},
			},
			want: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := hasNonScheduleItems(testCase.queue)
			if got != testCase.want {
				t.Fatalf("hasNonScheduleItems() = %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestFilterStaleScheduleItems(t *testing.T) {
	testCases := []struct {
		name    string
		queue   Queue
		wantIDs []string
	}{
		{
			name:    "empty queue",
			queue:   Queue{},
			wantIDs: nil,
		},
		{
			name: "only schedule item",
			queue: Queue{
				Items: []QueueItem{
					{ID: "schedule"},
				},
			},
			wantIDs: []string{},
		},
		{
			name: "non-schedule items unchanged",
			queue: Queue{
				Items: []QueueItem{
					{ID: "42"},
					{ID: "43"},
				},
			},
			wantIDs: []string{"42", "43"},
		},
		{
			name: "mixed items remove schedule",
			queue: Queue{
				Items: []QueueItem{
					{ID: "42"},
					{ID: "schedule"},
					{ID: "43"},
				},
			},
			wantIDs: []string{"42", "43"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			filtered := filterStaleScheduleItems(testCase.queue)
			if len(filtered.Items) != len(testCase.wantIDs) {
				t.Fatalf("len(items) = %d, want %d", len(filtered.Items), len(testCase.wantIDs))
			}
			for index, expectedID := range testCase.wantIDs {
				if got := filtered.Items[index].ID; got != expectedID {
					t.Fatalf("items[%d].ID = %q, want %q", index, got, expectedID)
				}
			}
		})
	}
}
