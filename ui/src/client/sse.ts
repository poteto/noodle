import type { QueryClient } from "@tanstack/react-query";
import type { Snapshot } from "./types";

const SNAPSHOT_KEY = ["snapshot"] as const;
const RECONNECT_DELAY_MS = 2000;

export function connectSSE(queryClient: QueryClient): () => void {
  let eventSource: EventSource | null = null;
  let closed = false;

  function connect() {
    if (closed) return;

    eventSource = new EventSource("/api/events");

    eventSource.onmessage = (event) => {
      try {
        const snapshot: Snapshot = JSON.parse(event.data);
        queryClient.setQueryData(SNAPSHOT_KEY, snapshot);
      } catch {
        // Ignore malformed messages.
      }
    };

    eventSource.onerror = () => {
      eventSource?.close();
      eventSource = null;
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

export { SNAPSHOT_KEY };
