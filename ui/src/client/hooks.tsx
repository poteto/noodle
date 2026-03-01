import { createElement, createContext, useContext, useEffect, useSyncExternalStore } from "react";
import { useQuery, useSuspenseQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import {
  connectWS,
  SNAPSHOT_KEY,
  subscribeSession,
  unsubscribeSession,
  sendWSControl,
  subscribeWSStatus,
  getWSStatus,
} from "./ws";
import type { WSStatus } from "./ws";
import { fetchSnapshot, fetchSessionEvents, sendControl, fetchReviewDiff } from "./api";
import type {
  Snapshot,
  EventLine,
  ControlCommand,
  ControlAck,
  DiffResponse,
  ChannelId,
} from "./types";

// Connects WS on mount, seeds cache with initial fetch, and keeps
// snapshot data live via WebSocket.
export function useSnapshot() {
  const queryClient = useQueryClient();

  useEffect(() => connectWS(queryClient), [queryClient]);

  return useQuery<Snapshot>({
    queryKey: SNAPSHOT_KEY,
    queryFn: fetchSnapshot,
    // WS handles updates; only fetch once for initial seed.
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
}

// Suspense variant -- throws until the initial snapshot arrives.
// The component calling this must be wrapped in a <Suspense> boundary.
export function useSuspenseSnapshot() {
  const queryClient = useQueryClient();

  useEffect(() => connectWS(queryClient), [queryClient]);

  return useSuspenseQuery<Snapshot>({
    queryKey: SNAPSHOT_KEY,
    queryFn: fetchSnapshot,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
}

export function useSessionEvents(sessionId: string | undefined, initialData?: EventLine[]) {
  useEffect(() => {
    if (!sessionId) {
      return;
    }
    subscribeSession(sessionId);
    return () => unsubscribeSession(sessionId);
  }, [sessionId]);

  return useQuery<EventLine[]>({
    queryKey: ["sessionEvents", sessionId],
    queryFn: () => (sessionId ? fetchSessionEvents(sessionId) : []),
    enabled: Boolean(sessionId),
    initialData,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    refetchOnMount: false,
  });
}

export function useWSStatus(): WSStatus {
  return useSyncExternalStore(subscribeWSStatus, getWSStatus);
}

export function useReviewDiff(itemId: string) {
  return useQuery<DiffResponse>({
    queryKey: ["review-diff", itemId],
    queryFn: () => fetchReviewDiff(itemId),
    staleTime: Infinity,
  });
}

interface ActiveChannelContextValue {
  activeChannel: ChannelId;
  setActiveChannel: (channel: ChannelId) => void;
}

const ActiveChannelContext = createContext<ActiveChannelContextValue | null>(null);

export function ActiveChannelProvider({
  channel,
  onChannelChange,
  children,
}: {
  channel: ChannelId;
  onChannelChange: (channel: ChannelId) => void;
  children: React.ReactNode;
}) {
  return createElement(
    ActiveChannelContext.Provider,
    { value: { activeChannel: channel, setActiveChannel: onChannelChange } },
    children,
  );
}

export function useActiveChannel() {
  const ctx = useContext(ActiveChannelContext);
  if (!ctx) {
    throw new Error("useActiveChannel used outside ActiveChannelProvider");
  }
  return ctx;
}

export function useSendControl() {
  const queryClient = useQueryClient();
  return useMutation<ControlAck, Error, ControlCommand>({
    mutationFn: async (cmd) => {
      try {
        return await sendWSControl(cmd);
      } catch {
        return sendControl(cmd);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: SNAPSHOT_KEY });
    },
  });
}
