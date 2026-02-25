package loop

import "testing"

func TestHasNonPrioritizeItems(t *testing.T) {
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
			name: "only prioritize item",
			queue: Queue{
				Items: []QueueItem{
					{ID: "prioritize"},
				},
			},
			want: false,
		},
		{
			name: "one non-prioritize item",
			queue: Queue{
				Items: []QueueItem{
					{ID: "42"},
				},
			},
			want: true,
		},
		{
			name: "mixed prioritize and non-prioritize items",
			queue: Queue{
				Items: []QueueItem{
					{ID: "prioritize"},
					{ID: "42"},
				},
			},
			want: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := hasNonPrioritizeItems(testCase.queue)
			if got != testCase.want {
				t.Fatalf("hasNonPrioritizeItems() = %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestFilterStalePrioritizeItems(t *testing.T) {
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
			name: "only prioritize item",
			queue: Queue{
				Items: []QueueItem{
					{ID: "prioritize"},
				},
			},
			wantIDs: []string{},
		},
		{
			name: "non-prioritize items unchanged",
			queue: Queue{
				Items: []QueueItem{
					{ID: "42"},
					{ID: "43"},
				},
			},
			wantIDs: []string{"42", "43"},
		},
		{
			name: "mixed items remove prioritize",
			queue: Queue{
				Items: []QueueItem{
					{ID: "42"},
					{ID: "prioritize"},
					{ID: "43"},
				},
			},
			wantIDs: []string{"42", "43"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			filtered := filterStalePrioritizeItems(testCase.queue)
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
