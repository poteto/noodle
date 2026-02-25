import { useEffect, useRef } from "react";
import type { Session } from "~/client";
import { useSessionEvents } from "~/client";
import { Badge } from "./Badge";
import { ChatMessages } from "./ChatMessages";
import { ChatInput } from "./ChatInput";

export function ChatPanel({
  session,
  onClose,
}: {
  session: Session;
  onClose: () => void;
}) {
  const backdropRef = useRef<HTMLDivElement>(null);
  const { data: events } = useSessionEvents(session.id);

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [onClose]);

  function handleBackdropClick(e: React.MouseEvent) {
    if (e.target === backdropRef.current) onClose();
  }

  const taskKey = session.display_name.split("-")[0] ?? "";

  return (
    <div
      className="fixed inset-0 bg-[rgba(26,20,0,0.3)] z-100 flex justify-end animate-fade-in"
      ref={backdropRef}
      onClick={handleBackdropClick}
    >
      <div className="w-[560px] max-w-[100vw] h-screen bg-bg-1 border-l-[3px] border-border flex flex-col shadow-chat animate-slide-right">
        <div className="px-5 pt-[18px] pb-3 border-b-2 border-border shrink-0">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              {taskKey && <Badge type={taskKey} />}
              <span className="font-bold text-[1.125rem] text-text-0">{session.display_name}</span>
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
              className="h-full bg-border progress-fill"
              style={{ width: `${Math.round(session.context_window_usage_pct)}%` }}
            />
          </div>
        </div>

        <ChatMessages events={events ?? []} />

        <ChatInput sessionId={session.id} />
      </div>
    </div>
  );
}
