import type {
  Snapshot,
  EventLine,
  ControlCommand,
  ControlAck,
  ConfigDefaults,
} from "./types";

export async function fetchSnapshot(): Promise<Snapshot> {
  const res = await fetch("/api/snapshot");
  if (!res.ok) throw new Error(`fetchSnapshot: ${res.status}`);
  return res.json();
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
  return res.json();
}

export async function sendControl(cmd: ControlCommand): Promise<ControlAck> {
  const res = await fetch("/api/control", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(cmd),
  });
  if (!res.ok) throw new Error(`sendControl: ${res.status}`);
  return res.json();
}

export async function fetchConfig(): Promise<ConfigDefaults> {
  const res = await fetch("/api/config");
  if (!res.ok) throw new Error(`fetchConfig: ${res.status}`);
  return res.json();
}
