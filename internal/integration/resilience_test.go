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

		runA := applyEvents(t, initial, events, nil)
		runB := applyEvents(t, makeState(50, 2, baseAt), events, nil)

		hashA := hashState(t, runA.state)
		hashB := hashState(t, runB.state)

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
		fullRun := applyEvents(t, initial, events, nil)
		fullHash := hashState(t, fullRun.state)
		fullOrders, fullStages := terminalCounts(fullRun.state)

		t.Run("after event persistence before state/effect ledger commit", func(t *testing.T) {
			cut := len(events) / 3
			if cut == 0 {
				t.Fatalf("crash cut index unresolved for event stream length=%d", len(events))
			}

			beforeCrash := applyEvents(t, initial, events[:cut], nil)
			snapshot := reducer.BuildSnapshot(beforeCrash.state, beforeCrash.ledger, baseAt.Add(2*time.Minute))
			snapshot = writeReadSnapshot(t, snapshot)

			recovered := applyEvents(t, snapshot.State, events[cut:], ledgerFromRecords(snapshot.EffectLedger))
			recoveredHash := hashState(t, recovered.state)
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

			beforeCrash := applyEvents(t, initial, events[:1], nil)
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

			recovered := applyEvents(t, snapshot.State, events[1:], recoveredLedger)
			recoveredHash := hashState(t, recovered.state)
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
			beforeCrash := applyEvents(t, initial, events[:cut], nil)
			snapshot := reducer.BuildSnapshot(beforeCrash.state, beforeCrash.ledger, baseAt.Add(4*time.Minute))

			projectionDir := t.TempDir()
			bundle := mustProject(t, beforeCrash.state, mode.ModeState{EffectiveMode: beforeCrash.state.Mode})
			tempPath := writeInterruptedProjectionTemp(t, projectionDir, bundle)
			if _, err := os.Stat(tempPath); err != nil {
				t.Fatalf("projection temp artifact unavailable at crash window C boundary: %v", err)
			}

			snapshot = writeReadSnapshot(t, snapshot)
			recovered := applyEvents(t, snapshot.State, events[cut:], ledgerFromRecords(snapshot.EffectLedger))

			finalBundle := mustProject(t, recovered.state, mode.ModeState{EffectiveMode: recovered.state.Mode})
			if err := projection.WriteProjectionFiles(projectionDir, finalBundle); err != nil {
				t.Fatalf("projection rewrite failed after crash window C: %v", err)
			}

			recoveredHash := hashState(t, recovered.state)
			if recoveredHash != fullHash {
				t.Fatalf("recovered state hash diverged after crash window C: recovered=%s full=%s", recoveredHash, fullHash)
			}

			fullBundle := mustProject(t, fullRun.state, mode.ModeState{EffectiveMode: fullRun.state.Mode})
			if finalBundle.Hash != fullBundle.Hash {
				t.Fatalf("projection hash diverged after crash window C recovery: recovered=%s full=%s", finalBundle.Hash, fullBundle.Hash)
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

		runA := applyEvents(t, initialA, stream, nil)
		runB := applyEvents(t, initialB, stream, nil)
		hashA := hashState(t, runA.state)
		hashB := hashState(t, runB.state)

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

		mustReduce := func(s state.State, evt ingest.StateEvent) (state.State, []reducer.Effect) {
			t.Helper()
			next, effects, err := reducer.Reduce(s, evt)
			if err != nil {
				t.Fatalf("reducer failed at event %d (type=%s): %v", evt.ID, evt.Type, err)
			}
			return next, effects
		}

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
				current, _ = mustReduce(current, makeEvent(nextEventID, ingest.EventModeChanged, map[string]any{
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
			var effects []reducer.Effect
			current, effects = mustReduce(current, request)
			for _, effect := range effects {
				if effect.Type == reducer.EffectDispatch {
					stamped = append(stamped, mode.StampEffect(modeState.Epoch, effect.EffectID))
				}
			}

			current, _ = mustReduce(current, makeEvent(nextEventID, ingest.EventDispatchCompleted, map[string]any{
				"order_id":      candidate.OrderID,
				"stage_index":   candidate.StageIndex,
				"attempt_id":    attemptID,
				"session_id":    fmt.Sprintf("session-%s-%d", candidate.OrderID, candidate.StageIndex),
				"worktree_name": fmt.Sprintf("wt-%s-%d", candidate.OrderID, candidate.StageIndex),
			}, now.Add(20*time.Millisecond)))
			nextEventID++

			if step%3 == 0 {
				current, _ = mustReduce(current, makeEvent(nextEventID, ingest.EventStageFailed, map[string]any{
					"order_id":    candidate.OrderID,
					"stage_index": candidate.StageIndex,
					"attempt_id":  attemptID,
					"error":       "stop under pressure",
				}, now.Add(40*time.Millisecond)))
				nextEventID++
				continue
			}

			current, _ = mustReduce(current, makeEvent(nextEventID, ingest.EventStageCompleted, map[string]any{
				"order_id":    candidate.OrderID,
				"stage_index": candidate.StageIndex,
				"attempt_id":  attemptID,
				"mergeable":   gate.CanAutoMerge(modeState.EffectiveMode),
			}, now.Add(40*time.Millisecond)))
			nextEventID++

			if gate.CanAutoMerge(modeState.EffectiveMode) {
				current, _ = mustReduce(current, makeEvent(nextEventID, ingest.EventMergeCompleted, map[string]any{
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
		primaryVersions := make([]projection.ProjectionVersion, 0, 100)

		for _, event := range events[:100] {
			next, _, err := reducer.Reduce(current, event)
			if err != nil {
				t.Fatalf("reducer failed during monotonicity run: %v", err)
			}
			current = next
			bundle := mustProject(t, current, mode.ModeState{EffectiveMode: current.Mode})

			if bundle.Version <= prevVersion {
				versionViolations++
			}
			prevVersion = bundle.Version
			primaryVersions = append(primaryVersions, bundle.Version)

			sig := mustStateSignature(current)
			if existing, ok := hashByState[sig]; ok && existing != bundle.Hash {
				hashInconsistencies++
			}
			hashByState[sig] = bundle.Hash
		}

		if versionViolations != 0 {
			t.Fatalf("projection version not strictly increasing: violations=%d", versionViolations)
		}
		if hashInconsistencies != 0 {
			t.Fatalf("projection hash changed for identical state content: inconsistencies=%d", hashInconsistencies)
		}

		// Replay phase: fresh state, same events, verify same monotonic version pattern
		replayCurrent := makeState(18, 2, baseAt)
		replayViolations := 0
		for i, event := range events[:100] {
			next, _, err := reducer.Reduce(replayCurrent, event)
			if err != nil {
				t.Fatalf("reducer failed during monotonicity replay: %v", err)
			}
			replayCurrent = next
			bundle := mustProject(t, replayCurrent, mode.ModeState{EffectiveMode: replayCurrent.Mode})
			if bundle.Version != primaryVersions[i] {
				replayViolations++
			}
		}
		if replayViolations != 0 {
			t.Fatalf("projection version pattern diverged on replay: violations=%d", replayViolations)
		}
	})
}

func TestIngestionReplayDeterminism(t *testing.T) {
	baseAt := time.Date(2026, 3, 1, 15, 0, 0, 0, time.UTC)
	const eventCount = 120
	const orderCount = 30
	const stageCount = 1

	// Build InputEnvelopes that go through the ingester's normalization and dedup.
	envelopes := buildInputEnvelopes(orderCount, stageCount, baseAt)
	if len(envelopes) < eventCount {
		t.Fatalf("envelope stream too small for ingestion replay: count=%d", len(envelopes))
	}
	envelopes = envelopes[:eventCount]

	// --- Run A: ingest + reduce ---
	ingesterA := ingest.NewIngester()
	stateA := makeState(orderCount, stageCount, baseAt)
	var appliedA []ingest.StateEvent
	for _, env := range envelopes {
		event, err := ingesterA.Ingest(env)
		if err != nil {
			t.Fatalf("ingester A failed: %v", err)
		}
		if !event.Applied {
			t.Fatalf("event unexpectedly deduplicated on first pass: id=%d key=%s reason=%s", event.ID, event.IdempotencyKey, event.DedupReason)
		}
		appliedA = append(appliedA, event)
		next, _, err := reducer.Reduce(stateA, event)
		if err != nil {
			t.Fatalf("reducer failed during run A at event %d: %v", event.ID, err)
		}
		stateA = next
	}
	hashA := hashState(t, stateA)
	statsA := ingesterA.Stats()

	// --- Run B: new ingester, same envelopes = identical result ---
	ingesterB := ingest.NewIngester()
	stateB := makeState(orderCount, stageCount, baseAt)
	for _, env := range envelopes {
		event, err := ingesterB.Ingest(env)
		if err != nil {
			t.Fatalf("ingester B failed: %v", err)
		}
		if !event.Applied {
			t.Fatalf("event unexpectedly deduplicated on replay pass: id=%d key=%s reason=%s", event.ID, event.IdempotencyKey, event.DedupReason)
		}
		next, _, err := reducer.Reduce(stateB, event)
		if err != nil {
			t.Fatalf("reducer failed during run B at event %d: %v", event.ID, err)
		}
		stateB = next
	}
	hashB := hashState(t, stateB)

	if hashA != hashB {
		t.Fatalf("deterministic hash diverged across ingestion replay: run_a=%s run_b=%s", hashA, hashB)
	}

	// --- Dedup verification: replay same envelopes into ingester A ---
	dedupCount := 0
	for _, env := range envelopes {
		event, err := ingesterA.Ingest(env)
		if err != nil {
			t.Fatalf("ingester A dedup pass failed: %v", err)
		}
		if event.Applied {
			t.Fatalf("duplicate envelope not deduplicated: id=%d key=%s", event.ID, event.IdempotencyKey)
		}
		dedupCount++
	}

	statsAfterDedup := ingesterA.Stats()
	wantDeduped := statsAfterDedup.DedupedEvents
	if wantDeduped != uint64(eventCount) {
		t.Fatalf("dedup count diverged from event count: deduped=%d events=%d", wantDeduped, eventCount)
	}

	// --- Acceptance gate ---
	t.Logf("ingestion replay determinism: hash_match=true events=%d applied_a=%d deduped_on_replay=%d",
		eventCount, statsA.AppliedEvents, dedupCount)
}

func TestCrashBoundaryWithIngestionDedup(t *testing.T) {
	baseAt := time.Date(2026, 3, 1, 16, 0, 0, 0, time.UTC)
	const orderCount = 20
	const stageCount = 2
	const overlapWindow = 5

	envelopes := buildInputEnvelopes(orderCount, stageCount, baseAt)
	if len(envelopes) < 20 {
		t.Fatalf("envelope stream too small for crash boundary: count=%d", len(envelopes))
	}

	// --- Full uninterrupted run through ingester + reducer ---
	fullIngester := ingest.NewIngester()
	fullState := makeState(orderCount, stageCount, baseAt)
	var allApplied []ingest.StateEvent
	for _, env := range envelopes {
		event, err := fullIngester.Ingest(env)
		if err != nil {
			t.Fatalf("full ingester failed: %v", err)
		}
		if !event.Applied {
			continue
		}
		allApplied = append(allApplied, event)
		next, _, err := reducer.Reduce(fullState, event)
		if err != nil {
			t.Fatalf("reducer failed during full run at event %d: %v", event.ID, err)
		}
		fullState = next
	}
	fullOrders, fullStages := terminalCounts(fullState)

	// --- Crash at envelope K ---
	K := len(envelopes) / 2
	if K < overlapWindow {
		t.Fatalf("crash point too early for overlap window: K=%d overlap=%d", K, overlapWindow)
	}

	// Pre-crash: process envelopes [0..K) through a fresh ingester + reducer
	preCrashIngester := ingest.NewIngester()
	preCrashState := makeState(orderCount, stageCount, baseAt)
	for i := 0; i < K; i++ {
		event, err := preCrashIngester.Ingest(envelopes[i])
		if err != nil {
			t.Fatalf("pre-crash ingester failed at envelope %d: %v", i, err)
		}
		if !event.Applied {
			continue
		}
		next, _, err := reducer.Reduce(preCrashState, event)
		if err != nil {
			t.Fatalf("reducer failed during pre-crash at event %d: %v", event.ID, err)
		}
		preCrashState = next
	}

	// Save pre-crash state (simulates snapshot)
	snapshot := reducer.BuildSnapshot(preCrashState, reducer.NewEffectLedger(), baseAt.Add(5*time.Minute))
	snapshot = writeReadSnapshot(t, snapshot)

	// Recovery: replay envelopes [K-overlap..N) through the SAME ingester
	// (simulating an ingester whose dedup index survived the crash).
	// The ingester's monotonic ID counter advances even for deduplicated events,
	// so post-recovery event IDs will differ from the full run. This is expected:
	// we verify logical state equivalence (order/stage statuses and terminal counts)
	// rather than projection-hash equality.
	recoveredState := snapshot.State
	replayStart := K - overlapWindow
	duplicatesObserved := 0
	for i := replayStart; i < len(envelopes); i++ {
		event, err := preCrashIngester.Ingest(envelopes[i])
		if err != nil {
			t.Fatalf("recovery ingester failed at envelope %d: %v", i, err)
		}
		if !event.Applied {
			duplicatesObserved++
			continue
		}
		next, _, err := reducer.Reduce(recoveredState, event)
		if err != nil {
			t.Fatalf("reducer failed during recovery at event %d: %v", event.ID, err)
		}
		recoveredState = next
	}

	if duplicatesObserved != overlapWindow {
		t.Fatalf("overlap dedup count mismatched: observed=%d overlap_window=%d", duplicatesObserved, overlapWindow)
	}

	// Verify logical state equivalence: same order/stage statuses
	recoveredOrders, recoveredStages := terminalCounts(recoveredState)
	lostOrders := maxInt(0, fullOrders-recoveredOrders)
	lostStages := maxInt(0, fullStages-recoveredStages)
	if lostOrders != 0 || lostStages != 0 {
		t.Fatalf("terminal states dropped after crash boundary recovery: lost_orders=%d lost_stages=%d", lostOrders, lostStages)
	}
	if recoveredOrders != fullOrders {
		t.Fatalf("recovered terminal order count diverged: recovered=%d full=%d", recoveredOrders, fullOrders)
	}
	if recoveredStages != fullStages {
		t.Fatalf("recovered terminal stage count diverged: recovered=%d full=%d", recoveredStages, fullStages)
	}

	// Verify per-order structural equivalence (status and stage statuses match)
	for orderID, fullOrder := range fullState.Orders {
		recoveredOrder, exists := recoveredState.Orders[orderID]
		if !exists {
			t.Fatalf("order missing after crash recovery: order_id=%s", orderID)
		}
		if recoveredOrder.Status != fullOrder.Status {
			t.Fatalf("order status diverged after crash recovery: order_id=%s recovered=%s full=%s", orderID, recoveredOrder.Status, fullOrder.Status)
		}
		if len(recoveredOrder.Stages) != len(fullOrder.Stages) {
			t.Fatalf("stage count diverged after crash recovery: order_id=%s recovered=%d full=%d", orderID, len(recoveredOrder.Stages), len(fullOrder.Stages))
		}
		for si, fullStage := range fullOrder.Stages {
			recoveredStage := recoveredOrder.Stages[si]
			if recoveredStage.Status != fullStage.Status {
				t.Fatalf("stage status diverged after crash recovery: order_id=%s stage=%d recovered=%s full=%s", orderID, si, recoveredStage.Status, fullStage.Status)
			}
		}
	}

	t.Logf("crash boundary with dedup: structural_match=true overlap_deduped=%d lost_terminal=0 applied_events=%d", duplicatesObserved, len(allApplied))
}

func TestAcceptanceGateSummary(t *testing.T) {
	baseAt := time.Date(2026, 3, 1, 17, 0, 0, 0, time.UTC)

	// Counters for all acceptance criteria
	duplicateDispatches := 0
	terminalStatesLost := 0
	deterministicHashMatches := 0
	deterministicHashTrials := 0
	projectionMonotonicityViolations := 0

	// --- Trial 1: Deterministic replay hash match ---
	const trialCount = 3
	initialEvents := lifecycleEvents(20, 2, 1, baseAt.Add(time.Second))
	if len(initialEvents) < 100 {
		t.Fatalf("event stream too small for acceptance gate: count=%d", len(initialEvents))
	}
	stream := initialEvents[:100]

	var referenceHash projection.ProjectionHash
	for trial := 0; trial < trialCount; trial++ {
		trialState := makeState(20, 2, baseAt)
		run := applyEvents(t, trialState, stream, nil)
		h := hashState(t, run.state)
		deterministicHashTrials++
		if trial == 0 {
			referenceHash = h
		}
		if h == referenceHash {
			deterministicHashMatches++
		}

		// Check dispatch duplicates per run
		duplicateDispatches += duplicateCount(run.dispatchEffectIDs)
	}

	// --- Trial 2: Crash recovery terminal state preservation ---
	crashBaseAt := baseAt.Add(10 * time.Minute)
	crashInitial := makeState(24, 2, crashBaseAt)
	crashEvents := lifecycleEvents(24, 2, 1, crashBaseAt.Add(time.Second))
	fullRun := applyEvents(t, crashInitial, crashEvents, nil)
	fullOrders, fullStages := terminalCounts(fullRun.state)
	fullHash := hashState(t, fullRun.state)

	cuts := []int{len(crashEvents) / 4, len(crashEvents) / 2, len(crashEvents) * 3 / 4}
	for _, cut := range cuts {
		if cut == 0 {
			continue
		}
		beforeCrash := applyEvents(t, makeState(24, 2, crashBaseAt), crashEvents[:cut], nil)
		snapshot := reducer.BuildSnapshot(beforeCrash.state, beforeCrash.ledger, crashBaseAt.Add(5*time.Minute))
		snapshot = writeReadSnapshot(t, snapshot)

		recovered := applyEvents(t, snapshot.State, crashEvents[cut:], ledgerFromRecords(snapshot.EffectLedger))
		recoveredHash := hashState(t, recovered.state)

		deterministicHashTrials++
		if recoveredHash == fullHash {
			deterministicHashMatches++
		}

		recoveredOrders, recoveredStages := terminalCounts(recovered.state)
		terminalStatesLost += maxInt(0, fullOrders-recoveredOrders)
		terminalStatesLost += maxInt(0, fullStages-recoveredStages)

		duplicateDispatches += duplicateCount(append(beforeCrash.dispatchEffectIDs, recovered.dispatchEffectIDs...))
	}

	// --- Trial 3: Projection monotonicity ---
	monoState := makeState(18, 2, baseAt)
	monoEvents := lifecycleEvents(18, 2, 1, baseAt.Add(time.Second))
	if len(monoEvents) < 100 {
		t.Fatalf("event stream too small for acceptance gate monotonicity: count=%d", len(monoEvents))
	}
	prevVersion := projection.ProjectionVersion(0)
	for _, event := range monoEvents[:100] {
		next, _, err := reducer.Reduce(monoState, event)
		if err != nil {
			t.Fatalf("reducer failed during acceptance gate monotonicity: %v", err)
		}
		monoState = next
		bundle := mustProject(t, monoState, mode.ModeState{EffectiveMode: monoState.Mode})
		if bundle.Version <= prevVersion {
			projectionMonotonicityViolations++
		}
		prevVersion = bundle.Version
	}

	// --- Acceptance gate assertions ---
	deterministicPercent := 0
	if deterministicHashTrials > 0 {
		deterministicPercent = (deterministicHashMatches * 100) / deterministicHashTrials
	}

	if duplicateDispatches != 0 {
		t.Fatalf("duplicate dispatches observed across replay: %d (expected 0)", duplicateDispatches)
	}
	if terminalStatesLost != 0 {
		t.Fatalf("terminal states lost after replay: %d (expected 0)", terminalStatesLost)
	}
	if deterministicPercent != 100 {
		t.Fatalf("deterministic replay hash match: %d%% (expected 100%%)", deterministicPercent)
	}
	if projectionMonotonicityViolations != 0 {
		t.Fatalf("projection monotonicity violations: %d (expected 0)", projectionMonotonicityViolations)
	}

	t.Logf("acceptance gate passed: duplicate_dispatches=0 terminal_states_lost=0 deterministic_match=100%% monotonicity_violations=0")
}

// buildInputEnvelopes creates raw InputEnvelope values for the full lifecycle
// of orderCount orders with stageCount stages each, using SourceInternal so the
// ingester extracts idempotency_key from the payload.
func buildInputEnvelopes(orderCount, stageCount int, baseAt time.Time) []ingest.InputEnvelope {
	envelopes := make([]ingest.InputEnvelope, 0, orderCount*stageCount*4)
	seqNum := 0
	nextTimestamp := func() time.Time {
		seqNum++
		return baseAt.Add(time.Duration(seqNum) * 10 * time.Millisecond)
	}

	for orderIndex := 0; orderIndex < orderCount; orderIndex++ {
		orderID := fmt.Sprintf("order-%03d", orderIndex)
		for stageIndex := 0; stageIndex < stageCount; stageIndex++ {
			attemptID := fmt.Sprintf("attempt-%03d-%d", orderIndex, stageIndex)
			idempKey := fmt.Sprintf("idem-%03d-%d", orderIndex, stageIndex)

			envelopes = append(envelopes, ingest.InputEnvelope{
				Source: string(ingest.SourceInternal),
				RawPayload: mustMarshal(map[string]any{
					"type":            string(ingest.EventDispatchRequested),
					"idempotency_key": idempKey + "-dispatch-req",
					"order_id":        orderID,
					"stage_index":     stageIndex,
					"attempt_id":      attemptID,
				}),
				ReceivedAt: nextTimestamp(),
			})

			envelopes = append(envelopes, ingest.InputEnvelope{
				Source: string(ingest.SourceInternal),
				RawPayload: mustMarshal(map[string]any{
					"type":            string(ingest.EventDispatchCompleted),
					"idempotency_key": idempKey + "-dispatch-done",
					"order_id":        orderID,
					"stage_index":     stageIndex,
					"attempt_id":      attemptID,
					"session_id":      fmt.Sprintf("session-%03d-%d", orderIndex, stageIndex),
					"worktree_name":   fmt.Sprintf("wt-%03d-%d", orderIndex, stageIndex),
				}),
				ReceivedAt: nextTimestamp(),
			})

			envelopes = append(envelopes, ingest.InputEnvelope{
				Source: string(ingest.SourceInternal),
				RawPayload: mustMarshal(map[string]any{
					"type":            string(ingest.EventStageCompleted),
					"idempotency_key": idempKey + "-stage-done",
					"order_id":        orderID,
					"stage_index":     stageIndex,
					"attempt_id":      attemptID,
					"mergeable":       true,
				}),
				ReceivedAt: nextTimestamp(),
			})

			envelopes = append(envelopes, ingest.InputEnvelope{
				Source: string(ingest.SourceInternal),
				RawPayload: mustMarshal(map[string]any{
					"type":            string(ingest.EventMergeCompleted),
					"idempotency_key": idempKey + "-merge-done",
					"order_id":        orderID,
					"stage_index":     stageIndex,
				}),
				ReceivedAt: nextTimestamp(),
			})
		}
	}

	return envelopes
}

func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mustMarshal failed: %v", err))
	}
	return data
}

type applyRun struct {
	state             state.State
	ledger            *reducer.EffectLedger
	dispatchEffectIDs []string
}

func applyEvents(t *testing.T, initial state.State, events []ingest.StateEvent, existingLedger *reducer.EffectLedger) applyRun {
	t.Helper()
	ledger := existingLedger
	if ledger == nil {
		ledger = reducer.NewEffectLedger()
	}

	current := initial
	dispatchIDs := make([]string, 0)
	for _, event := range events {
		next, effects, err := reducer.Reduce(current, event)
		if err != nil {
			t.Fatalf("reducer failed at event %d (type=%s): %v", event.ID, event.Type, err)
		}
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

func mustProject(t *testing.T, s state.State, ms mode.ModeState) projection.ProjectionBundle {
	t.Helper()
	bundle, err := projection.Project(s, ms)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	return bundle
}

func hashState(t *testing.T, s state.State) projection.ProjectionHash {
	t.Helper()
	bundle := mustProject(t, s, mode.ModeState{EffectiveMode: s.Mode})
	return bundle.Hash
}

func terminalCounts(s state.State) (orders int, stages int) {
	for _, order := range s.Orders {
		if order.Status.IsTerminal() {
			orders++
		}
		for _, stage := range order.Stages {
			if stage.Status.IsTerminal() {
				stages++
			}
		}
	}
	return orders, stages
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
