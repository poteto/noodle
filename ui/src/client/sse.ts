import type { QueryClient } from "@tanstack/react-query";
import type { Snapshot } from "./types";

const SNAPSHOT_KEY = ["snapshot"] as const;
const SSE_STATUS_KEY = ["sseStatus"] as const;
const RECONNECT_DELAY_MS = 2000;

export type SSEStatus = "connected" | "connecting" | "disconnected";

export function connectSSE(queryClient: QueryClient): () => void {
  let eventSource: EventSource | null = null;
  let closed = false;

  function setStatus(status: SSEStatus) {
    queryClient.setQueryData(SSE_STATUS_KEY, status);
  }

  function connect() {
    if (closed) return;
    setStatus("connecting");

    eventSource = new EventSource("/api/events");

    eventSource.onopen = () => {
      setStatus("connected");
    };

    eventSource.onmessage = (event) => {
      try {
        // Same-origin Go API server — earned boundary cast.
        const snapshot = JSON.parse(event.data) as Snapshot;
        queryClient.setQueryData(SNAPSHOT_KEY, snapshot);
        setStatus("connected");
      } catch {
        // Ignore malformed messages.
      }
    };

    eventSource.onerror = () => {
      eventSource?.close();
      eventSource = null;
      setStatus("disconnected");
      if (!closed) {
        setTimeout(connect, RECONNECT_DELAY_MS);
      }
    };
  }

  connect();

  return () => {
    closed = true;
    eventSource?.close();
    eventSource = null;
  };
}

export { SNAPSHOT_KEY, SSE_STATUS_KEY };
