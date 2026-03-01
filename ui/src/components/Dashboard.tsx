import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useSuspenseSnapshot, formatCost, formatDuration } from "~/client";
import type { Snapshot } from "~/client";

type SortKey = "id" | "title" | "status" | "host" | "model" | "duration" | "cost";
type SortDir = "asc" | "desc";

interface SessionRow {
  id: string;
  title: string;
  status: string;
  host: string;
  model: string;
  duration: number;
  cost: number;
}

function deriveSessionRows(snapshot: Snapshot): SessionRow[] {
  const rows: SessionRow[] = [];
  for (const s of [...snapshot.active, ...snapshot.recent]) {
    rows.push({
      id: s.id,
      title: s.title || s.display_name,
      status: s.status,
      host: s.remote_host || "local",
      model: s.model,
      duration: s.duration_seconds,
      cost: s.total_cost_usd,
    });
  }
  return rows;
}

function isCompletedStatus(status: string): boolean {
  const normalized = status.toLowerCase();
  return normalized === "merged" || normalized === "completed" || normalized === "done";
}

function needsReview(snapshot: Snapshot, sessionID: string): boolean {
  return (snapshot.pending_reviews ?? []).some((review) => review.session_id === sessionID);
}

function sortRows(rows: SessionRow[], key: SortKey, dir: SortDir): SessionRow[] {
  const sorted = [...rows];
  sorted.sort((a, b) => {
    const av = a[key];
    const bv = b[key];
    if (typeof av === "number" && typeof bv === "number") {
      return dir === "asc" ? av - bv : bv - av;
    }
    const as = String(av);
    const bs = String(bv);
    return dir === "asc" ? as.localeCompare(bs) : bs.localeCompare(as);
  });
  return sorted;
}

function statusClass(status: string): string {
  const s = status.toLowerCase();
  if (s === "running" || s === "starting") {
    return "bg-accent text-bg-depth";
  }
  if (s === "merged" || s === "completed" || s === "done") {
    return "bg-green text-bg-depth";
  }
  if (s === "failed" || s === "error") {
    return "bg-red text-bg-depth";
  }
  return "bg-border-subtle text-text-primary";
}

const COLUMNS: { key: SortKey; label: string }[] = [
  { key: "id", label: "ID" },
  { key: "title", label: "TITLE" },
  { key: "status", label: "STATUS" },
  { key: "host", label: "HOST" },
  { key: "model", label: "MODEL" },
  { key: "duration", label: "DURATION" },
  { key: "cost", label: "COST" },
];

export function Dashboard() {
  const { data: snapshot } = useSuspenseSnapshot();
  const navigate = useNavigate();
  const [sortKey, setSortKey] = useState<SortKey>("id");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const rows = sortRows(deriveSessionRows(snapshot), sortKey, sortDir);

  const activeCount = snapshot.active.length;
  const recentCount = snapshot.recent.length;
  const totalCount = activeCount + recentCount;

  function handleSort(key: SortKey) {
    if (key === sortKey) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  }

  function handleRowClick(row: SessionRow) {
    if (isCompletedStatus(row.status) || needsReview(snapshot, row.id)) {
      navigate({ to: "/review" });
      return;
    }
    navigate({ to: "/actor/$id", params: { id: row.id } });
  }

  return (
    <div className="flex flex-col h-full bg-bg-depth text-text-primary">
      {/* Header */}
      <header className="feed-header">
        <div className="feed-title">Dashboard</div>
        <button
          type="button"
          onClick={() => navigate({ to: "/" })}
          className="feed-action-btn"
          style={{
            background: "var(--color-accent)",
            color: "var(--color-bg-depth)",
            fontWeight: 700,
          }}
        >
          New Order
        </button>
      </header>

      {/* Stats bar */}
      <div className="flex gap-4 px-6 py-4 border-b border-border-subtle" data-testid="stats-bar">
        <StatCard label="TOTAL" value={totalCount} />
        <StatCard label="ACTIVE" value={activeCount} />
        <StatCard label="COMPLETED" value={recentCount} />
        <StatCard label="COST" value={formatCost(snapshot.total_cost_usd)} />
      </div>

      {/* Session grid */}
      <div className="flex-1 overflow-auto px-6 py-4">
        <table className="w-full border-collapse">
          <thead>
            <tr className="border-b-2 border-accent">
              {COLUMNS.map((col) => (
                <th
                  key={col.key}
                  onClick={() => handleSort(col.key)}
                  className="text-left px-3 py-2 font-display font-bold text-sm text-accent tracking-wider cursor-pointer select-none hover:underline transition-colors duration-150"
                >
                  {col.label}
                  {sortKey === col.key && (
                    <span className="ml-1">{sortDir === "asc" ? "\u2191" : "\u2193"}</span>
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr
                key={row.id}
                onClick={() => handleRowClick(row)}
                className="border-b border-border-subtle hover:bg-bg-surface transition-[background,border-left] duration-[120ms] hover:border-l-2 hover:border-l-accent cursor-pointer"
              >
                <td className="px-3 py-2 font-mono text-sm">{row.id}</td>
                <td className="px-3 py-2 font-mono text-sm max-w-[300px] truncate">{row.title}</td>
                <td className="px-3 py-2">
                  <span
                    className={`inline-block px-2 py-0.5 font-mono text-xs font-bold uppercase ${statusClass(row.status)}`}
                    data-testid="status-badge"
                  >
                    {row.status}
                  </span>
                </td>
                <td className="px-3 py-2 font-mono text-sm">{row.host}</td>
                <td className="px-3 py-2 font-mono text-sm">{row.model}</td>
                <td className="px-3 py-2 font-mono text-sm">{formatDuration(row.duration)}</td>
                <td className="px-3 py-2 font-mono text-sm">{formatCost(row.cost)}</td>
              </tr>
            ))}
            {rows.length === 0 && (
              <tr>
                <td
                  colSpan={7}
                  className="px-3 py-8 text-center font-mono text-sm text-border-subtle"
                >
                  No sessions
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="border border-border-subtle bg-bg-surface px-4 py-3 transition-[border-color] duration-150 hover:border-border-active">
      <div className="font-mono text-xl font-bold text-text-primary">{value}</div>
      <div className="font-display text-xs font-semibold text-text-secondary uppercase tracking-wider">
        {label}
      </div>
    </div>
  );
}
