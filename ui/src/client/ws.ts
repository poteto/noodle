import type { QueryClient } from "@tanstack/react-query";
import type { Snapshot, EventLine, ControlCommand, ControlAck } from "./types";
import { normalizeSnapshot } from "./api";
import { chooseNewerSnapshot } from "./snapshot-freshness";

const SNAPSHOT_KEY = ["snapshot"] as const;
const RECONNECT_DELAY_MS = 2000;

export type WSStatus = "connected" | "connecting" | "disconnected";

// Server -> Client message types
type WSServerMessage =
  | { type: "snapshot"; data: Snapshot }
  | { type: "backfill"; session_id: string; data: EventLine[] }
  | { type: "session_event"; session_id: string; data: EventLine }
  | { type: "session_delta"; session_id: string; text: string; at: string }
  | { type: "subscribed"; session_id: string }
  | { type: "unsubscribed"; session_id: string }
  | { type: "control_ack"; data: ControlAck }
  | { type: "error"; message: string };

// Client -> Server message types
type WSClientMessage =
  | { type: "subscribe"; session_id: string }
  | { type: "unsubscribe"; session_id: string }
  | { type: "control"; data: ControlCommand };

// Module-level state
let ws: WebSocket | null = null;
let queryClientRef: QueryClient | null = null;
let closed = false;

// WS status — module-level store for useSyncExternalStore (avoids useQuery
// returning a new object reference on every snapshot push).
let currentWSStatus: WSStatus = "connecting";
const wsStatusListeners = new Set<() => void>();

export function subscribeWSStatus(cb: () => void): () => void {
  wsStatusListeners.add(cb);
  return () => wsStatusListeners.delete(cb);
}

export function getWSStatus(): WSStatus {
  return currentWSStatus;
}

// Single ref-count map for both counting and reconnection tracking
const sessionRefCounts = new Map<string, number>();

// Pending control command promises
const pendingControls = new Map<
  string,
  { resolve: (ack: ControlAck) => void; reject: (err: Error) => void }
>();

// Delta event bus — per-session text delta listeners (not React Query — too
// high frequency for per-token re-renders).
const deltaListeners = new Map<string, Set<(text: string) => void>>();

export function subscribeDelta(sessionId: string, listener: (text: string) => void): () => void {
  let set = deltaListeners.get(sessionId);
  if (!set) {
    set = new Set();
    deltaListeners.set(sessionId, set);
  }
  set.add(listener);
  return () => {
    set?.delete(listener);
    if (set?.size === 0) {
      deltaListeners.delete(sessionId);
    }
  };
}

function setStatus(status: WSStatus) {
  if (currentWSStatus === status) {
    return;
  }
  currentWSStatus = status;
  for (const fn of wsStatusListeners) {
    fn();
  }
}

function sendJSON(msg: WSClientMessage) {
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(msg));
  }
}

function handleMessage(event: MessageEvent) {
  let msg: WSServerMessage;
  try {
    msg = JSON.parse(event.data as string) as WSServerMessage;
  } catch {
    return;
  }

  switch (msg.type) {
    case "snapshot": {
      // Prevent an older HTTP refetch result from overwriting this newer WS
      // snapshot (race between control-triggered invalidate and WS push).
      void queryClientRef?.cancelQueries({ queryKey: SNAPSHOT_KEY });
      const incoming = normalizeSnapshot(msg.data);
      queryClientRef?.setQueryData<Snapshot>(SNAPSHOT_KEY, (current) =>
        chooseNewerSnapshot(current, incoming),
      );
      break;
    }
    case "backfill": {
      // Cancel any in-flight HTTP fetch to prevent it from overwriting
      // the backfill with stale data (race between HTTP seed and WS backfill).
      void queryClientRef?.cancelQueries({ queryKey: ["sessionEvents", msg.session_id] });
      queryClientRef?.setQueryData(["sessionEvents", msg.session_id], msg.data);
      break;
    }
    case "session_event": {
      // Append single live event, dedup by full event identity.
      // Do not dedup by timestamp alone: steer/user events can arrive
      // slightly out of order relative to tool/cost events and should
      // still render immediately without requiring refresh.
      queryClientRef?.setQueryData<EventLine[]>(["sessionEvents", msg.session_id], (old = []) => {
        const exists = old.some(
          (ev) =>
            ev.at === msg.data.at &&
            ev.label === msg.data.label &&
            ev.body === msg.data.body &&
            ev.category === msg.data.category,
        );
        if (exists) {
          return old;
        }
        const next = [...old, msg.data];
        next.sort((a, b) => a.at.localeCompare(b.at));
        return next;
      });
      break;
    }
    case "session_delta": {
      const listeners = deltaListeners.get(msg.session_id);
      if (listeners) {
        for (const fn of listeners) {
          fn(msg.text);
        }
      }
      break;
    }
    case "subscribed":
    case "unsubscribed": {
      // Confirmations -- no-op
      break;
    }
    case "control_ack": {
      const pending = pendingControls.get(msg.data.id);
      if (pending) {
        pending.resolve(msg.data);
        pendingControls.delete(msg.data.id);
      }
      break;
    }
    case "error": {
      console.warn("[ws] server error:", msg.message);
      break;
    }
    default: {
      break;
    }
  }
}

function connect() {
  if (closed) {
    return;
  }
  setStatus("connecting");

  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  ws = new WebSocket(`${protocol}//${location.host}/api/ws`);

  ws.addEventListener("open", () => {
    setStatus("connected");
    // Re-subscribe all sessions with refcount > 0
    for (const [sessionId, count] of sessionRefCounts) {
      if (count > 0) {
        sendJSON({ type: "subscribe", session_id: sessionId });
      }
    }
  });

  ws.addEventListener("message", handleMessage);

  ws.addEventListener("close", () => {
    ws = null;
    setStatus("disconnected");
    if (!closed) {
      setTimeout(connect, RECONNECT_DELAY_MS);
    }
  });

  ws.addEventListener("error", () => {
    ws?.close();
  });
}

/**
 * Connect WebSocket -- called once at app root.
 * Returns cleanup function.
 */
export function connectWS(queryClient: QueryClient): () => void {
  queryClientRef = queryClient;
  closed = false;
  connect();

  return () => {
    closed = true;
    ws?.close();
    ws = null;
  };
}

/**
 * Reference-counted session subscription.
 * Multiple components can subscribe to the same session.
 */
export function subscribeSession(sessionId: string) {
  const current = sessionRefCounts.get(sessionId) ?? 0;
  sessionRefCounts.set(sessionId, current + 1);
  if (current === 0) {
    sendJSON({ type: "subscribe", session_id: sessionId });
  }
}

/**
 * Decrement ref count. Only unsubscribes when count hits 0.
 */
export function unsubscribeSession(sessionId: string) {
  const current = sessionRefCounts.get(sessionId) ?? 0;
  if (current <= 1) {
    sessionRefCounts.delete(sessionId);
    sendJSON({ type: "unsubscribe", session_id: sessionId });
  } else {
    sessionRefCounts.set(sessionId, current - 1);
  }
}

/**
 * Send a control command over WebSocket with ack correlation.
 */
export function sendWSControl(cmd: ControlCommand): Promise<ControlAck> {
  const id = cmd.id || `ws-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const cmdWithId = { ...cmd, id };

  return new Promise<ControlAck>((resolve, reject) => {
    pendingControls.set(id, { resolve, reject });
    sendJSON({ type: "control", data: cmdWithId });

    // Timeout after 10s
    setTimeout(() => {
      if (pendingControls.has(id)) {
        pendingControls.delete(id);
        reject(new Error("control command timed out"));
      }
    }, 10_000);
  });
}

export { SNAPSHOT_KEY };
