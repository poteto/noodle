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

export async function fetchSnapshot(): Promise<Snapshot> {
  const res = await fetch("/api/snapshot");
  if (!res.ok) throw new Error(`fetchSnapshot: ${res.status}`);
  return jsonBody<Snapshot>(res);
}

export async function fetchSessionEvents(
  sessionId: string,
  after?: string,
): Promise<EventLine[]> {
  const params = new URLSearchParams();
  if (after) params.set("after", after);
  const qs = params.toString();
  const url = `/api/sessions/${encodeURIComponent(sessionId)}/events${qs ? `?${qs}` : ""}`;
  const res = await fetch(url);
  if (!res.ok) throw new Error(`fetchSessionEvents: ${res.status}`);
  return jsonBody<EventLine[]>(res);
}

export async function sendControl(cmd: ControlCommand): Promise<ControlAck> {
  const res = await fetch("/api/control", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(cmd),
  });
  if (!res.ok) throw new Error(`sendControl: ${res.status}`);
  return jsonBody<ControlAck>(res);
}

export async function fetchConfig(): Promise<ConfigDefaults> {
  const res = await fetch("/api/config");
  if (!res.ok) throw new Error(`fetchConfig: ${res.status}`);
  return jsonBody<ConfigDefaults>(res);
}

export async function fetchReviewDiff(
  reviewId: string,
): Promise<DiffResponse> {
  const res = await fetch(
    `/api/reviews/${encodeURIComponent(reviewId)}/diff`,
  );
  if (!res.ok) throw new Error(`fetchReviewDiff: ${res.status}`);
  return jsonBody<DiffResponse>(res);
}
