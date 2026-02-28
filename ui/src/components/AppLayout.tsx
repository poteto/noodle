import { Suspense, useEffect, useCallback } from "react";
import { ActiveChannelProvider, useActiveChannel, useSuspenseSnapshot } from "~/client";
import type { ChannelId } from "~/client";
import { Sidebar } from "./Sidebar";
import { FeedPanel } from "./FeedPanel";
import { ContextPanel } from "./ContextPanel";

function buildChannelList(snapshot: { orders: Array<{ stages: Array<{ session_id?: string }> }> }): ChannelId[] {
  const channels: ChannelId[] = [{ type: "scheduler" }];
  for (const order of snapshot.orders) {
    for (const stage of order.stages) {
      if (stage.session_id) {
        channels.push({ type: "agent", sessionId: stage.session_id });
      }
    }
  }
  return channels;
}

function channelIndex(channels: ChannelId[], active: ChannelId): number {
  return channels.findIndex((ch) => {
    if (ch.type !== active.type) return false;
    if (ch.type === "agent" && active.type === "agent") return ch.sessionId === active.sessionId;
    return true;
  });
}

function KeyboardHandler() {
  const { activeChannel, setActiveChannel } = useActiveChannel();
  const { data: snapshot } = useSuspenseSnapshot();

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement).tagName;
      const inInput = tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";

      if (e.key === "n" && !inInput && !e.metaKey && !e.ctrlKey) {
        e.preventDefault();
        const textarea = document.querySelector<HTMLTextAreaElement>("textarea");
        textarea?.focus();
        return;
      }

      if ((e.key === "ArrowUp" || e.key === "ArrowDown") && !inInput) {
        e.preventDefault();
        const channels = buildChannelList(snapshot);
        if (channels.length < 2) return;
        const idx = channelIndex(channels, activeChannel);
        const next =
          e.key === "ArrowDown"
            ? (idx + 1) % channels.length
            : (idx - 1 + channels.length) % channels.length;
        setActiveChannel(channels[next]);
      }
    },
    [activeChannel, setActiveChannel, snapshot],
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  return null;
}

export function AppLayout() {
  return (
    <ActiveChannelProvider>
      <Suspense fallback={<div className="h-screen bg-bg-depth" />}>
        <KeyboardHandler />
        <div className="grid grid-cols-[260px_1fr_300px] h-screen bg-bg-depth text-text-primary font-body">
          <Sidebar />
          <FeedPanel />
          <ContextPanel />
        </div>
      </Suspense>
    </ActiveChannelProvider>
  );
}
