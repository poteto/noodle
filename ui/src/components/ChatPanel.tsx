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
    <div className="chat-backdrop" ref={backdropRef} onClick={handleBackdropClick}>
      <div className="chat-panel">
        <div className="chat-header">
          <div className="chat-header-top">
            <div className="chat-header-left">
              {taskKey && <Badge type={taskKey} />}
              <span className="chat-agent-name">{session.display_name}</span>
            </div>
            <button className="chat-close" onClick={onClose}>
              x
            </button>
          </div>
          <div className="chat-header-meta">
            <span className="chat-meta-item">{session.model}</span>
            <span className="chat-meta-item">
              ctx {Math.round(session.context_window_usage_pct)}%
            </span>
            <span className="chat-meta-item">
              ${session.total_cost_usd.toFixed(2)}
            </span>
          </div>
          <div className="chat-progress-track">
            <div
              className="chat-progress-fill"
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
