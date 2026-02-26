import type { Order, Stage, StageStatus } from "~/client";
import { Badge } from "./Badge";

const STAGE_INDICATOR: Record<StageStatus, { symbol: string; class: string }> = {
  completed: { symbol: "\u2713", class: "text-ngreen" },
  active: { symbol: "\u25CF", class: "text-nyellow" },
  pending: { symbol: "\u25CB", class: "text-text-3" },
  failed: { symbol: "\u2717", class: "text-nred" },
  cancelled: { symbol: "\u2013", class: "text-text-3" },
};

function StageIndicator({ stage }: { stage: Stage }) {
  const ind = STAGE_INDICATOR[stage.status];
  return (
    <span className={`font-mono text-xs ${ind.class}`} title={`${stage.task_key ?? "stage"}: ${stage.status}`}>
      {ind.symbol}
    </span>
  );
}

function StagePipeline({ stages, label }: { stages: Stage[]; label?: string }) {
  if (stages.length === 0) {
    return null;
  }
  return (
    <div className="flex items-center gap-1 font-mono text-xs text-text-2">
      {label && <span className="text-text-3 mr-0.5">{label}</span>}
      {stages.map((stage, i) => (
        <span key={stage.task_key ?? i} className="flex items-center gap-1">
          {i > 0 && <span className="text-text-3">&rarr;</span>}
          <span className="flex items-center gap-0.5">
            {stage.task_key && <span>{stage.task_key}</span>}
            <StageIndicator stage={stage} />
          </span>
        </span>
      ))}
    </div>
  );
}

/** Find the active stage, or the first pending stage if none active. */
function currentStage(order: Order): Stage | undefined {
  return order.stages.find((s) => s.status === "active") ?? order.stages.find((s) => s.status === "pending");
}

export function OrderCard({
  order,
  index,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
  isDragOver,
  isDragging,
}: {
  order: Order;
  index?: number;
  onDragStart?: (e: React.DragEvent, index: number) => void;
  onDragOver?: (e: React.DragEvent, index: number) => void;
  onDrop?: (e: React.DragEvent, index: number) => void;
  onDragEnd?: () => void;
  isDragOver?: boolean;
  isDragging?: boolean;
}) {
  const active = currentStage(order);
  const isSingleStage = order.stages.length <= 1;
  const isSchedule = order.stages[0]?.task_key === "schedule";
  const isFailing = order.status === "failing";

  const classes = [
    "bg-bg-1 border-2 border-border p-[18px] shadow-card transition-[transform,box-shadow] duration-150 ease-out group-hover:-translate-x-0.5 group-hover:-translate-y-1 group-hover:shadow-card-hover",
    isDragOver && "border-t-[3px] border-t-nyellow pt-[15px]",
    isDragging && "rotate-[-2deg] opacity-60 scale-[1.02] shadow-poster-md",
    isSchedule && "border-l-4 border-l-norange bg-norange-bg",
    isFailing && "border-l-4 border-l-nred",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div
      className="group"
      draggable
      onDragStart={(e) => onDragStart?.(e, index ?? 0)}
      onDragOver={(e) => {
        e.preventDefault();
        onDragOver?.(e, index ?? 0);
      }}
      onDrop={(e) => {
        e.preventDefault();
        onDrop?.(e, index ?? 0);
      }}
      onDragEnd={onDragEnd}
    >
      <div className={classes}>
        <div className="flex items-center gap-1.5 mb-2">
          {active?.task_key && <Badge type={active.task_key} />}
        </div>
        <div className="font-bold text-[1.0625rem] text-text-0 mb-1">{order.title || order.id}</div>
        {order.rationale && isSchedule && (
          <div className="font-mono text-xs text-text-2 leading-[1.4] mb-2 italic">
            {order.rationale}
          </div>
        )}
        {!isSingleStage && (
          <div className="mt-2 mb-1 flex flex-col gap-1">
            <StagePipeline stages={order.stages} />
            {isFailing && order.on_failure && order.on_failure.length > 0 && (
              <StagePipeline stages={order.on_failure} label="recovery" />
            )}
          </div>
        )}
        <div className="flex items-center gap-1.5 font-mono text-xs text-text-2 mt-0.5">
          {active?.model && (
            <span className="px-1.5 py-px bg-bg-3 text-[0.6875rem] text-text-2 ml-auto">
              {active.model}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
