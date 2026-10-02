package snapshot

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/poteto/noodle/adapter"
	"github.com/poteto/noodle/loop"
)

func TestSnapshotBacklogMapsAdapterFields(t *testing.T) {
	// Decode through the adapter's own unmarshal so Extra holds what a real
	// sync produces (tags arrive as []any).
	lines := []string{
		`{"id":"4","title":"Add search","status":"open","section":"Inbox","tags":["ui","search"],"estimate":"small"}`,
		`{"id":"9","title":"From a custom adapter","status":"done"}`,
	}
	items := make([]adapter.BacklogItem, 0, len(lines))
	for _, line := range lines {
		var item adapter.BacklogItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			t.Fatalf("decode backlog line: %v", err)
		}
		items = append(items, item)
	}

	snap, err := LoadSnapshot(t.TempDir(), time.Now(), loop.LoopState{
		Backlog:      items,
		BacklogError: "edit failed",
	})
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}

	if len(snap.Backlog) != 2 {
		t.Fatalf("backlog = %+v, want 2 items", snap.Backlog)
	}
	full := snap.Backlog[0]
	if full.ID != "4" || full.Title != "Add search" || full.Status != "open" ||
		full.Section != "Inbox" || full.Estimate != "small" ||
		len(full.Tags) != 2 || full.Tags[0] != "ui" || full.Tags[1] != "search" {
		t.Fatalf("mapped item = %+v", full)
	}
	bare := snap.Backlog[1]
	if bare.Section != "" || bare.Estimate != "" || bare.Tags == nil || len(bare.Tags) != 0 {
		t.Fatalf("item without adapter extras = %+v, want empty fields and an empty tag list", bare)
	}
	if snap.BacklogError != "edit failed" {
		t.Fatalf("BacklogError = %q", snap.BacklogError)
	}

	// The UI reads an array, never null, even with no backlog at all.
	empty, err := LoadSnapshot(t.TempDir(), time.Now(), loop.LoopState{})
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	encoded, err := json.Marshal(empty)
	if err != nil {
		t.Fatalf("encode snapshot: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if string(decoded["backlog"]) != "[]" {
		t.Fatalf(`snapshot "backlog" = %s, want []`, decoded["backlog"])
	}
}
