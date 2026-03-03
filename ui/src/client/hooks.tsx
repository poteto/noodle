import { createElement, createContext, useContext, useEffect, useSyncExternalStore } from "react";
import { useQuery, useSuspenseQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import {
  SNAPSHOT_KEY,
  subscribeSession,
  unsubscribeSession,
  sendWSControl,
  subscribeWSStatus,
  getWSStatus,
} from "./ws";
import type { WSStatus } from "./ws";
import { fetchSnapshot, fetchSessionEvents, sendControl, fetchReviewDiff } from "./api";
import { chooseNewerSnapshot } from "./snapshot-freshness";
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

  return useQuery<Snapshot>({
    queryKey: SNAPSHOT_KEY,
    queryFn: async ({ signal }) => {
      const incoming = await fetchSnapshot(signal);
      const current = queryClient.getQueryData<Snapshot>(SNAPSHOT_KEY);
      return chooseNewerSnapshot(current, incoming);
    },
    // WS handles updates; only fetch once for initial seed.
    staleTime: Infinity,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
}

// Suspense variant -- throws until the initial snapshot arrives.
// The component calling this must be wrapped in a <Suspense> boundary.
export function useSuspenseSnapshot() {
  const queryClient = useQueryClient();

  return useSuspenseQuery<Snapshot>({
    queryKey: SNAPSHOT_KEY,
    queryFn: async ({ signal }) => {
      const incoming = await fetchSnapshot(signal);
      const current = queryClient.getQueryData<Snapshot>(SNAPSHOT_KEY);
      return chooseNewerSnapshot(current, incoming);
    },
    staleTime: Infinity,
    refetchOnMount: false,
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
    queryFn: ({ signal }) => (sessionId ? fetchSessionEvents(sessionId, undefined, signal) : []),
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
      const cmdWithID: ControlCommand = {
        ...cmd,
        id: cmd.id ?? `ui-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      };
      try {
        return await sendWSControl(cmdWithID);
      } catch {
        return sendControl(cmdWithID);
      }
    },
    onSuccess: () => {
      // When WS is connected, snapshot pushes are authoritative and arrive
      // after control processing. Invalidating here can race and overwrite a
      // newer WS snapshot with an older HTTP response.
      if (getWSStatus() !== "connected") {
        queryClient.invalidateQueries({ queryKey: SNAPSHOT_KEY });
      }
    },
  });
}
