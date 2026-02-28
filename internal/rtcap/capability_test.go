package rtcap

import (
	"sort"
	"testing"
)

func TestRuntimeCapabilities_Has(t *testing.T) {
	caps := RuntimeCapabilities{
		Name: "test",
		Caps: map[Capability]bool{
			CapSteerable:  true,
			CapPolling:    false,
			CapRemoteSync: true,
		},
	}

	if !caps.Has(CapSteerable) {
		t.Error("expected Has(CapSteerable) = true")
	}
	if caps.Has(CapPolling) {
		t.Error("expected Has(CapPolling) = false")
	}
	if !caps.Has(CapRemoteSync) {
		t.Error("expected Has(CapRemoteSync) = true")
	}
	if caps.Has(CapHeartbeat) {
		t.Error("expected Has(CapHeartbeat) = false for absent key")
	}
}

func TestRuntimeCapabilities_All(t *testing.T) {
	caps := RuntimeCapabilities{
		Name: "test",
		Caps: map[Capability]bool{
			CapSteerable:  true,
			CapPolling:    false,
			CapRemoteSync: true,
			CapHeartbeat:  true,
		},
	}
	all := caps.All()
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })

	want := []Capability{CapHeartbeat, CapRemoteSync, CapSteerable}
	if len(all) != len(want) {
		t.Fatalf("All() returned %d caps, want %d", len(all), len(want))
	}
	for i, cap := range all {
		if cap != want[i] {
			t.Errorf("All()[%d] = %q, want %q", i, cap, want[i])
		}
	}
}

func TestRuntimeCapabilities_EmptyCaps(t *testing.T) {
	caps := RuntimeCapabilities{Name: "empty"}
	if caps.Has(CapSteerable) {
		t.Error("nil caps map should return false for Has()")
	}
	if len(caps.All()) != 0 {
		t.Error("nil caps map should return empty All()")
	}
}

func TestHelpers(t *testing.T) {
	process := ProcessCaps
	cursor := CursorCaps

	if !CanSteer(process) {
		t.Error("process should be steerable")
	}
	if NeedsPolling(process) {
		t.Error("process should not need polling")
	}
	if NeedsRemoteSync(process) {
		t.Error("process should not need remote sync")
	}

	if CanSteer(cursor) {
		t.Error("cursor should not be steerable")
	}
	if !NeedsPolling(cursor) {
		t.Error("cursor should need polling")
	}
	if !NeedsRemoteSync(cursor) {
		t.Error("cursor should need remote sync")
	}
}
