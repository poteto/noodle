import type {
  Snapshot,
  EventLine,
  ControlCommand,
  ControlAck,
  ConfigDefaults,
  DiffResponse,
} from "./types";

// Typed wrapper around res.json(). The Go API server is same-origin and
// its JSON shapes are defined alongside the Go structs, so the cast is
// an earned boundary assertion rather than a blind trust of unknown data.
async function jsonBody<T>(res: Response): Promise<T> {
  return (await res.json()) as T;
}

// Go nil slices marshal to null. Normalize at the boundary so downstream
// code never sees null where it expects [].
export function normalizeSnapshot(raw: Snapshot): Snapshot {
  return {
    ...raw,
    sessions: raw.sessions ?? [],
    active: raw.active ?? [],
    recent: raw.recent ?? [],
    orders: (raw.orders ?? []).map(normalizeOrder),
    active_order_ids: raw.active_order_ids ?? [],
    action_needed: raw.action_needed ?? [],
    events_by_session: raw.events_by_session ?? {},
    feed_events: raw.feed_events ?? [],
    pending_reviews: raw.pending_reviews ?? [],
    warnings: raw.warnings ?? [],
  };
}

function normalizeOrder(order: Snapshot["orders"][number]): Snapshot["orders"][number] {
  return {
    ...order,
    stages: order.stages ?? [],
    on_failure: order.on_failure ?? [],
    plan: order.plan ?? [],
  };
}

export async function fetchSnapshot(): Promise<Snapshot> {
  const res = await fetch("/api/snapshot");
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`fetchSnapshot: ${res.status}${body ? ` — ${body.trim()}` : ""}`);
  }
  return normalizeSnapshot(await jsonBody<Snapshot>(res));
}

export async function fetchSessionEvents(sessionId: string, after?: string): Promise<EventLine[]> {
  const params = new URLSearchParams();
  if (after) {
    params.set("after", after);
  }
  const qs = params.toString();
  const url = `/api/sessions/${encodeURIComponent(sessionId)}/events${qs ? `?${qs}` : ""}`;
  const res = await fetch(url);
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`fetchSessionEvents: ${res.status}${body ? ` — ${body.trim()}` : ""}`);
  }
  return jsonBody<EventLine[]>(res);
}

export async function sendControl(cmd: ControlCommand): Promise<ControlAck> {
  const res = await fetch("/api/control", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(cmd),
  });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`sendControl: ${res.status}${body ? ` — ${body.trim()}` : ""}`);
  }
  return jsonBody<ControlAck>(res);
}

export async function fetchConfig(): Promise<ConfigDefaults> {
  const res = await fetch("/api/config");
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`fetchConfig: ${res.status}${body ? ` — ${body.trim()}` : ""}`);
  }
  return jsonBody<ConfigDefaults>(res);
}

export async function fetchReviewDiff(reviewId: string): Promise<DiffResponse> {
  const res = await fetch(`/api/reviews/${encodeURIComponent(reviewId)}/diff`);
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`fetchReviewDiff: ${res.status}${body ? ` — ${body.trim()}` : ""}`);
  }
  return jsonBody<DiffResponse>(res);
}
