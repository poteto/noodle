// Package reducer provides deterministic state transitions and effect planning.
//
// Concurrency model:
//   - A single reducer goroutine owns canonical state mutation.
//   - Effect workers may execute concurrently, but never mutate canonical state.
//   - Effect workers emit effect-result events back to ingestion.
//   - Reducer commits effect lifecycle transitions from those events.
//
// Crash-consistency protocol:
//  1. Persist ingested event with sequence.
//  2. Reduce event to next state and effect set.
//  3. Persist canonical state snapshot with embedded effect ledger atomically
//     (write temp file, then rename).
//  4. Execute effects using deterministic effect_id idempotency keys.
//     EffectDispatch is a Phase 3 stub and depends on Phase 4 two-phase launch.
//  5. Persist effect result.
//  6. Emit ack/projection updates only after durable commit.
package reducer
