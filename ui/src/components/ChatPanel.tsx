import type { Session } from "~/client";
import { useSessionEvents } from "~/client";
import { Badge } from "./Badge";
import { ChatMessages } from "./ChatMessages";
import { ChatInput } from "./ChatInput";
import { SidePanel } from "./SidePanel";
import { Tooltip } from "./Tooltip";

export function ChatPanel({
  session,
  onClose,
}: {
  session: Session;
  onClose: () => void;
}) {
  const { data: events } = useSessionEvents(session.id);

  const taskKey = session.task_key ?? "";

  return (
    <SidePanel defaultWidth={560} onClose={onClose}>
      <div className="px-5 pt-[18px] pb-3 border-b-2 border-border shrink-0">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            {taskKey && <Badge type={taskKey} />}
            <Tooltip content={session.display_name}>
              <span className="inline-flex items-center justify-center w-6 h-6 bg-bg-3 font-mono text-[0.625rem] font-bold text-text-2 uppercase shrink-0 select-none">
                {session.display_name.slice(0, 2)}
              </span>
            </Tooltip>
            <span className="font-bold text-[1.125rem] text-text-0 whitespace-nowrap overflow-hidden text-ellipsis">
              {session.title || session.display_name}
            </span>
          </div>
          <button
            className="bg-transparent border-2 border-border py-0.5 px-[10px] font-mono text-[0.8125rem] font-bold cursor-pointer text-text-1 hover:bg-bg-hover active:translate-x-px active:translate-y-px active:shadow-btn-active"
            onClick={onClose}
          >
            x
          </button>
        </div>
        <div className="flex gap-2 font-mono text-xs text-text-2 mb-2">
          <span className="py-px px-[6px] bg-bg-3">{session.model}</span>
          <span className="py-px px-[6px] bg-bg-3">
            ctx {Math.round(session.context_window_usage_pct)}%
          </span>
          <span className="py-px px-[6px] bg-bg-3">
            ${session.total_cost_usd.toFixed(2)}
          </span>
        </div>
        <div className="h-1 bg-bg-3 overflow-hidden">
          <div
            className="h-full bg-accent progress-fill"
            style={{ width: `${Math.round(session.context_window_usage_pct)}%` }}
          />
        </div>
      </div>

      <ChatMessages events={events ?? []} />

      <ChatInput sessionId={session.id} />
    </SidePanel>
  );
}
