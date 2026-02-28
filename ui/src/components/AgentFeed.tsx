import { useEffect, useRef, useState } from "react";
import { useSuspenseSnapshot, useSessionEvents, useSendControl, formatCost } from "~/client";
import type { Session } from "~/client";
import { MessageRow } from "./MessageRow";
import { ReviewBanner } from "./ReviewBanner";

function AgentHeader({ session, onStop }: { session: Session; onStop: () => void }) {
  return (
    <div className="p-4 border-b border-border-subtle flex items-center gap-3">
      <div className="flex-1 min-w-0">
        <h2 className="text-sm font-display font-bold uppercase tracking-wider truncate">
          {session.display_name}
        </h2>
        <div className="flex items-center gap-2 mt-1">
          {session.task_key && (
            <span className="text-[10px] font-body bg-accent text-black px-1.5 py-0.5 font-bold">
              {session.task_key}
            </span>
          )}
          <span className="text-[10px] font-body bg-neutral-800 text-neutral-300 px-1.5 py-0.5">
            {session.model}
          </span>
          <span
            className={`text-xs font-body ${
              session.status === "running"
                ? "text-green"
                : session.status === "failed"
                  ? "text-red"
                  : "text-neutral-500"
            }`}
          >
            {session.status}
          </span>
          <span className="text-xs text-neutral-600 font-body">
            {formatCost(session.total_cost_usd)}
          </span>
        </div>
      </div>
      <button
        type="button"
        onClick={onStop}
        className="text-xs font-body uppercase px-3 py-1.5 border border-red text-red hover:bg-red/10"
      >
        STOP
      </button>
    </div>
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
  const pendingReview = snapshot.pending_reviews?.find(
    (r) => r.session_id === sessionId,
  );

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
    send({ action: "steer", name: sessionId, prompt });
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
      <div className="flex items-center justify-center h-full text-neutral-600 text-sm font-body">
        Session not found
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      <AgentHeader session={session} onStop={handleStop} />

      <div className="relative flex-1 overflow-hidden">
        <div
          ref={containerRef}
          className="h-full overflow-y-auto py-3 scroll-smooth"
          onScroll={handleScroll}
        >
          {events.length === 0 && (
            <div className="text-neutral-600 text-sm font-body text-center pt-10">
              No events yet.
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
            className="absolute bottom-4 left-1/2 -translate-x-1/2 bg-accent text-black font-body text-xs font-bold uppercase tracking-wider px-3 py-1.5 animate-fade-in"
          >
            New messages
          </button>
        )}
      </div>

      {pendingReview && <ReviewBanner review={pendingReview} />}

      <div className="p-4 border-t border-border-subtle">
        <div className="flex gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Steer this agent..."
            rows={2}
            className="flex-1 bg-transparent border border-border-subtle focus:border-accent font-body text-sm text-text-primary p-2 resize-none outline-none placeholder:text-neutral-600"
          />
          <button
            type="button"
            onClick={handleSubmit}
            disabled={!input.trim()}
            className="self-end bg-accent text-black font-body uppercase text-xs px-4 py-2 disabled:opacity-50"
          >
            SEND
          </button>
        </div>
      </div>
    </div>
  );
}
