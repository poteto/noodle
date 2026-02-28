import type { Stage, StageStatus } from "~/client";

const dotClass: Record<StageStatus, string> = {
  completed: "done",
  active: "active",
  merging: "active",
  pending: "pending",
  failed: "failed",
  cancelled: "pending",
};

export function StageRail({ stages }: { stages: Stage[] }) {
  if (stages.length === 0) return null;

  return (
    <div className="stage-rail">
      {stages.map((stage, i) => {
        const isActive = stage.status === "active";
        return (
          <div key={stage.task_key || i} className={`stage-item ${isActive ? "current" : ""}`}>
            <div className={`stage-dot ${dotClass[stage.status] || "pending"}`} />
            <span>{stage.task_key || stage.skill || `Stage ${i + 1}`}</span>
          </div>
        );
      })}
    </div>
  );
}
