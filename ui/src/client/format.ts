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
