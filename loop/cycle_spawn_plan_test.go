package loop

import (
	"reflect"
	"testing"
)

func TestPlanSpawnItemsSkipsBusyFailedAdoptedAndTicketedTargets(t *testing.T) {
	t.Parallel()

	plan := planSpawnItems(spawnPlanInput{
		QueueItems: []QueueItem{
			{ID: "busy"},
			{ID: "failed"},
			{ID: "adopted"},
			{ID: "ticket"},
			{ID: "eligible-1"},
			{ID: "eligible-2"},
		},
		Capacity:        3,
		ActiveCount:     1,
		BusyTargets:     map[string]struct{}{"busy": {}},
		FailedTargets:   map[string]struct{}{"failed": {}},
		AdoptedTargets:  map[string]struct{}{"adopted": {}},
		TicketedTargets: map[string]struct{}{"ticket": {}},
	})

	got := queueItemIDs(plan)
	want := []string{"eligible-1", "eligible-2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("spawn plan IDs mismatch: got %v want %v", got, want)
	}
}

func TestPlanSpawnItemsRespectsCapacityWithExistingActiveCook(t *testing.T) {
	t.Parallel()

	plan := planSpawnItems(spawnPlanInput{
		QueueItems: []QueueItem{
			{ID: "a"},
			{ID: "b"},
			{ID: "c"},
		},
		Capacity:    3,
		ActiveCount: 1,
	})
	if got, want := queueItemIDs(plan), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("capacity plan mismatch: got %v want %v", got, want)
	}
}

func queueItemIDs(items []QueueItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
