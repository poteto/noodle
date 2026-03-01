import { useEffect, useCallback } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useActiveChannel, useSuspenseSnapshot } from "~/client";
import type { ChannelId } from "~/client";

function buildChannelList(snapshot: {
  orders: { stages: { session_id?: string }[] }[];
}): ChannelId[] {
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
    if (ch.type !== active.type) {
      return false;
    }
    if (ch.type === "agent" && active.type === "agent") {
      return ch.sessionId === active.sessionId;
    }
    return true;
  });
}

export function FeedKeyboardHandler() {
  const { activeChannel } = useActiveChannel();
  const { data: snapshot } = useSuspenseSnapshot();
  const navigate = useNavigate();

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
        if (channels.length < 2) {
          return;
        }
        const idx = channelIndex(channels, activeChannel);
        const next =
          e.key === "ArrowDown"
            ? (idx + 1) % channels.length
            : (idx - 1 + channels.length) % channels.length;
        const nextChannel = channels[next];
        if (nextChannel.type === "scheduler") {
          navigate({ to: "/" });
        } else {
          navigate({ to: "/actor/$id", params: { id: nextChannel.sessionId } });
        }
      }
    },
    [activeChannel, navigate, snapshot],
  );

  useEffect(() => {
    globalThis.addEventListener("keydown", handleKeyDown);
    return () => globalThis.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  return null;
}
