import type { Session } from "~/client";
import { middleTruncate, formatDuration, formatCost } from "~/client";
import { useControl } from "./ControlContext";
import { WorktreeLabel } from "./WorktreeLabel";
import { Badge } from "./Badge";
import { Square } from "lucide-react";
import { Tooltip } from "./Tooltip";

const EVENT_TYPE_CLASS: Record<string, string> = {
  Read: "bg-[#c8d8f0] text-[#2a4060]",
  Edit: "bg-[#f0e0a8] text-[#6a5010]",
  Write: "bg-[#d6e4c8] text-[#3a5028]",
  Bash: "bg-[#e8dfc0] text-[#4a4020]",
  Grep: "bg-[#dcd0e8] text-[#3a2860]",
  Glob: "bg-[#dcd0e8] text-[#3a2860]",
  Task: "bg-[#f0d8b8] text-[#6a4018]",
  Skill: "bg-[#f0d8b8] text-[#6a4018]",
  WebFetch: "bg-[#c8d8f0] text-[#2a4060]",
  WebSearch: "bg-[#c8d8f0] text-[#2a4060]",
};

function CurrentAction({ action }: { action: string }) {
  if (!action) {
    return <span className="text-text-3">working...</span>;
  }
  const spaceIdx = action.indexOf(" ");
  const type = spaceIdx > 0 ? action.slice(0, spaceIdx) : action;
  const rest = spaceIdx > 0 ? action.slice(spaceIdx + 1) : "";
  const badgeClass = EVENT_TYPE_CLASS[type];
  if (!badgeClass) {
    return <>{middleTruncate(action, 80)}</>;
  }
  return (
    <>
      <span
        className={`font-mono text-[0.625rem] font-bold px-1.5 py-px inline-block align-middle mr-1.5 ${badgeClass}`}
      >
        {type}
      </span>
      <span className="align-middle">{middleTruncate(rest, 70)}</span>
    </>
  );
}

export function AgentCard({ session, onClick }: { session: Session; onClick?: () => void }) {
  const send = useControl();
  const taskKey = session.task_key ?? "";

  function handleStop(e: React.MouseEvent) {
    e.stopPropagation();
    send({ action: "stop", name: session.id });
  }

  return (
    <button
      type="button"
      className="group cursor-pointer text-left w-full bg-transparent border-none p-0"
      onClick={onClick}
    >
      <div className="bg-bg-1 border-2 border-border p-[18px] shadow-card transition-[transform,box-shadow] duration-150 ease-out group-hover:-translate-x-0.5 group-hover:-translate-y-1 group-hover:shadow-card-hover">
        <div className="flex items-center gap-1.5 mb-2">
          {taskKey ? <Badge type={taskKey} /> : null}
          {session.remote_host && (
            <Tooltip content={session.remote_host}>
              <span className="relative flex items-center text-text-3">cloud</span>
            </Tooltip>
          )}
        </div>

        <div className="font-bold text-[1.0625rem] text-text-0 mb-0.5 whitespace-nowrap overflow-hidden text-ellipsis">
          {session.title || session.display_name}
          {session.retry_count > 0 && (
            <span className="ml-1.5 font-mono text-xs font-semibold text-norange">
              retry {session.retry_count}
            </span>
          )}
        </div>
        {session.title && (
          <div className="font-mono text-xs text-text-3 mb-1">{session.display_name}</div>
        )}
        <div className="text-[0.8125rem] text-text-2 leading-[1.4] mb-2.5 whitespace-nowrap overflow-hidden text-ellipsis">
          <CurrentAction action={session.current_action || ""} />
        </div>

        <div className="mb-2">
          <div className="h-1.5 bg-bg-3 overflow-hidden mb-1">
            <div
              className="h-full bg-accent progress-fill"
              style={{ width: `${Math.round(session.context_window_usage_pct)}%` }}
            />
          </div>
          <div className="font-mono text-xs text-text-3 text-right">
            ctx {Math.round(session.context_window_usage_pct)}%
          </div>
        </div>

        <div className="flex items-center gap-1.5 font-mono text-xs text-text-2 mt-0.5">
          <WorktreeLabel name={session.worktree_name} />
          <span>{formatDuration(session.duration_seconds)}</span>
          <span className="text-text-3">/</span>
          <span>{formatCost(session.total_cost_usd)}</span>
          {session.dispatch_warning && (
            <Tooltip content={session.dispatch_warning}>
              <span className="inline-flex items-center justify-center w-4 h-4 bg-norange text-white text-[0.625rem] font-extrabold ml-auto cursor-help">
                !
              </span>
            </Tooltip>
          )}
          <span className="px-1.5 py-px bg-bg-3 text-[0.6875rem] text-text-2 ml-auto">
            {session.model}
          </span>
          <Tooltip content="Stop and return to queue">
            <button
              type="button"
              className="flex items-center justify-center w-6 h-6 p-0 bg-transparent border-[1.5px] border-border-subtle text-text-2 cursor-pointer shrink-0 transition-all duration-[0.12s] hover:not-disabled:border-nred hover:not-disabled:text-nred hover:not-disabled:bg-nred-dim active:not-disabled:scale-90"
              onClick={handleStop}
            >
              <Square size={10} fill="currentColor" />
            </button>
          </Tooltip>
        </div>
      </div>
    </button>
  );
}
