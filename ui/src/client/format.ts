/** Format a USD cost, showing "<$0.01" for sub-penny amounts. */
export function formatCost(usd: number): string {
  if (usd < 0.01) return "<$0.01";
  return `$${usd.toFixed(2)}`;
}

/** Format a duration in seconds to a human-readable string. */
export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m < 60) return `${m}m${s > 0 ? ` ${s}s` : ""}`;
  const h = Math.floor(m / 60);
  return `${h}h ${m % 60}m`;
}

/**
 * Middle-truncate a string, replacing the center with "…".
 * For paths with slashes, preserves first and last segments.
 * Mirrors Go's stringx.MiddleTruncate.
 */
export function middleTruncate(s: string, maxWidth: number): string {
  if (maxWidth <= 0) return "";
  if (s.length <= maxWidth) return s;
  if (maxWidth <= 1) return "…";

  if (s.includes("/")) return middleTruncatePath(s, maxWidth);

  const half = Math.floor((maxWidth - 1) / 2);
  const tail = maxWidth - 1 - half;
  return s.slice(0, half) + "…" + s.slice(s.length - tail);
}

function middleTruncatePath(s: string, maxWidth: number): string {
  const parts = s.split("/");

  if (parts.length >= 3) {
    const first = parts[0];
    const last = parts[parts.length - 1];
    let candidate = first + "/…/" + last;
    if (candidate.length <= maxWidth) {
      for (let i = parts.length - 2; i >= 1; i--) {
        const attempt = first + "/…/" + parts.slice(i).join("/");
        if (attempt.length <= maxWidth) {
          candidate = attempt;
        } else {
          break;
        }
      }
      return candidate;
    }
  }

  const half = Math.floor((maxWidth - 1) / 2);
  const tail = maxWidth - 1 - half;
  return s.slice(0, half) + "…" + s.slice(s.length - tail);
}
