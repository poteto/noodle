import { useEffect, useRef, useState } from "react";
import { useSuspenseSnapshot, useSessionEvents, useSendControl, formatCost } from "~/client";
import type { Session } from "~/client";
import { MessageRow } from "./MessageRow";
import { ReviewBanner } from "./ReviewBanner";
import { StreamingDelta } from "./StreamingDelta";

function statusColor(status: string): string {
  if (status === "running") {
    return "var(--color-green)";
  }
  if (status === "failed") {
    return "var(--color-red)";
  }
  return "var(--color-text-secondary)";
}

function AgentHeader({ session, onStop }: { session: Session; onStop: () => void }) {
  const color = statusColor(session.status);
  return (
    <header className="feed-header">
      <div className="feed-title">
        {session.display_name}
        {session.task_key && <span className="feed-badge badge-task">{session.task_key}</span>}
        <span className="feed-badge">{session.model}</span>
        <span className="feed-badge" style={{ color, borderColor: color }}>
          {session.status}
        </span>
      </div>
      <div className="feed-actions">
        <span
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: 12,
            color: "var(--color-text-tertiary)",
          }}
        >
          {formatCost(session.total_cost_usd)}
        </span>
        <button type="button" className="feed-action-btn stop-btn" onClick={onStop}>
          Stop
        </button>
      </div>
    </header>
  );
}

export function AgentFeed({ sessionId }: { sessionId: string }) {
  const { data: snapshot } = useSuspenseSnapshot();
  const { data: events = [] } = useSessionEvents(sessionId);
  const { mutate: send } = useSendControl();
  const [input, setInput] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  const session = snapshot.sessions.find((s) => s.id === sessionId);
  const pendingReview = snapshot.pending_reviews?.find((r) => r.session_id === sessionId);

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
    send({ action: "steer", target: sessionId, prompt });
    setInput("");
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  }

  function handleStop() {
    send({ action: "stop", name: sessionId });
  }

  if (!session) {
    return (
      <div className="feed-container" style={{ alignItems: "center", justifyContent: "center" }}>
        <span
          style={{
            color: "var(--color-text-tertiary)",
            fontFamily: "var(--font-mono)",
            fontSize: 14,
          }}
        >
          Session not found
        </span>
      </div>
    );
  }

  return (
    <>
      <AgentHeader session={session} onStop={handleStop} />

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
            No events yet.
          </div>
        )}
        {events.map((event) => (
          <MessageRow key={event.at} event={event} />
        ))}
        {session.status === "running" && <StreamingDelta sessionId={sessionId} />}
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

      {pendingReview && <ReviewBanner review={pendingReview} />}

      <div className="input-area">
        <div className="input-wrapper">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Steer this agent..."
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
              disabled={!input.trim()}
            >
              SEND
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
