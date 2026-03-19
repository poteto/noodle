package projection

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/mode"
	"github.com/poteto/noodle/internal/orderx"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/statever"
)

func TestProjectProducesDeterministicOutput(t *testing.T) {
	t.Parallel()

	st := fixtureStateForProjection()
	modeState := mode.ModeState{
		EffectiveMode: state.RunModeSupervised,
		Epoch:         42,
	}

	first := mustProject(t, st, modeState)
	second := mustProject(t, st, modeState)

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("projection changed across identical inputs:\nfirst=%+v\nsecond=%+v", first, second)
	}
}

func TestProjectMapsAllLifecycleStatuses(t *testing.T) {
	t.Parallel()

	stageStatuses := []state.StageLifecycleStatus{
		state.StagePending,
		state.StageDispatching,
		state.StageRunning,
		state.StageMerging,
		state.StageReview,
		state.StageCompleted,
		state.StageFailed,
		state.StageSkipped,
		state.StageCancelled,
	}
	orderStatuses := []state.OrderLifecycleStatus{
		state.OrderPending,
		state.OrderActive,
		state.OrderCompleted,
		state.OrderFailed,
		state.OrderCancelled,
	}

	orders := make(map[string]state.OrderNode, len(orderStatuses))
	now := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	for i, orderStatus := range orderStatuses {
		stages := make([]state.StageNode, 0, len(stageStatuses))
		for j, stageStatus := range stageStatuses {
			stages = append(stages, state.StageNode{
				StageIndex: j,
				Status:     stageStatus,
				Skill:      "skill",
				Runtime:    "runtime",
				Group:      1,
			})
		}
		orderID := "order-" + string(rune('a'+i))
		orders[orderID] = state.OrderNode{
			OrderID:   orderID,
			Status:    orderStatus,
			Stages:    stages,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	bundle := mustProject(t, state.State{
		Orders:        orders,
		Mode:          state.RunModeAuto,
		SchemaVersion: statever.Current,
		LastEventID:   "5",
	}, mode.ModeState{EffectiveMode: state.RunModeAuto, Epoch: 3})

	gotOrders := make(map[string]string, len(bundle.OrdersProjection))
	for _, o := range bundle.OrdersProjection {
		gotOrders[o.ID] = o.Status
		for i, s := range o.Stages {
			if s.Status != string(stageStatuses[i]) {
				t.Fatalf("order %s stage %d status = %q, want %q", o.ID, i, s.Status, stageStatuses[i])
			}
		}
	}

	for i, orderStatus := range orderStatuses {
		orderID := "order-" + string(rune('a'+i))
		if gotOrders[orderID] != string(orderStatus) {
			t.Fatalf("order %s status = %q, want %q", orderID, gotOrders[orderID], orderStatus)
		}
	}
}

func TestProjectRepresentativeLifecycleScenarios(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 2, 15, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		state state.State
		want  []OrderProjection
	}{
		{
			name: "running stage remains visible with pending tail",
			state: state.State{
				Orders: map[string]state.OrderNode{
					"pipeline-1": {
						OrderID:   "pipeline-1",
						Status:    state.OrderActive,
						CreatedAt: now,
						UpdatedAt: now,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageRunning, Skill: "execute", Runtime: "process"},
							{StageIndex: 1, Status: state.StagePending, Skill: "reflect", Runtime: "process"},
						},
					},
				},
				Mode:          state.RunModeAuto,
				SchemaVersion: statever.Current,
				LastEventID:   "12",
			},
			want: []OrderProjection{{
				ID:     "pipeline-1",
				Status: "active",
				Stages: []StageProjection{
					{Skill: "execute", Runtime: "process", Status: "running"},
					{Skill: "reflect", Runtime: "process", Status: "pending"},
				},
			}},
		},
		{
			name: "merge review remains visible without mutating next stage",
			state: state.State{
				Orders: map[string]state.OrderNode{
					"conflict-1": {
						OrderID:   "conflict-1",
						Status:    state.OrderActive,
						CreatedAt: now,
						UpdatedAt: now,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageReview, Skill: "execute", Runtime: "process"},
							{StageIndex: 1, Status: state.StagePending, Skill: "reflect", Runtime: "process"},
						},
					},
				},
				Mode:          state.RunModeAuto,
				SchemaVersion: statever.Current,
				LastEventID:   "18",
			},
			want: []OrderProjection{{
				ID:     "conflict-1",
				Status: "active",
				Stages: []StageProjection{
					{Skill: "execute", Runtime: "process", Status: "review"},
					{Skill: "reflect", Runtime: "process", Status: "pending"},
				},
			}},
		},
		{
			name: "merge completion advances the next stage but keeps order active",
			state: state.State{
				Orders: map[string]state.OrderNode{
					"advance-1": {
						OrderID:   "advance-1",
						Status:    state.OrderActive,
						CreatedAt: now,
						UpdatedAt: now,
						Stages: []state.StageNode{
							{StageIndex: 0, Status: state.StageCompleted, Skill: "execute", Runtime: "process"},
							{StageIndex: 1, Status: state.StagePending, Skill: "reflect", Runtime: "process"},
						},
					},
				},
				Mode:          state.RunModeAuto,
				SchemaVersion: statever.Current,
				LastEventID:   "24",
			},
			want: []OrderProjection{{
				ID:     "advance-1",
				Status: "active",
				Stages: []StageProjection{
					{Skill: "execute", Runtime: "process", Status: "completed"},
					{Skill: "reflect", Runtime: "process", Status: "pending"},
				},
			}},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			bundle := mustProject(t, tc.state, mode.ModeState{EffectiveMode: tc.state.Mode, Epoch: tc.state.ModeEpoch})
			if reflect.DeepEqual(bundle.OrdersProjection, tc.want) {
				return
			}
			t.Fatalf("orders projection mismatch\nactual: %#v\nwant: %#v", bundle.OrdersProjection, tc.want)
		})
	}
}

func TestProjectionHashDeterministicAndChangesWhenStateChanges(t *testing.T) {
	t.Parallel()

	st := fixtureStateForProjection()
	modeState := mode.ModeState{EffectiveMode: state.RunModeAuto, Epoch: 1}
	bundle := mustProject(t, st, modeState)

	hashA, err := ComputeHash(bundle)
	if err != nil {
		t.Fatalf("compute hash A: %v", err)
	}
	hashB, err := ComputeHash(bundle)
	if err != nil {
		t.Fatalf("compute hash B: %v", err)
	}
	if hashA != hashB {
		t.Fatalf("hash changed across identical bundle: %q vs %q", hashA, hashB)
	}

	st2 := fixtureStateForProjection()
	order := st2.Orders["order-1"]
	order.Stages[0].Status = state.StageRunning
	order.UpdatedAt = order.UpdatedAt.Add(5 * time.Second)
	st2.Orders["order-1"] = order
	st2.LastEventID = "13"

	changed := mustProject(t, st2, modeState)
	if hashA == changed.Hash {
		t.Fatalf("hash did not change after projection content changed: %q", hashA)
	}
}

func TestProjectionHashIdenticalForIdenticalStates(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 28, 12, 30, 0, 0, time.UTC)
	stateA := state.State{
		Orders: map[string]state.OrderNode{
			"b": {
				OrderID:   "b",
				Status:    state.OrderActive,
				CreatedAt: now,
				UpdatedAt: now,
				Stages: []state.StageNode{
					{StageIndex: 0, Skill: "x", Runtime: "go", Status: state.StagePending, Group: 1},
				},
			},
			"a": {
				OrderID:   "a",
				Status:    state.OrderCompleted,
				CreatedAt: now,
				UpdatedAt: now,
				Stages: []state.StageNode{
					{StageIndex: 0, Skill: "y", Runtime: "go", Status: state.StageCompleted, Group: 2},
				},
			},
		},
		Mode:          state.RunModeManual,
		SchemaVersion: statever.Current,
		LastEventID:   "9",
	}
	stateB := state.State{
		Orders: map[string]state.OrderNode{
			"a": stateA.Orders["a"],
			"b": stateA.Orders["b"],
		},
		Mode:          stateA.Mode,
		SchemaVersion: stateA.SchemaVersion,
		LastEventID:   stateA.LastEventID,
	}

	modeState := mode.ModeState{EffectiveMode: state.RunModeManual, Epoch: 7}
	bundleA := mustProject(t, stateA, modeState)
	bundleB := mustProject(t, stateB, modeState)

	if bundleA.Hash != bundleB.Hash {
		t.Fatalf("identical state produced different hashes: %q vs %q", bundleA.Hash, bundleB.Hash)
	}
}

func TestWriteProjectionFilesCreatesValidJSONFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bundle := mustProject(t, fixtureStateForProjection(), mode.ModeState{
		EffectiveMode: state.RunModeSupervised,
		Epoch:         22,
	})

	if err := WriteProjectionFiles(dir, bundle); err != nil {
		t.Fatalf("write projection files: %v", err)
	}

	ordersData, err := os.ReadFile(filepath.Join(dir, ordersFileName))
	if err != nil {
		t.Fatalf("read orders file: %v", err)
	}
	var orders orderx.OrdersFile
	if err := json.Unmarshal(ordersData, &orders); err != nil {
		t.Fatalf("decode orders file: %v", err)
	}
	expectedOrders := 0
	for _, order := range bundle.OrdersProjection {
		if legacyOrderRemoves(order.Status) {
			continue
		}
		expectedOrders++
	}
	if len(orders.Orders) != expectedOrders {
		t.Fatalf("orders count = %d, want %d", len(orders.Orders), expectedOrders)
	}

	stateData, err := os.ReadFile(filepath.Join(dir, stateFileName))
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	var marker statever.StateMarker
	if err := json.Unmarshal(stateData, &marker); err != nil {
		t.Fatalf("decode state marker: %v", err)
	}
	if marker.SchemaVersion != bundle.StateMarker.SchemaVersion {
		t.Fatalf("schema version = %d, want %d", marker.SchemaVersion, bundle.StateMarker.SchemaVersion)
	}
}

func TestWriteProjectionFilesAtomicWriteLeavesNoTempFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bundle := mustProject(t, fixtureStateForProjection(), mode.ModeState{
		EffectiveMode: state.RunModeAuto,
		Epoch:         1,
	})

	if err := WriteProjectionFiles(dir, bundle); err != nil {
		t.Fatalf("first write projection files: %v", err)
	}
	if err := WriteProjectionFiles(dir, bundle); err != nil {
		t.Fatalf("second write projection files: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.tmp.*"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("leftover temp files: %v", matches)
	}
}

func TestComputeDeltaDetectsAddedRemovedAndChangedOrders(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 28, 16, 0, 0, 0, time.UTC)
	prevState := state.State{
		Orders: map[string]state.OrderNode{
			"order-a": {
				OrderID:   "order-a",
				Status:    state.OrderActive,
				CreatedAt: now,
				UpdatedAt: now,
				Stages: []state.StageNode{
					{StageIndex: 0, Skill: "execute", Runtime: "go", Status: state.StagePending, Group: 1},
				},
			},
			"order-b": {
				OrderID:   "order-b",
				Status:    state.OrderFailed,
				CreatedAt: now,
				UpdatedAt: now,
				Stages: []state.StageNode{
					{StageIndex: 0, Skill: "debugging", Runtime: "go", Status: state.StageFailed, Group: 1},
				},
			},
		},
		Mode:          state.RunModeAuto,
		SchemaVersion: statever.Current,
		LastEventID:   "100",
	}
	currState := state.State{
		Orders: map[string]state.OrderNode{
			"order-a": {
				OrderID:   "order-a",
				Status:    state.OrderActive,
				CreatedAt: now,
				UpdatedAt: now.Add(1 * time.Minute),
				Stages: []state.StageNode{
					{StageIndex: 0, Skill: "execute", Runtime: "go", Status: state.StageRunning, Group: 1},
				},
			},
			"order-c": {
				OrderID:   "order-c",
				Status:    state.OrderPending,
				CreatedAt: now,
				UpdatedAt: now,
				Stages: []state.StageNode{
					{StageIndex: 0, Skill: "quality", Runtime: "go", Status: state.StagePending, Group: 2},
				},
			},
		},
		Mode:          state.RunModeManual,
		SchemaVersion: statever.Current,
		LastEventID:   "101",
	}

	previous := mustProject(t, prevState, mode.ModeState{EffectiveMode: state.RunModeAuto, Epoch: 9})
	current := mustProject(t, currState, mode.ModeState{EffectiveMode: state.RunModeManual, Epoch: 10})
	delta, err := ComputeDelta(previous, current)
	if err != nil {
		t.Fatalf("compute delta: %v", err)
	}

	if delta.Version != current.Version {
		t.Fatalf("delta version = %d, want %d", delta.Version, current.Version)
	}
	if delta.PreviousVersion != previous.Version {
		t.Fatalf("delta previous version = %d, want %d", delta.PreviousVersion, previous.Version)
	}

	changeByPath := make(map[string]DeltaChange, len(delta.Changes))
	for _, change := range delta.Changes {
		changeByPath[change.Path] = change
	}

	if c, ok := changeByPath["orders.order-b"]; !ok || c.Op != string(DeltaOpDelete) {
		t.Fatalf("missing delete for removed order-b: %+v", c)
	}
	if c, ok := changeByPath["orders.order-c"]; !ok || c.Op != string(DeltaOpSet) {
		t.Fatalf("missing set for added order-c: %+v", c)
	}
	if c, ok := changeByPath["orders.order-a.stages.0.status"]; !ok || c.Op != string(DeltaOpSet) {
		t.Fatalf("missing set for changed order-a stage status: %+v", c)
	}
}

func TestComputeDeltaIdenticalBundlesReturnsEmptyDelta(t *testing.T) {
	t.Parallel()

	bundle := mustProject(t, fixtureStateForProjection(), mode.ModeState{
		EffectiveMode: state.RunModeAuto,
		Epoch:         2,
	})
	delta, err := ComputeDelta(bundle, bundle)
	if err != nil {
		t.Fatalf("compute delta: %v", err)
	}

	if len(delta.Changes) != 0 {
		t.Fatalf("identical bundles should produce no changes, got %d", len(delta.Changes))
	}
	if delta.Version != bundle.Version || delta.PreviousVersion != bundle.Version {
		t.Fatalf("delta versions mismatch for identical bundles: %+v", delta)
	}
}

func TestVersionCursorTracksLastSeenAndBackfillNeed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cursor  VersionCursor
		current ProjectionVersion
		want    bool
	}{
		{name: "same version", cursor: VersionCursor{LastSeen: 8}, current: 8, want: false},
		{name: "next version", cursor: VersionCursor{LastSeen: 8}, current: 9, want: false},
		{name: "gap needs backfill", cursor: VersionCursor{LastSeen: 8}, current: 11, want: true},
		{name: "client ahead needs backfill", cursor: VersionCursor{LastSeen: 8}, current: 7, want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.cursor.NeedsBackfill(tt.current); got != tt.want {
				t.Fatalf("NeedsBackfill(%d) = %v, want %v", tt.current, got, tt.want)
			}
		})
	}
}

func TestSnapshotViewContainsAllRequiredFields(t *testing.T) {
	t.Parallel()

	st := fixtureStateForProjection()
	modeState := mode.ModeState{
		EffectiveMode: state.RunModeSupervised,
		Epoch:         55,
	}
	bundle := mustProject(t, st, modeState)

	snap := bundle.SnapshotView
	if snap.Mode != string(modeState.EffectiveMode) {
		t.Fatalf("snapshot mode = %q, want %q", snap.Mode, modeState.EffectiveMode)
	}
	if snap.ModeEpoch != uint64(modeState.Epoch) {
		t.Fatalf("snapshot mode_epoch = %d, want %d", snap.ModeEpoch, modeState.Epoch)
	}
	if snap.SchemaVersion != int(st.SchemaVersion) {
		t.Fatalf("snapshot schema_version = %d, want %d", snap.SchemaVersion, st.SchemaVersion)
	}
	if snap.LastEventID != st.LastEventID {
		t.Fatalf("snapshot last_event_id = %q, want %q", snap.LastEventID, st.LastEventID)
	}
	if len(snap.Orders) == 0 {
		t.Fatal("snapshot orders should not be empty for non-empty state")
	}
	if snap.GeneratedAt.IsZero() {
		t.Fatal("snapshot generated_at should be non-zero for this fixture")
	}
}

func TestOrderProjectionMapsFromOrderNode(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 28, 18, 0, 0, 0, time.UTC)
	st := state.State{
		Orders: map[string]state.OrderNode{
			"order-1": {
				OrderID:   "order-1",
				Status:    state.OrderCancelled,
				CreatedAt: now,
				UpdatedAt: now,
				Stages: []state.StageNode{
					{
						StageIndex: 0,
						Skill:      "execute",
						Runtime:    "go",
						Status:     state.StageCancelled,
						Group:      9,
					},
				},
			},
		},
		Mode:          state.RunModeManual,
		SchemaVersion: statever.Current,
		LastEventID:   "7",
	}

	bundle := mustProject(t, st, mode.ModeState{
		EffectiveMode: state.RunModeManual,
		Epoch:         3,
	})

	if len(bundle.OrdersProjection) != 1 {
		t.Fatalf("orders projection count = %d, want 1", len(bundle.OrdersProjection))
	}
	order := bundle.OrdersProjection[0]
	if order.ID != "order-1" || order.Status != string(state.OrderCancelled) {
		t.Fatalf("order projection mismatch: %+v", order)
	}
	if len(order.Stages) != 1 {
		t.Fatalf("stage projection count = %d, want 1", len(order.Stages))
	}
	stage := order.Stages[0]
	if stage.Skill != "execute" || stage.Runtime != "go" || stage.Status != string(state.StageCancelled) || stage.Group != 9 {
		t.Fatalf("stage projection mismatch: %+v", stage)
	}
}

func TestProjectionTypesJSONRoundTrip(t *testing.T) {
	t.Parallel()

	bundle := mustProject(t, fixtureStateForProjection(), mode.ModeState{
		EffectiveMode: state.RunModeAuto,
		Epoch:         4,
	})
	delta, err := ComputeDelta(bundle, bundle)
	if err != nil {
		t.Fatalf("compute delta: %v", err)
	}
	change := DeltaChange{
		Path:  "orders.order-1.status",
		Op:    string(DeltaOpSet),
		Value: json.RawMessage(`"active"`),
	}
	cursor := VersionCursor{LastSeen: 9}

	assertJSONRoundTrip(t, bundle)
	assertJSONRoundTrip(t, bundle.OrdersProjection[0])
	assertJSONRoundTrip(t, bundle.OrdersProjection[0].Stages[0])
	assertJSONRoundTrip(t, bundle.SnapshotView)
	assertJSONRoundTrip(t, delta)
	assertJSONRoundTrip(t, change)
	assertJSONRoundTrip(t, cursor)
}

func TestProjectEdgeCases(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 28, 20, 0, 0, 0, time.UTC)
	manyOrders := make(map[string]state.OrderNode, 120)
	for i := 119; i >= 0; i-- {
		orderID := formatOrderID(i)
		manyOrders[orderID] = state.OrderNode{
			OrderID:   orderID,
			Status:    state.OrderActive,
			CreatedAt: now,
			UpdatedAt: now,
			Stages: []state.StageNode{
				{StageIndex: 0, Skill: "execute", Runtime: "go", Status: state.StagePending, Group: 1},
			},
		}
	}

	tests := []struct {
		name       string
		state      state.State
		modeState  mode.ModeState
		assertions func(t *testing.T, bundle ProjectionBundle)
	}{
		{
			name: "empty state",
			state: state.State{
				Orders:        map[string]state.OrderNode{},
				Mode:          state.RunModeAuto,
				SchemaVersion: statever.Current,
				LastEventID:   "",
			},
			modeState: mode.ModeState{EffectiveMode: state.RunModeAuto, Epoch: 0},
			assertions: func(t *testing.T, bundle ProjectionBundle) {
				if len(bundle.OrdersProjection) != 0 {
					t.Fatalf("orders projection = %d, want 0", len(bundle.OrdersProjection))
				}
				if bundle.Version != 0 {
					t.Fatalf("version = %d, want 0", bundle.Version)
				}
			},
		},
		{
			name: "single order",
			state: state.State{
				Orders: map[string]state.OrderNode{
					"single": {
						OrderID:   "single",
						Status:    state.OrderActive,
						CreatedAt: now,
						UpdatedAt: now,
						Stages: []state.StageNode{
							{StageIndex: 0, Skill: "execute", Runtime: "go", Status: state.StagePending, Group: 1},
						},
					},
				},
				Mode:          state.RunModeSupervised,
				SchemaVersion: statever.Current,
				LastEventID:   "1",
			},
			modeState: mode.ModeState{EffectiveMode: state.RunModeSupervised, Epoch: 1},
			assertions: func(t *testing.T, bundle ProjectionBundle) {
				if len(bundle.OrdersProjection) != 1 {
					t.Fatalf("orders projection = %d, want 1", len(bundle.OrdersProjection))
				}
			},
		},
		{
			name: "many orders sorted deterministically",
			state: state.State{
				Orders:        manyOrders,
				Mode:          state.RunModeAuto,
				SchemaVersion: statever.Current,
				LastEventID:   "900",
			},
			modeState: mode.ModeState{EffectiveMode: state.RunModeAuto, Epoch: 2},
			assertions: func(t *testing.T, bundle ProjectionBundle) {
				if len(bundle.OrdersProjection) != 120 {
					t.Fatalf("orders projection = %d, want 120", len(bundle.OrdersProjection))
				}
				gotIDs := make([]string, 0, len(bundle.OrdersProjection))
				for _, order := range bundle.OrdersProjection {
					gotIDs = append(gotIDs, order.ID)
				}
				wantIDs := append([]string(nil), gotIDs...)
				sort.Strings(wantIDs)
				if !reflect.DeepEqual(gotIDs, wantIDs) {
					t.Fatalf("orders are not sorted deterministically")
				}
			},
		},
		{
			name: "all terminal states",
			state: state.State{
				Orders: map[string]state.OrderNode{
					"done": {
						OrderID:   "done",
						Status:    state.OrderCompleted,
						CreatedAt: now,
						UpdatedAt: now,
						Stages: []state.StageNode{
							{StageIndex: 0, Skill: "execute", Runtime: "go", Status: state.StageCompleted, Group: 1},
							{StageIndex: 1, Skill: "quality", Runtime: "go", Status: state.StageSkipped, Group: 1},
						},
					},
					"failed": {
						OrderID:   "failed",
						Status:    state.OrderFailed,
						CreatedAt: now,
						UpdatedAt: now,
						Stages: []state.StageNode{
							{StageIndex: 0, Skill: "execute", Runtime: "go", Status: state.StageFailed, Group: 1},
							{StageIndex: 1, Skill: "cleanup", Runtime: "go", Status: state.StageCancelled, Group: 1},
						},
					},
				},
				Mode:          state.RunModeManual,
				SchemaVersion: statever.Current,
				LastEventID:   "777",
			},
			modeState: mode.ModeState{EffectiveMode: state.RunModeManual, Epoch: 3},
			assertions: func(t *testing.T, bundle ProjectionBundle) {
				if len(bundle.OrdersProjection) != 2 {
					t.Fatalf("orders projection = %d, want 2", len(bundle.OrdersProjection))
				}
				if bundle.Version != 777 {
					t.Fatalf("version = %d, want 777", bundle.Version)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bundle := mustProject(t, tt.state, tt.modeState)
			tt.assertions(t, bundle)
		})
	}
}

func mustProject(t *testing.T, s state.State, modeState mode.ModeState) ProjectionBundle {
	t.Helper()
	bundle, err := Project(s, modeState)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	return bundle
}

func fixtureStateForProjection() state.State {
	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	exitCode := 0
	return state.State{
		Orders: map[string]state.OrderNode{
			"order-2": {
				OrderID:   "order-2",
				Status:    state.OrderCompleted,
				CreatedAt: now.Add(-2 * time.Hour),
				UpdatedAt: now.Add(-time.Hour),
				Stages: []state.StageNode{
					{
						StageIndex: 0,
						Status:     state.StageCompleted,
						Skill:      "quality",
						Runtime:    "go",
						Group:      2,
						Attempts: []state.AttemptNode{
							{
								AttemptID:    "att-2",
								SessionID:    "sess-2",
								Status:       state.AttemptCompleted,
								StartedAt:    now.Add(-95 * time.Minute),
								CompletedAt:  now.Add(-90 * time.Minute),
								ExitCode:     &exitCode,
								WorktreeName: "wt-2",
							},
						},
					},
				},
			},
			"order-1": {
				OrderID:   "order-1",
				Status:    state.OrderActive,
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now,
				Stages: []state.StageNode{
					{
						StageIndex: 0,
						Status:     state.StagePending,
						Skill:      "execute",
						Runtime:    "go",
						Group:      1,
						Attempts:   []state.AttemptNode{},
					},
				},
			},
		},
		Mode:          state.RunModeAuto,
		SchemaVersion: statever.Current,
		LastEventID:   "12",
	}
}

func assertJSONRoundTrip[T any](t *testing.T, value T) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %T: %v", value, err)
	}
	var decoded T
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal %T: %v", value, err)
	}
	if !reflect.DeepEqual(value, decoded) {
		t.Fatalf("round-trip mismatch for %T\noriginal=%+v\ndecoded=%+v", value, value, decoded)
	}
}

func formatOrderID(i int) string {
	return fmt.Sprintf("order-%03d", i)
}
