import type { TreeNodeData } from "./tree-utils";
import { formatCost } from "~/client";

const statusColors: Record<string, string> = {
  active: "bg-[var(--color-accent)]",
  running: "bg-[var(--color-green)]",
  completed: "bg-[var(--color-green)]",
  failed: "bg-[var(--color-red)]",
  pending: "bg-[var(--color-border-subtle)]",
  paused: "bg-[var(--color-accent)]",
};

export function TreeNodeCard({ data }: { data: TreeNodeData }) {
  const isActive = data.status === "active" || data.status === "running";
  const dotColor = statusColors[data.status] ?? "bg-[var(--color-border-subtle)]";

  return (
    <div
      className="font-mono text-xs leading-tight"
      style={{
        background: "var(--color-bg-surface)",
        border: `1px solid ${isActive ? "var(--color-accent)" : "var(--color-border-subtle)"}`,
        padding: "8px 10px",
        width: "160px",
        color: "var(--color-text-primary)",
      }}
    >
      <div className="flex items-center gap-1.5 mb-1">
        <span className={`inline-block w-2 h-2 shrink-0 ${dotColor}`} />
        <span className="truncate font-medium">{data.name}</span>
      </div>
      {data.currentAction && (
        <div className="truncate text-[10px] opacity-60">{data.currentAction}</div>
      )}
      {data.cost != null && (
        <div className="text-[10px] opacity-60 mt-0.5">{formatCost(data.cost)}</div>
      )}
      {data.model && !data.currentAction && (
        <div className="text-[10px] opacity-60">{data.model}</div>
      )}
    </div>
  );
}
