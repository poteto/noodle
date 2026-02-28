// Package ingest normalizes external inputs into canonical state events.
//
// It provides a single ingestion arbiter with deterministic event IDs and
// per-source idempotency tracking before reducers mutate canonical state.
package ingest
