import { useCallback, useMemo, useRef, useState } from "react";
import { useSuspenseSnapshot, useSessionEvents, useSendControl, formatCost } from "~/client";
import type { Session } from "~/client";
import { ReviewBanner } from "./ReviewBanner";
import { StreamingDelta } from "./StreamingDelta";
import { groupConsecutiveTools } from "./group-tools";
import { VirtualizedFeed } from "./VirtualizedFeed";

function statusColor(status: string): string {
  if (status === "running") {
    return "var(--color-green)";
  }
  if (status === "failed") {
    return "var(--color-red)";
  }
  return "var(--color-text-secondary)";
}

function AgentHeader({ session, onStopAll }: { session: Session; onStopAll: () => void }) {
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
        <button type="button" className="feed-action-btn stop-btn" onClick={onStopAll}>
          Stop All
        </button>
      </div>
    </header>
  );
}

export function AgentFeed({ sessionId }: { sessionId: string }) {
  const { data: snapshot } = useSuspenseSnapshot();
  const initialEvents = snapshot.events_by_session[sessionId];
  const { data: events = [] } = useSessionEvents(sessionId, initialEvents);
  const { mutate: send } = useSendControl();
  const [input, setInput] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const resizeTextarea = useCallback(() => {
    const el = textareaRef.current;
    if (!el) {
      return;
    }
    el.style.height = "auto";
    el.style.height = `${el.scrollHeight}px`;
  }, []);

  const session = snapshot.sessions.find((s) => s.id === sessionId);
  const activeSession = snapshot.active.find((s) => s.id === sessionId && s.status === "running");
  const isSessionThinking =
    snapshot.loop_state === "running" &&
    Boolean(activeSession?.current_action?.trim());
  const pendingReview = snapshot.pending_reviews?.find((r) => r.session_id === sessionId);

  const items = useMemo(() => groupConsecutiveTools(events), [events]);

  function handleSubmit() {
    const prompt = input.trim();
    if (!prompt) {
      return;
    }
    send({ action: "steer", target: sessionId, prompt });
    setInput("");
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
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
      <AgentHeader session={session} onStopAll={handleStopAll} />

      <div className="feed-watermark">NOODLE</div>

      <VirtualizedFeed
        items={items}
        tail={isSessionThinking ? <StreamingDelta sessionId={sessionId} /> : undefined}
      />

      {pendingReview && <ReviewBanner review={pendingReview} />}

      <div className="input-area">
        <div className="input-label">
          {isSessionThinking ? (
            <>
              <span className="thinking-dots">
                <span className="thinking-dot" />
                <span className="thinking-dot" />
                <span className="thinking-dot" />
              </span>
              Thinking…
            </>
          ) : (
            "Steer this agent..."
          )}
        </div>
        <div className="input-row">
          <div className="input-row-field">
            <textarea
              ref={textareaRef}
              value={input}
              onChange={(e) => {
                setInput(e.target.value);
                resizeTextarea();
              }}
              onKeyDown={handleKeyDown}
              placeholder="Enter instructions or critique..."
              rows={1}
            />
            <div className="input-hint">
              <kbd>Enter</kbd>
              <span>send</span>
              <kbd>Shift+Enter</kbd>
              <span>newline</span>
            </div>
          </div>
          <div className="input-row-actions">
            {isSessionThinking && (
              <button type="button" className="btn-stop" onClick={handleStop}>
                Stop
              </button>
            )}
            <button
              type="button"
              className="btn-submit"
              onClick={handleSubmit}
              disabled={!input.trim()}
            >
              Send
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
