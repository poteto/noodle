import { useEffect, useRef, useState } from "react";
import type { EventLine } from "~/client";

function MessageBubble({ event }: { event: EventLine }) {
  switch (event.category) {
    case "think":
      return (
        <div className="chat-msg chat-msg-think">
          <span className="chat-msg-label">{event.label}</span>
          <div className="chat-msg-body">{event.body}</div>
        </div>
      );
    case "tools":
      return (
        <div className="chat-msg chat-msg-tool">
          <span className="chat-msg-label">{event.label}</span>
          <pre className="chat-msg-code">{event.body}</pre>
        </div>
      );
    case "ticket":
      return (
        <div className="chat-msg chat-msg-system">
          <span className="chat-msg-divider">{event.label}</span>
        </div>
      );
    case "all":
      return (
        <div className="chat-msg chat-msg-default">
          <span className="chat-msg-label">{event.label}</span>
          <div className="chat-msg-body">{event.body}</div>
        </div>
      );
    default: {
      const _exhaustive: never = event.category;
      return _exhaustive;
    }
  }
}

export function ChatMessages({ events }: { events: EventLine[] }) {
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

  return (
    <div className="chat-messages" ref={containerRef} onScroll={handleScroll}>
      {events.map((event, i) => (
        <MessageBubble key={i} event={event} />
      ))}
      {events.length === 0 && (
        <div className="chat-empty">No events yet.</div>
      )}
    </div>
  );
}
