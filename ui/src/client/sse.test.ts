import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { connectSSE, SNAPSHOT_KEY, SSE_STATUS_KEY } from "./sse";

// Minimal EventSource mock.
type ESListener = (event: { data?: string }) => void;

class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  listeners: Record<string, ESListener[]> = {};
  closed = false;

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: ESListener) {
    (this.listeners[type] ??= []).push(listener);
  }

  close() {
    this.closed = true;
  }

  // Test helpers to simulate events.
  emit(type: string, data?: string) {
    for (const fn of this.listeners[type] ?? []) {
      fn({ data });
    }
  }
}

// Minimal queryClient mock.
function mockQueryClient() {
  const store = new Map<string, unknown>();
  return {
    setQueryData: vi.fn((key: readonly string[], value: unknown) => {
      store.set(JSON.stringify(key), value);
    }),
    getQueryData: (key: readonly string[]) => store.get(JSON.stringify(key)),
    _store: store,
  };
}

beforeEach(() => {
  MockEventSource.instances = [];
  vi.stubGlobal("EventSource", MockEventSource);
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("connectSSE", () => {
  it("connects to /api/events and sets connecting status", () => {
    const qc = mockQueryClient();
    connectSSE(qc as never);

    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0]!.url).toBe("/api/events");
    expect(qc.setQueryData).toHaveBeenCalledWith(SSE_STATUS_KEY, "connecting");
  });

  it("sets connected on open", () => {
    const qc = mockQueryClient();
    connectSSE(qc as never);

    const es = MockEventSource.instances[0]!;
    es.emit("open");

    expect(qc.setQueryData).toHaveBeenCalledWith(SSE_STATUS_KEY, "connected");
  });

  it("parses snapshot from message and sets query data", () => {
    const qc = mockQueryClient();
    connectSSE(qc as never);

    const es = MockEventSource.instances[0]!;
    const snapshot = { loop_state: "running", sessions: [] };
    es.emit("message", JSON.stringify(snapshot));

    expect(qc.setQueryData).toHaveBeenCalledWith(SNAPSHOT_KEY, snapshot);
    expect(qc.setQueryData).toHaveBeenCalledWith(SSE_STATUS_KEY, "connected");
  });

  it("ignores malformed messages", () => {
    const qc = mockQueryClient();
    connectSSE(qc as never);

    const es = MockEventSource.instances[0]!;
    // Should not throw
    es.emit("message", "not json{{{");

    // setQueryData should NOT have been called with SNAPSHOT_KEY
    const snapshotCalls = qc.setQueryData.mock.calls.filter(
      (call: unknown[]) => JSON.stringify(call[0]) === JSON.stringify(SNAPSHOT_KEY),
    );
    expect(snapshotCalls).toHaveLength(0);
  });

  it("reconnects after error with delay", () => {
    const qc = mockQueryClient();
    connectSSE(qc as never);

    const es = MockEventSource.instances[0]!;
    es.emit("error");

    expect(es.closed).toBe(true);
    expect(qc.setQueryData).toHaveBeenCalledWith(SSE_STATUS_KEY, "disconnected");

    // No reconnect yet
    expect(MockEventSource.instances).toHaveLength(1);

    // Advance past reconnect delay
    vi.advanceTimersByTime(2000);

    expect(MockEventSource.instances).toHaveLength(2);
  });

  it("does not reconnect after cleanup", () => {
    const qc = mockQueryClient();
    const cleanup = connectSSE(qc as never);

    const es = MockEventSource.instances[0]!;
    cleanup();

    expect(es.closed).toBe(true);

    // Trigger error after cleanup — should not reconnect
    es.emit("error");
    vi.advanceTimersByTime(5000);

    expect(MockEventSource.instances).toHaveLength(1);
  });
});
