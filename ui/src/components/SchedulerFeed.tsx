import { useEffect, useRef, useState } from "react";
import { useSuspenseSnapshot, useSessionEvents, useSendControl, formatCost } from "~/client";
import type { Session } from "~/client";
import { MessageRow } from "./MessageRow";
import { StreamingDelta } from "./StreamingDelta";

function findSchedulerSession(sessions: Session[]): Session | undefined {
  return sessions.find((s) => s.task_key?.toLowerCase().trim() === "schedule");
}

function eventKey(event: { at: string; category: string; label: string; body: string }): string {
  return `${event.at}:${event.category}:${event.label}:${event.body}`;
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
    if (!el) {
      return;
    }
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
    setAutoScroll(atBottom);
  }

  function handleSubmit() {
    const prompt = input.trim();
    if (!prompt) {
      return;
    }
    send({ action: "steer", target: "schedule", prompt });
    setInput("");
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  }

  function handleStopAll() {
    send({ action: "stop-all" });
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
          {schedulerSession && <span className="feed-badge">{schedulerSession.model}</span>}
        </div>
        <div className="feed-actions">
          <span
            style={{
              fontFamily: "var(--font-mono)",
              fontSize: 12,
              color: "var(--color-text-tertiary)",
            }}
          >
            {formatCost(snapshot.total_cost_usd)}
          </span>
          <button type="button" className="feed-action-btn stop-btn" onClick={handleStopAll}>
            Stop All
          </button>
        </div>
      </header>

      <div className="feed-watermark">NOODLE</div>

      <div ref={containerRef} className="feed-content" onScroll={handleScroll}>
        {events.length === 0 && (
          <div
            style={{
              textAlign: "center",
              paddingTop: 40,
              color: "var(--color-text-tertiary)",
              fontFamily: "var(--font-mono)",
              fontSize: 13,
            }}
          >
            {schedulerSession
              ? "No events yet."
              : "No scheduler session found. Send a prompt to start."}
          </div>
        )}
        {events.map((event) => (
          <MessageRow key={eventKey(event)} event={event} />
        ))}
        {schedulerSession?.status === "running" && schedulerSession.id && (
          <StreamingDelta sessionId={schedulerSession.id} />
        )}
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
          style={{
            position: "absolute",
            bottom: 100,
            left: "50%",
            transform: "translateX(-50%)",
            zIndex: 20,
          }}
        >
          New messages
        </button>
      )}

      <div className="input-area">
        <div className="input-label">Talk to the scheduler...</div>
        <div className="input-row">
          <div className="input-row-field">
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Enter instructions or critique..."
              rows={1}
            />
            <div className="input-hint">
              <kbd>Enter</kbd><span>send</span>
              <kbd>Shift+Enter</kbd><span>newline</span>
            </div>
          </div>
          <div className="input-row-actions">
            {schedulerSession?.status === "running" && (
              <button type="button" className="btn-stop" onClick={handleStop}>
                Stop
              </button>
            )}
            <button
              type="button"
              className="btn-submit"
              onClick={handleSubmit}
              disabled={isPending || !input.trim()}
            >
              Send
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
