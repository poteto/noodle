import { useState } from "react";
import type { Session } from "~/client";
import { sendControl, formatCost } from "~/client";
import { useControl } from "./ControlContext";
import { Badge } from "./Badge";
import { WorktreeLabel } from "./WorktreeLabel";
import { RotateCcw } from "lucide-react";
import { Tooltip } from "./Tooltip";

export function DoneCard({ session, onClick }: { session: Session; onClick?: () => void }) {
  const send = useControl();
  const [replaying, setReplaying] = useState(false);
  const failed = session.status === "failed";
  const taskKey = session.task_key ?? "";

  async function handleReplay(e: React.MouseEvent) {
    e.stopPropagation();
    if (failed) {
      // Requeue: card disappears optimistically via reducer.
      send({ action: "requeue", item: session.id });
    } else {
      // Replay: no clear optimistic state — track locally.
      setReplaying(true);
      try {
        await sendControl({
          action: "enqueue",
          item: `replay-${Date.now()}`,
          task_key: taskKey || "execute",
          prompt: `Replay: ${session.display_name}`,
          model: session.model,
          provider: session.provider,
        });
      } catch {
        /* fire-and-forget */
      } finally {
        setReplaying(false);
      }
    }
  }

  return (
    <button
      type="button"
      className="group cursor-pointer text-left w-full bg-transparent border-none p-0"
      onClick={onClick}
    >
      <div
        className={`bg-bg-1 border-2 border-border p-[18px] shadow-card transition-[transform,box-shadow] duration-150 ease-out group-hover:-translate-x-0.5 group-hover:-translate-y-1 group-hover:shadow-card-hover${failed ? " border-l-[3px] border-l-nred" : ""}`}
      >
        <div className="flex items-center gap-1.5 mb-2">
          {taskKey ? <Badge type={taskKey} /> : null}
        </div>
        {failed ? (
          <div className="flex items-center gap-1.5 mb-1 font-mono text-xs text-nred font-semibold">
            failed
          </div>
        ) : (
          <div className="flex items-center gap-1.5 mb-1.5 font-mono text-xs text-ngreen font-medium">
            done
          </div>
        )}
        <div className="font-bold text-[1.0625rem] text-text-0 mb-0.5 whitespace-nowrap overflow-hidden text-ellipsis">
          {session.title || session.display_name}
        </div>
        {session.title && (
          <div className="font-mono text-xs text-text-3 mb-1">{session.display_name}</div>
        )}
        <div className="flex items-center gap-1.5 font-mono text-xs text-text-2 mt-0.5">
          <WorktreeLabel name={session.worktree_name} />
          <span>{formatCost(session.total_cost_usd)}</span>
          <span className="px-1.5 py-px bg-bg-3 text-[0.6875rem] text-text-2 ml-auto">
            {session.model}
          </span>
          <Tooltip content={failed ? "Retry" : "Replay"}>
            <button
              type="button"
              className="flex items-center justify-center w-6 h-6 p-0 bg-transparent border-[1.5px] border-border-subtle text-text-2 cursor-pointer shrink-0 transition-all duration-[0.12s] hover:not-disabled:border-ngreen hover:not-disabled:text-ngreen hover:not-disabled:bg-ngreen-dim active:not-disabled:scale-90 disabled:opacity-30 disabled:cursor-not-allowed"
              onClick={handleReplay}
              disabled={replaying}
            >
              <RotateCcw size={12} />
            </button>
          </Tooltip>
        </div>
      </div>
    </button>
  );
}
