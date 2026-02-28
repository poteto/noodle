import type { Stage, StageStatus } from "~/client";

const dotColor: Record<StageStatus, string> = {
  completed: "bg-green",
  active: "bg-accent animate-pulse-dot",
  pending: "bg-neutral-600",
  failed: "bg-red",
  cancelled: "bg-neutral-700",
};

const lineColor: Record<StageStatus, string> = {
  completed: "bg-green/30",
  active: "bg-accent/30",
  pending: "bg-neutral-700",
  failed: "bg-red/30",
  cancelled: "bg-neutral-800",
};

export function StageRail({ stages }: { stages: Stage[] }) {
  if (stages.length === 0) return null;

  return (
    <div className="flex flex-col">
      {stages.map((stage, i) => {
        const isActive = stage.status === "active";
        return (
          <div key={stage.task_key || i} className="flex items-stretch gap-3">
            {/* Rail: dot + connector line */}
            <div className="flex flex-col items-center w-3">
              <div
                className={`w-2 h-2 mt-2 shrink-0 ${dotColor[stage.status] || "bg-neutral-600"}`}
              />
              {i < stages.length - 1 && (
                <div
                  className={`w-px flex-1 ${lineColor[stage.status] || "bg-neutral-700"}`}
                />
              )}
            </div>

            {/* Stage label */}
            <div
              className={`flex-1 py-1.5 px-2 text-xs font-mono truncate ${
                isActive
                  ? "text-accent border-l-2 border-accent"
                  : stage.status === "completed"
                    ? "text-neutral-400"
                    : stage.status === "failed"
                      ? "text-red"
                      : "text-neutral-500"
              }`}
            >
              {stage.task_key || stage.skill || `Stage ${i + 1}`}
            </div>
          </div>
        );
      })}
    </div>
  );
}
