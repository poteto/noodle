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

func TestPlanSpawnItemsStopsWhenBlockingCookAlreadyActive(t *testing.T) {
	t.Parallel()

	plan := planSpawnItems(spawnPlanInput{
		QueueItems: []QueueItem{
			{ID: "a"},
			{ID: "b"},
		},
		Capacity:       2,
		BlockingActive: true,
	})
	if len(plan) != 0 {
		t.Fatalf("expected empty plan when blocking cook already active, got %v", queueItemIDs(plan))
	}
}

func TestPlanSpawnItemsRespectsBlockingItemRules(t *testing.T) {
	t.Parallel()

	isBlocking := func(item QueueItem) bool {
		return item.ID == "blocking"
	}

	planWithNoActive := planSpawnItems(spawnPlanInput{
		QueueItems: []QueueItem{
			{ID: "blocking"},
			{ID: "next"},
		},
		Capacity:   3,
		IsBlocking: isBlocking,
	})
	if got, want := queueItemIDs(planWithNoActive), []string{"blocking"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("blocking-first plan mismatch: got %v want %v", got, want)
	}

	planWithActive := planSpawnItems(spawnPlanInput{
		QueueItems: []QueueItem{
			{ID: "blocking"},
			{ID: "next"},
		},
		Capacity:    3,
		ActiveCount: 1,
		IsBlocking:  isBlocking,
	})
	if got, want := queueItemIDs(planWithActive), []string{"next"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("blocking-skip plan mismatch: got %v want %v", got, want)
	}
}

func queueItemIDs(items []QueueItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
