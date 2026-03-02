import type { Snapshot } from "./types";

function parseSnapshotTime(value: string): number | null {
  const ms = Date.parse(value);
  return Number.isNaN(ms) ? null : ms;
}

// chooseNewerSnapshot keeps the most recent snapshot by updated_at.
// This prevents older HTTP responses from clobbering newer WS cache state.
export function chooseNewerSnapshot(current: Snapshot | undefined, incoming: Snapshot): Snapshot {
  if (!current) {
    return incoming;
  }

  const currentMs = parseSnapshotTime(current.updated_at);
  const incomingMs = parseSnapshotTime(incoming.updated_at);

  if (currentMs === null && incomingMs === null) {
    return incoming;
  }
  if (currentMs === null) {
    return incoming;
  }
  if (incomingMs === null) {
    return current;
  }
  return incomingMs >= currentMs ? incoming : current;
}
