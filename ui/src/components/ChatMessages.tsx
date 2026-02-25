import { useEffect, useRef, useState } from "react";
import type { EventLine } from "~/client";

function MessageBubble({ event }: { event: EventLine }) {
  switch (event.category) {
    case "think":
      return (
        <div className="px-3 py-2 text-[0.8125rem] leading-normal bg-bg-2 border-l-[3px] border-border">
          <span className="font-mono text-xs font-semibold block mb-0.5 text-text-0">{event.label}</span>
          <div className="text-text-1 whitespace-pre-wrap break-words">{event.body}</div>
        </div>
      );
    case "tools":
      return (
        <div className="px-3 py-2 text-[0.8125rem] leading-normal bg-bg-2 border-l-[3px] border-nblue">
          <span className="font-mono text-xs font-semibold block mb-0.5 text-nblue">{event.label}</span>
          <pre className="font-mono text-xs text-text-1 overflow-x-auto whitespace-pre-wrap break-all m-0">{event.body}</pre>
        </div>
      );
    case "ticket":
      return (
        <div className="text-center py-1 px-0">
          <span className="font-mono text-xs text-text-3 px-3 py-0.5 bg-bg-2">{event.label}</span>
        </div>
      );
    case "all":
      return (
        <div className="px-3 py-2 text-[0.8125rem] leading-normal">
          <span className="font-mono text-xs font-semibold block mb-0.5 text-text-2">{event.label}</span>
          <div className="text-text-1 whitespace-pre-wrap break-words">{event.body}</div>
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
    <div className="flex-1 overflow-y-auto px-5 py-4 flex flex-col gap-2" ref={containerRef} onScroll={handleScroll}>
      {events.map((event, i) => (
        <MessageBubble key={i} event={event} />
      ))}
      {events.length === 0 && (
        <div className="text-text-3 font-mono text-[0.8125rem] text-center pt-10">No events yet.</div>
      )}
    </div>
  );
}
