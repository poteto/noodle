import { useEffect, useLayoutEffect, useRef, useState, useCallback } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import type { EventLine } from "~/client";
import type { ToolGroupData } from "./group-tools";
import { MessageRow } from "./MessageRow";
import { ToolGroup } from "./ToolGroup";

function eventKey(event: { at: string; category: string; label: string; body: string }): string {
  return `${event.at}:${event.category}:${event.label}:${event.body}`;
}

function itemKey(item: EventLine | ToolGroupData): string {
  if ("kind" in item) {
    return `group-${item.events[0].at}-${item.label}`;
  }
  return eventKey(item);
}

/** Rough height estimate per item type, used before measurement. */
function estimateSize(item: EventLine | ToolGroupData): number {
  if ("kind" in item) {
    return 36;
  }
  if (item.category === "ticket") {
    return 40;
  }
  if (item.label === "Cost") {
    return 28;
  }
  if (["Read", "Edit", "Write", "Bash", "Glob", "Grep"].includes(item.label)) {
    return 32;
  }
  return 72;
}

interface VirtualizedFeedProps {
  items: (EventLine | ToolGroupData)[];
  /** Override the empty-state message. */
  emptyMessage?: string;
  /** Rendered after all virtual items, inside the scroll container. Never unmounted. */
  tail?: React.ReactNode;
}

export function VirtualizedFeed({ items, emptyMessage, tail }: VirtualizedFeedProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const prevItemCount = useRef(items.length);

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, []);

  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (index) => estimateSize(items[index]),
    overscan: 10,
    getItemKey: (index) => itemKey(items[index]),
    paddingEnd: 24,
  });

  // Auto-scroll to true bottom (past tail/thinking indicator) when new items arrive.
  useLayoutEffect(() => {
    if (autoScroll && items.length > 0 && items.length !== prevItemCount.current) {
      scrollToBottom();
    }
    prevItemCount.current = items.length;
  }, [items.length, autoScroll, scrollToBottom]);

  // Keep feed pinned when measured height changes without list-length changes
  // (for example streaming delta growth or row remeasurement).
  useEffect(() => {
    if (!autoScroll || typeof ResizeObserver === "undefined") {
      return;
    }
    const target = contentRef.current;
    if (!target) {
      return;
    }
    const observer = new ResizeObserver(() => {
      scrollToBottom();
    });
    observer.observe(target);
    return () => observer.disconnect();
  }, [autoScroll, scrollToBottom]);

  const handleScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) {
      return;
    }
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight <= 64;
    setAutoScroll((prev) => (prev === atBottom ? prev : atBottom));
  }, []);

  const handleScrollToBottom = useCallback(() => {
    scrollToBottom();
    setAutoScroll(true);
  }, [scrollToBottom]);

  const virtualItems = virtualizer.getVirtualItems();
  const totalSize = virtualizer.getTotalSize();

  return (
    <>
      <div ref={scrollRef} className="feed-content" onScroll={handleScroll}>
        <div ref={contentRef}>
          {items.length === 0 ? (
            <div
              style={{
                textAlign: "center",
                paddingTop: 40,
                color: "var(--color-text-tertiary)",
                fontFamily: "var(--font-mono)",
                fontSize: 13,
              }}
            >
              {emptyMessage ?? "No events yet."}
            </div>
          ) : (
            <div data-virtualized style={{ height: totalSize, position: "relative" }}>
              {virtualItems.map((virtualRow) => {
                const item = items[virtualRow.index];
                return (
                  <div
                    key={virtualRow.key}
                    data-index={virtualRow.index}
                    ref={virtualizer.measureElement}
                    style={{
                      position: "absolute",
                      top: 0,
                      left: 0,
                      width: "100%",
                      transform: `translateY(${virtualRow.start}px)`,
                    }}
                  >
                    {"kind" in item ? <ToolGroup group={item} /> : <MessageRow event={item} />}
                  </div>
                );
              })}
            </div>
          )}
          {tail}
          {/* Bottom spacer so content doesn't clip under input area */}
          <div style={{ height: 128, flexShrink: 0 }} />
        </div>
      </div>

      {!autoScroll && (
        <button
          type="button"
          onClick={handleScrollToBottom}
          className="btn-new-order"
          style={{
            position: "absolute",
            bottom: 100,
            left: "50%",
            transform: "translateX(-50%)",
            zIndex: 20,
          }}
        >
          Scroll to bottom
        </button>
      )}
    </>
  );
}
