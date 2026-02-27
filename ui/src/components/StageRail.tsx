import type { Stage, StageStatus } from "~/client";

const DOT_STYLE: Record<StageStatus, { symbol: string; class: string }> = {
  completed: { symbol: "\u25CF", class: "text-ngreen" },
  active: { symbol: "\u25C9", class: "text-nyellow animate-pulse-dot" },
  merging: { symbol: "\u25C9", class: "text-nyellow animate-pulse-dot" },
  pending: { symbol: "\u25CB", class: "text-text-3" },
  failed: { symbol: "\u2717", class: "text-nred" },
  cancelled: { symbol: "\u2013", class: "text-text-3" },
};

const MAX_VISIBLE = 8;

export function StageRail({
  stages,
  onSelectSession,
}: {
  stages: Stage[];
  onSelectSession?: (sessionId: string) => void;
}) {
  if (stages.length <= 1) {
    return null;
  }

  const compressed = stages.length > MAX_VISIBLE;
  const visible = compressed ? [...stages.slice(0, 3), null, ...stages.slice(-3)] : stages;

  return (
    <div className="flex flex-col items-center gap-0.5 w-4 shrink-0 py-1">
      {visible.map((stage, i) => {
        if (stage === null) {
          return (
            <span
              key="ellipsis"
              className="text-text-3 text-[0.5rem] leading-none select-none"
              title={`${stages.length - 6} more stages`}
            >
              &middot;&middot;&middot;
            </span>
          );
        }

        const dot = DOT_STYLE[stage.status];
        const clickable = stage.session_id && onSelectSession;
        return (
          <button
            key={compressed ? `${i}-${stage.task_key ?? "s"}` : `${i}`}
            type="button"
            className={`text-[0.625rem] leading-none ${dot.class}${clickable ? " cursor-pointer hover:scale-125 transition-transform" : " cursor-default"}`}
            title={`${stage.task_key ?? "stage"}: ${stage.status}`}
            disabled={!clickable}
            onClick={() => {
              if (stage.session_id && onSelectSession) {
                onSelectSession(stage.session_id);
              }
            }}
          >
            {dot.symbol}
          </button>
        );
      })}
    </div>
  );
}
