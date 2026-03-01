package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/dispatch"
	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/mode"
	"github.com/poteto/noodle/internal/projection"
	"github.com/poteto/noodle/internal/reducer"
	"github.com/poteto/noodle/internal/state"
	"github.com/poteto/noodle/internal/statever"
)

func TestResilienceVerification(t *testing.T) {
	t.Run("TestHighConcurrencyReducer", func(t *testing.T) {
		baseAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
		initial := makeState(50, 2, baseAt)
		events := lifecycleEvents(50, 2, 1, baseAt.Add(time.Second))
		if len(events) < 200 {
			t.Fatalf("event stream too small for concurrency stress: count=%d", len(events))
		}

		runA := applyEvents(initial, events, nil)
		runB := applyEvents(makeState(50, 2, baseAt), events, nil)

		hashA := hashState(runA.state)
		hashB := hashState(runB.state)

		deterministicMatchPercent := 0
		if hashA == hashB {
			deterministicMatchPercent = 100
		}
		if deterministicMatchPercent != 100 {
			t.Fatalf("deterministic hash match dropped: percent=%d hash_a=%s hash_b=%s", deterministicMatchPercent, hashA, hashB)
		}

		duplicateDispatches := duplicateCount(runA.dispatchEffectIDs)
		if duplicateDispatches != 0 {
			t.Fatalf("duplicate dispatch effects detected: count=%d", duplicateDispatches)
		}

		ordersA, stagesA := terminalCounts(runA.state)
		ordersB, stagesB := terminalCounts(runB.state)
		lostTerminalOrders := maxInt(0, ordersA-ordersB)
		lostTerminalStages := maxInt(0, stagesA-stagesB)
		if lostTerminalOrders != 0 || lostTerminalStages != 0 {
			t.Fatalf("terminal state loss detected after replay: lost_orders=%d lost_stages=%d", lostTerminalOrders, lostTerminalStages)
		}
	})

	t.Run("TestCrashWindowRecovery", func(t *testing.T) {
		baseAt := time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC)
		initial := makeState(24, 2, baseAt)
		events := lifecycleEvents(24, 2, 1, baseAt.Add(time.Second))
		fullRun := applyEvents(initial, events, nil)
		fullHash := hashState(fullRun.state)
		fullOrders, fullStages := terminalCounts(fullRun.state)

		t.Run("after event persistence before state/effect ledger commit", func(t *testing.T) {
			cut := len(events) / 3
			if cut == 0 {
				t.Fatalf("crash cut index unresolved for event stream length=%d", len(events))
			}

			beforeCrash := applyEvents(initial, events[:cut], nil)
			snapshot := reducer.BuildSnapshot(beforeCrash.state, beforeCrash.ledger, baseAt.Add(2*time.Minute))
			snapshot = writeReadSnapshot(t, snapshot)

			recovered := applyEvents(snapshot.State, events[cut:], ledgerFromRecords(snapshot.EffectLedger))
			recoveredHash := hashState(recovered.state)
			if recoveredHash != fullHash {
				t.Fatalf("recovered state hash diverged after crash window A: recovered=%s full=%s", recoveredHash, fullHash)
			}

			dupDispatch := duplicateCount(append(beforeCrash.dispatchEffectIDs, recovered.dispatchEffectIDs...))
			if dupDispatch != 0 {
				t.Fatalf("duplicate dispatch effects detected across restart window A: count=%d", dupDispatch)
			}

			recoveredOrders, recoveredStages := terminalCounts(recovered.state)
			lostOrders := maxInt(0, fullOrders-recoveredOrders)
			lostStages := maxInt(0, fullStages-recoveredStages)
			if lostOrders != 0 || lostStages != 0 {
				t.Fatalf("terminal states dropped after crash window A: lost_orders=%d lost_stages=%d", lostOrders, lostStages)
			}
		})

		t.Run("after external effect success before effect-result persistence", func(t *testing.T) {
			if len(events) < 2 {
				t.Fatalf("event stream too short for crash window B: count=%d", len(events))
			}

			beforeCrash := applyEvents(initial, events[:1], nil)
			if len(beforeCrash.dispatchEffectIDs) == 0 {
				t.Fatalf("dispatch effect unavailable before crash window B")
			}
			effectID := beforeCrash.dispatchEffectIDs[0]
			if err := beforeCrash.ledger.MarkRunning(effectID); err != nil {
				t.Fatalf("effect lifecycle transition failed before crash window B: %v", err)
			}

			snapshot := reducer.BuildSnapshot(beforeCrash.state, beforeCrash.ledger, baseAt.Add(3*time.Minute))
			snapshot = writeReadSnapshot(t, snapshot)

			recoveredLedger := ledgerFromRecords(snapshot.EffectLedger)
			replayedExternalDispatches := 0
			for _, rec := range recoveredLedger.InFlight() {
				if rec.Effect.Type == reducer.EffectDispatch && rec.EffectID == effectID {
					// External success already happened before crash; do not replay launch.
					continue
				}
				if rec.Effect.Type == reducer.EffectDispatch {
					replayedExternalDispatches++
				}
			}
			if replayedExternalDispatches != 0 {
				t.Fatalf("dispatch replay occurred after external success in crash window B: count=%d", replayedExternalDispatches)
			}

			if err := recoveredLedger.MarkDone(effectID, reducer.EffectResult{
				EffectID:  effectID,
				Status:    reducer.EffectResultCompleted,
				Timestamp: baseAt.Add(3*time.Minute + time.Second),
			}); err != nil {
				t.Fatalf("effect result persistence failed during crash window B recovery: %v", err)
			}

			recovered := applyEvents(snapshot.State, events[1:], recoveredLedger)
			recoveredHash := hashState(recovered.state)
			if recoveredHash != fullHash {
				t.Fatalf("recovered state hash diverged after crash window B: recovered=%s full=%s", recoveredHash, fullHash)
			}

			dupDispatch := duplicateCount(append(beforeCrash.dispatchEffectIDs, recovered.dispatchEffectIDs...))
			if dupDispatch != 0 {
				t.Fatalf("duplicate dispatch effects detected across restart window B: count=%d", dupDispatch)
			}

			recoveredOrders, recoveredStages := terminalCounts(recovered.state)
			lostOrders := maxInt(0, fullOrders-recoveredOrders)
			lostStages := maxInt(0, fullStages-recoveredStages)
			if lostOrders != 0 || lostStages != 0 {
				t.Fatalf("terminal states dropped after crash window B: lost_orders=%d lost_stages=%d", lostOrders, lostStages)
			}
		})

		t.Run("after projection write to temp before atomic rename", func(t *testing.T) {
			cut := len(events) / 2
			beforeCrash := applyEvents(initial, events[:cut], nil)
			snapshot := reducer.BuildSnapshot(beforeCrash.state, beforeCrash.ledger, baseAt.Add(4*time.Minute))

			projectionDir := t.TempDir()
			bundle := projection.Project(beforeCrash.state, mode.ModeState{EffectiveMode: beforeCrash.state.Mode})
			tempPath := writeInterruptedProjectionTemp(t, projectionDir, bundle)
			if _, err := os.Stat(tempPath); err != nil {
				t.Fatalf("projection temp artifact unavailable at crash window C boundary: %v", err)
			}

			snapshot = writeReadSnapshot(t, snapshot)
			recovered := applyEvents(snapshot.State, events[cut:], ledgerFromRecords(snapshot.EffectLedger))

			finalBundle := projection.Project(recovered.state, mode.ModeState{EffectiveMode: recovered.state.Mode})
			if err := projection.WriteProjectionFiles(projectionDir, finalBundle); err != nil {
				t.Fatalf("projection rewrite failed after crash window C: %v", err)
			}

			recoveredHash := hashState(recovered.state)
			if recoveredHash != fullHash {
				t.Fatalf("recovered state hash diverged after crash window C: recovered=%s full=%s", recoveredHash, fullHash)
			}

			if finalBundle.Hash != projection.Project(fullRun.state, mode.ModeState{EffectiveMode: fullRun.state.Mode}).Hash {
				t.Fatalf("projection hash diverged after crash window C recovery: recovered=%s full=%s", finalBundle.Hash, projection.Project(fullRun.state, mode.ModeState{EffectiveMode: fullRun.state.Mode}).Hash)
			}

			dupDispatch := duplicateCount(append(beforeCrash.dispatchEffectIDs, recovered.dispatchEffectIDs...))
			if dupDispatch != 0 {
				t.Fatalf("duplicate dispatch effects detected across restart window C: count=%d", dupDispatch)
			}

			recoveredOrders, recoveredStages := terminalCounts(recovered.state)
			lostOrders := maxInt(0, fullOrders-recoveredOrders)
			lostStages := maxInt(0, fullStages-recoveredStages)
			if lostOrders != 0 || lostStages != 0 {
				t.Fatalf("terminal states dropped after crash window C: lost_orders=%d lost_stages=%d", lostOrders, lostStages)
			}
		})
	})

	t.Run("TestReplayDeterminism", func(t *testing.T) {
		baseAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		initialA := makeState(20, 2, baseAt)
		initialB := makeState(20, 2, baseAt)
		stream := lifecycleEvents(20, 2, 1, baseAt.Add(time.Second))
		if len(stream) < 100 {
			t.Fatalf("event stream too small for replay determinism run: count=%d", len(stream))
		}
		stream = stream[:100]

		runA := applyEvents(initialA, stream, nil)
		runB := applyEvents(initialB, stream, nil)
		hashA := hashState(runA.state)
		hashB := hashState(runB.state)

		deterministicMatchPercent := 0
		if hashA == hashB {
			deterministicMatchPercent = 100
		}
		if deterministicMatchPercent != 100 {
			t.Fatalf("replay determinism dropped below full match: percent=%d hash_a=%s hash_b=%s", deterministicMatchPercent, hashA, hashB)
		}
	})

	t.Run("TestControlPressureUnderLoad", func(t *testing.T) {
		baseAt := time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC)
		current := makeState(16, 2, baseAt)
		gate := mode.ModeGate{}
		modeState := mode.NewModeState(state.RunModeAuto)
		nextEventID := ingest.EventID(1)
		stamped := make([]mode.StampedEffect, 0)
		blockedDispatches := 0
		allowedDispatches := 0

		modeSequence := []state.RunMode{
			state.RunModeSupervised,
			state.RunModeManual,
			state.RunModeAuto,
			state.RunModeSupervised,
			state.RunModeAuto,
		}
		modeCursor := 0

		for step := 0; step < 120; step++ {
			now := baseAt.Add(time.Duration(step) * 200 * time.Millisecond)
			if step%12 == 0 && modeCursor < len(modeSequence) {
				nextMode := modeSequence[modeCursor]
				modeCursor++
				modeState = mode.TransitionMode(modeState, nextMode, "integration", fmt.Sprintf("pressure-step-%d", step), now)
				current, _ = reducer.Reduce(current, makeEvent(nextEventID, ingest.EventModeChanged, map[string]any{
					"mode": string(nextMode),
				}, now))
				nextEventID++
			}

			plan := dispatch.PlanDispatches(current, 64, nil)
			if len(plan.Candidates) == 0 {
				continue
			}
			candidate := plan.Candidates[0]

			if !gate.CanDispatch(modeState.EffectiveMode) {
				blockedDispatches++
				reason := gate.BlockedReason(modeState.EffectiveMode, mode.ActionDispatch)
				if reason == "" {
					t.Fatalf("dispatch block reason missing under pressure mode=%s step=%d", modeState.EffectiveMode, step)
				}
				continue
			}

			allowedDispatches++
			attemptID := fmt.Sprintf("pressure-%03d-%d", step, candidate.StageIndex)
			request := makeEvent(nextEventID, ingest.EventDispatchRequested, map[string]any{
				"order_id":    candidate.OrderID,
				"stage_index": candidate.StageIndex,
				"attempt_id":  attemptID,
			}, now)
			nextEventID++
			nextState, effects := reducer.Reduce(current, request)
			current = nextState
			for _, effect := range effects {
				if effect.Type == reducer.EffectDispatch {
					stamped = append(stamped, mode.StampEffect(modeState.Epoch, effect.EffectID))
				}
			}

			current, _ = reducer.Reduce(current, makeEvent(nextEventID, ingest.EventDispatchCompleted, map[string]any{
				"order_id":      candidate.OrderID,
				"stage_index":   candidate.StageIndex,
				"attempt_id":    attemptID,
				"session_id":    fmt.Sprintf("session-%s-%d", candidate.OrderID, candidate.StageIndex),
				"worktree_name": fmt.Sprintf("wt-%s-%d", candidate.OrderID, candidate.StageIndex),
			}, now.Add(20*time.Millisecond)))
			nextEventID++

			if step%3 == 0 {
				current, _ = reducer.Reduce(current, makeEvent(nextEventID, ingest.EventStageFailed, map[string]any{
					"order_id":    candidate.OrderID,
					"stage_index": candidate.StageIndex,
					"attempt_id":  attemptID,
					"error":       "stop under pressure",
				}, now.Add(40*time.Millisecond)))
				nextEventID++
				continue
			}

			current, _ = reducer.Reduce(current, makeEvent(nextEventID, ingest.EventStageCompleted, map[string]any{
				"order_id":    candidate.OrderID,
				"stage_index": candidate.StageIndex,
				"attempt_id":  attemptID,
				"mergeable":   gate.CanAutoMerge(modeState.EffectiveMode),
			}, now.Add(40*time.Millisecond)))
			nextEventID++

			if gate.CanAutoMerge(modeState.EffectiveMode) {
				current, _ = reducer.Reduce(current, makeEvent(nextEventID, ingest.EventMergeCompleted, map[string]any{
					"order_id":    candidate.OrderID,
					"stage_index": candidate.StageIndex,
				}, now.Add(60*time.Millisecond)))
				nextEventID++
			}
		}

		if blockedDispatches == 0 {
			t.Fatalf("dispatch blocking not observed during control pressure run")
		}
		if allowedDispatches == 0 {
			t.Fatalf("dispatch allowance not observed during control pressure run")
		}

		for i, transition := range modeState.Transitions {
			wantEpoch := mode.ModeEpoch(i + 1)
			if transition.Epoch != wantEpoch {
				t.Fatalf("mode transition epoch sequence drifted at index=%d epoch=%d", i, transition.Epoch)
			}
		}

		epochValidationMismatches := 0
		for _, stampedEffect := range stamped {
			result := mode.ValidateEpoch(stampedEffect.Epoch, modeState.Epoch)
			if stampedEffect.Epoch == modeState.Epoch && result != mode.EpochValid {
				epochValidationMismatches++
			}
			if stampedEffect.Epoch < modeState.Epoch && result != mode.EpochStale {
				epochValidationMismatches++
			}
		}
		if epochValidationMismatches != 0 {
			t.Fatalf("mode epoch validation drifted under load: mismatches=%d", epochValidationMismatches)
		}
	})

	t.Run("TestProjectionMonotonicity", func(t *testing.T) {
		baseAt := time.Date(2026, 3, 1, 14, 0, 0, 0, time.UTC)
		current := makeState(18, 2, baseAt)
		events := lifecycleEvents(18, 2, 1, baseAt.Add(time.Second))
		if len(events) < 100 {
			t.Fatalf("event stream too small for projection monotonicity run: count=%d", len(events))
		}

		prevVersion := projection.ProjectionVersion(0)
		versionViolations := 0
		hashInconsistencies := 0
		hashByState := make(map[string]projection.ProjectionHash)

		for i, event := range events[:100] {
			current, _ = reducer.Reduce(current, event)
			bundle := projection.Project(current, mode.ModeState{EffectiveMode: current.Mode})

			if i > 0 && bundle.Version < prevVersion {
				versionViolations++
			}
			prevVersion = bundle.Version

			sig := mustStateSignature(current)
			if existing, ok := hashByState[sig]; ok && existing != bundle.Hash {
				hashInconsistencies++
			}
			hashByState[sig] = bundle.Hash
		}

		if versionViolations != 0 {
			t.Fatalf("projection version moved backwards during replay: violations=%d", versionViolations)
		}
		if hashInconsistencies != 0 {
			t.Fatalf("projection hash changed for identical state content: inconsistencies=%d", hashInconsistencies)
		}
	})
}

type applyRun struct {
	state             state.State
	ledger            *reducer.EffectLedger
	dispatchEffectIDs []string
}

func applyEvents(initial state.State, events []ingest.StateEvent, existingLedger *reducer.EffectLedger) applyRun {
	ledger := existingLedger
	if ledger == nil {
		ledger = reducer.NewEffectLedger()
	}

	current := initial
	dispatchIDs := make([]string, 0)
	for _, event := range events {
		next, effects := reducer.Reduce(current, event)
		current = next
		for _, effect := range effects {
			ledger.Record(effect)
			if effect.Type == reducer.EffectDispatch {
				dispatchIDs = append(dispatchIDs, effect.EffectID)
			}
		}
	}

	return applyRun{
		state:             current,
		ledger:            ledger,
		dispatchEffectIDs: dispatchIDs,
	}
}

func makeState(orderCount, stageCount int, at time.Time) state.State {
	orders := make(map[string]state.OrderNode, orderCount)
	for i := 0; i < orderCount; i++ {
		orderID := fmt.Sprintf("order-%03d", i)
		stages := make([]state.StageNode, 0, stageCount)
		for stageIndex := 0; stageIndex < stageCount; stageIndex++ {
			stages = append(stages, state.StageNode{
				StageIndex: stageIndex,
				Status:     state.StagePending,
				Skill:      fmt.Sprintf("skill-%d", stageIndex),
				Runtime:    "process",
				Group:      "main",
			})
		}

		orders[orderID] = state.OrderNode{
			OrderID:   orderID,
			Status:    state.OrderActive,
			Stages:    stages,
			CreatedAt: at,
			UpdatedAt: at,
		}
	}

	return state.State{
		Orders:        orders,
		Mode:          state.RunModeAuto,
		SchemaVersion: statever.Current,
	}
}

func lifecycleEvents(orderCount, stageCount int, startID ingest.EventID, at time.Time) []ingest.StateEvent {
	events := make([]ingest.StateEvent, 0, orderCount*stageCount*4)
	nextID := startID
	nextTimestamp := func() time.Time {
		ts := at.Add(time.Duration(len(events)) * 10 * time.Millisecond)
		return ts
	}

	for orderIndex := 0; orderIndex < orderCount; orderIndex++ {
		orderID := fmt.Sprintf("order-%03d", orderIndex)
		for stageIndex := 0; stageIndex < stageCount; stageIndex++ {
			attemptID := fmt.Sprintf("attempt-%03d-%d", orderIndex, stageIndex)
			events = append(events, makeEvent(nextID, ingest.EventDispatchRequested, map[string]any{
				"order_id":    orderID,
				"stage_index": stageIndex,
				"attempt_id":  attemptID,
			}, nextTimestamp()))
			nextID++

			events = append(events, makeEvent(nextID, ingest.EventDispatchCompleted, map[string]any{
				"order_id":      orderID,
				"stage_index":   stageIndex,
				"attempt_id":    attemptID,
				"session_id":    fmt.Sprintf("session-%03d-%d", orderIndex, stageIndex),
				"worktree_name": fmt.Sprintf("wt-%03d-%d", orderIndex, stageIndex),
			}, nextTimestamp()))
			nextID++

			events = append(events, makeEvent(nextID, ingest.EventStageCompleted, map[string]any{
				"order_id":    orderID,
				"stage_index": stageIndex,
				"attempt_id":  attemptID,
				"mergeable":   true,
			}, nextTimestamp()))
			nextID++

			events = append(events, makeEvent(nextID, ingest.EventMergeCompleted, map[string]any{
				"order_id":    orderID,
				"stage_index": stageIndex,
			}, nextTimestamp()))
			nextID++
		}
	}

	return events
}

func makeEvent(id ingest.EventID, eventType ingest.EventType, payload map[string]any, ts time.Time) ingest.StateEvent {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("event payload encoding failed: %v", err))
	}
	return ingest.StateEvent{
		ID:             id,
		Source:         string(ingest.SourceInternal),
		Type:           string(eventType),
		Timestamp:      ts,
		Payload:        data,
		IdempotencyKey: fmt.Sprintf("idempotency-%d", id),
		Applied:        true,
	}
}

func hashState(s state.State) projection.ProjectionHash {
	bundle := projection.Project(s, mode.ModeState{EffectiveMode: s.Mode})
	return bundle.Hash
}

func terminalCounts(s state.State) (orders int, stages int) {
	for _, order := range s.Orders {
		if isTerminalOrder(order.Status) {
			orders++
		}
		for _, stage := range order.Stages {
			if isTerminalStage(stage.Status) {
				stages++
			}
		}
	}
	return orders, stages
}

func isTerminalOrder(status state.OrderLifecycleStatus) bool {
	switch status {
	case state.OrderCompleted, state.OrderFailed, state.OrderCancelled:
		return true
	default:
		return false
	}
}

func isTerminalStage(status state.StageLifecycleStatus) bool {
	switch status {
	case state.StageCompleted, state.StageFailed, state.StageSkipped, state.StageCancelled:
		return true
	default:
		return false
	}
}

func duplicateCount(ids []string) int {
	seen := make(map[string]struct{}, len(ids))
	duplicates := 0
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			duplicates++
			continue
		}
		seen[id] = struct{}{}
	}
	return duplicates
}

func writeReadSnapshot(t *testing.T, snapshot reducer.DurableSnapshot) reducer.DurableSnapshot {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.snapshot.json")
	if err := reducer.WriteSnapshotAtomic(path, snapshot); err != nil {
		t.Fatalf("snapshot persistence failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("snapshot read failed: %v", err)
	}

	var restored reducer.DurableSnapshot
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("snapshot decode failed: %v", err)
	}
	return restored
}

func ledgerFromRecords(records []reducer.EffectLedgerRecord) *reducer.EffectLedger {
	ledger := reducer.NewEffectLedger()
	for _, rec := range records {
		ledger.Record(rec.Effect)

		switch rec.Status {
		case reducer.EffectLedgerPending:
			continue
		case reducer.EffectLedgerRunning:
			_ = ledger.MarkRunning(rec.EffectID)
		case reducer.EffectLedgerDone:
			_ = ledger.MarkRunning(rec.EffectID)
			result := rec.Result
			if result == nil {
				result = &reducer.EffectResult{EffectID: rec.EffectID, Status: reducer.EffectResultCompleted}
			}
			_ = ledger.MarkDone(rec.EffectID, *result)
		case reducer.EffectLedgerFailed:
			_ = ledger.MarkRunning(rec.EffectID)
			result := rec.Result
			if result == nil {
				result = &reducer.EffectResult{EffectID: rec.EffectID, Status: reducer.EffectResultFailed}
			}
			_ = ledger.MarkFailed(rec.EffectID, *result)
		}
	}
	return ledger
}

func writeInterruptedProjectionTemp(t *testing.T, dir string, bundle projection.ProjectionBundle) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("projection directory creation failed: %v", err)
	}

	temp, err := os.CreateTemp(dir, "orders.json.tmp.*")
	if err != nil {
		t.Fatalf("projection temp file creation failed: %v", err)
	}
	defer temp.Close()

	payload, err := json.Marshal(bundle.OrdersProjection)
	if err != nil {
		t.Fatalf("projection temp payload encoding failed: %v", err)
	}
	if _, err := temp.Write(payload); err != nil {
		t.Fatalf("projection temp payload write failed: %v", err)
	}
	return temp.Name()
}

func mustStateSignature(s state.State) string {
	data, err := json.Marshal(s)
	if err != nil {
		panic(fmt.Sprintf("state signature encoding failed: %v", err))
	}
	return string(data)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
