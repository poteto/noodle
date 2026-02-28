package reducer

import (
	"reflect"
	"testing"
	"time"
)

func TestEffectLedgerLifecycleTransitions(t *testing.T) {
	ledger := NewEffectLedger()
	now := time.Date(2026, 2, 28, 19, 0, 0, 0, time.UTC)

	effectA := Effect{
		EffectID:  "event-1-effect-0",
		Type:      EffectDispatch,
		Payload:   []byte(`{"order_id":"o1"}`),
		CreatedAt: now,
	}
	effectB := Effect{
		EffectID:  "event-2-effect-0",
		Type:      EffectCleanup,
		Payload:   []byte(`{"order_id":"o1"}`),
		CreatedAt: now,
	}

	ledger.Record(effectA)
	ledger.Record(effectB)

	if got := len(ledger.Pending()); got != 2 {
		t.Fatalf("pending count mismatch after record: got %d", got)
	}

	if err := ledger.MarkRunning(effectA.EffectID); err != nil {
		t.Fatalf("mark running effect A: %v", err)
	}
	inFlight := ledger.InFlight()
	if len(inFlight) != 1 || inFlight[0].EffectID != effectA.EffectID {
		t.Fatalf("in-flight mismatch: %+v", inFlight)
	}
	if inFlight[0].Attempts != 1 {
		t.Fatalf("attempt count mismatch after mark running: %+v", inFlight[0])
	}

	doneResult := EffectResult{
		EffectID:  effectA.EffectID,
		Status:    EffectResultCompleted,
		Timestamp: now.Add(time.Minute),
	}
	if err := ledger.MarkDone(effectA.EffectID, doneResult); err != nil {
		t.Fatalf("mark done effect A: %v", err)
	}

	if err := ledger.MarkRunning(effectB.EffectID); err != nil {
		t.Fatalf("mark running effect B: %v", err)
	}
	failedResult := EffectResult{
		EffectID:  effectB.EffectID,
		Status:    EffectResultFailed,
		Error:     "merge conflict",
		Timestamp: now.Add(2 * time.Minute),
	}
	if err := ledger.MarkFailed(effectB.EffectID, failedResult); err != nil {
		t.Fatalf("mark failed effect B: %v", err)
	}

	all := ledger.All()
	if len(all) != 2 {
		t.Fatalf("all record count mismatch: got %d", len(all))
	}
	if all[0].Status != EffectLedgerDone {
		t.Fatalf("effect A status mismatch: %+v", all[0])
	}
	if all[0].Result == nil || all[0].Result.Status != EffectResultCompleted {
		t.Fatalf("effect A result mismatch: %+v", all[0])
	}
	if all[1].Status != EffectLedgerFailed {
		t.Fatalf("effect B status mismatch: %+v", all[1])
	}
	if all[1].Result == nil || all[1].Result.Status != EffectResultFailed {
		t.Fatalf("effect B result mismatch: %+v", all[1])
	}
	if got := len(ledger.Pending()); got != 0 {
		t.Fatalf("pending count should be zero after terminal transitions, got %d", got)
	}
}

func TestEffectLedgerRejectsInvalidTransitions(t *testing.T) {
	ledger := NewEffectLedger()
	now := time.Date(2026, 2, 28, 19, 30, 0, 0, time.UTC)
	effect := Effect{
		EffectID:  "event-9-effect-0",
		Type:      EffectAck,
		Payload:   []byte(`{"order_id":"o1"}`),
		CreatedAt: now,
	}

	if err := ledger.MarkRunning("does-not-exist"); err == nil {
		t.Fatal("missing effect should fail mark running")
	}

	ledger.Record(effect)
	if err := ledger.MarkDone(effect.EffectID, EffectResult{EffectID: effect.EffectID, Status: EffectResultCompleted, Timestamp: now}); err == nil {
		t.Fatal("pending effect should fail mark done")
	}
	if err := ledger.MarkFailed(effect.EffectID, EffectResult{EffectID: effect.EffectID, Status: EffectResultFailed, Timestamp: now}); err == nil {
		t.Fatal("pending effect should fail mark failed")
	}

	if err := ledger.MarkRunning(effect.EffectID); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if err := ledger.MarkDone(effect.EffectID, EffectResult{EffectID: effect.EffectID, Status: EffectResultCompleted, Timestamp: now}); err != nil {
		t.Fatalf("mark done: %v", err)
	}
	if err := ledger.MarkRunning(effect.EffectID); err == nil {
		t.Fatal("done effect should fail mark running")
	}
}

func TestEffectLedgerRecordIsIdempotentForDuplicateEffectID(t *testing.T) {
	ledger := NewEffectLedger()
	effect := Effect{EffectID: "event-1-effect-0", Type: EffectAck}

	ledger.Record(effect)
	ledger.Record(effect)

	all := ledger.All()
	if len(all) != 1 {
		t.Fatalf("duplicate record should not duplicate entries, got %d", len(all))
	}
}

func TestEffectLedgerReturnsCopies(t *testing.T) {
	ledger := NewEffectLedger()
	effect := Effect{EffectID: "event-1-effect-0", Type: EffectAck, Payload: []byte(`{"x":1}`)}
	ledger.Record(effect)

	all := ledger.All()
	if len(all) != 1 {
		t.Fatalf("record count mismatch: %d", len(all))
	}
	all[0].Status = EffectLedgerDone
	all[0].Effect.Payload[0] = '{'

	again := ledger.All()
	if again[0].Status != EffectLedgerPending {
		t.Fatalf("ledger state was mutated through returned copy: %+v", again[0])
	}
	if reflect.DeepEqual(all[0], again[0]) {
		t.Fatalf("copy expectation failed, records unexpectedly equal: %+v", again[0])
	}
}
