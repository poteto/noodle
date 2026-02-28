import { useEffect, useRef, useState } from "react";
import { useSuspenseSnapshot, useSessionEvents, useSendControl, formatCost } from "~/client";
import type { Session } from "~/client";
import { MessageRow } from "./MessageRow";

function findSchedulerSession(sessions: Session[]): Session | undefined {
  return sessions.find((s) => s.task_key?.toLowerCase().trim() === "schedule");
}

export function SchedulerFeed() {
  const { data: snapshot } = useSuspenseSnapshot();
  const { mutate: send, isPending } = useSendControl();
  const [input, setInput] = useState("");

  const schedulerSession = findSchedulerSession(snapshot.sessions);
  const { data: events = [] } = useSessionEvents(schedulerSession?.id);

  const containerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  useEffect(() => {
    if (autoScroll && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [events, autoScroll]);

  function handleScroll() {
    const el = containerRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
    setAutoScroll(atBottom);
  }

  function handleSubmit() {
    const prompt = input.trim();
    if (!prompt) return;
    send({ action: "steer", name: "schedule", prompt });
    setInput("");
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  }

  function handleStop() {
    if (schedulerSession) {
      send({ action: "stop", name: schedulerSession.id });
    }
  }

  return (
    <>
      <header className="feed-header">
        <div className="feed-title">
          Scheduler
          <span className="feed-badge badge-task">{snapshot.loop_state}</span>
          {schedulerSession && (
            <span className="feed-badge">{schedulerSession.model}</span>
          )}
        </div>
        <div className="feed-actions">
          <span style={{ fontFamily: "var(--font-mono)", fontSize: 12, color: "var(--color-text-tertiary)" }}>
            {formatCost(snapshot.total_cost_usd)}
          </span>
          {schedulerSession?.status === "running" && (
            <button type="button" className="feed-action-btn stop-btn" onClick={handleStop}>
              Stop
            </button>
          )}
        </div>
      </header>

      <div
        ref={containerRef}
        className="feed-content"
        onScroll={handleScroll}
      >
        {events.length === 0 && (
          <div style={{ textAlign: "center", paddingTop: 40, color: "var(--color-text-tertiary)", fontFamily: "var(--font-mono)", fontSize: 13 }}>
            {schedulerSession ? "No events yet." : "No scheduler session found. Send a prompt to start."}
          </div>
        )}
        {events.map((event) => (
          <MessageRow key={event.at} event={event} />
        ))}
      </div>

      {!autoScroll && (
        <button
          type="button"
          onClick={() => {
            if (containerRef.current) {
              containerRef.current.scrollTop = containerRef.current.scrollHeight;
            }
            setAutoScroll(true);
          }}
          className="btn-new-order"
          style={{ position: "absolute", bottom: 100, left: "50%", transform: "translateX(-50%)", zIndex: 20 }}
        >
          New messages
        </button>
      )}

      <div className="input-area">
        <div className="input-wrapper">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Talk to the scheduler..."
            rows={1}
          />
          <div className="input-footer">
            <div className="input-hint">
              <kbd>Enter</kbd> send · <kbd>Shift+Enter</kbd> newline
            </div>
            <button
              type="button"
              className="btn-submit"
              onClick={handleSubmit}
              disabled={isPending || !input.trim()}
            >
              SEND
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
