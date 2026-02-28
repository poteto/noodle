import { createElement, createContext, useContext, useEffect, useState } from "react";
import { useQuery, useSuspenseQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import { connectSSE, SNAPSHOT_KEY, SSE_STATUS_KEY } from "./sse";
import type { SSEStatus } from "./sse";
import { fetchSnapshot, fetchSessionEvents, sendControl, fetchReviewDiff } from "./api";
import type { Snapshot, EventLine, ControlCommand, ControlAck, DiffResponse, ChannelId } from "./types";

// Connects SSE on mount, seeds cache with initial fetch, and keeps
// snapshot data live via server-sent events.
export function useSnapshot() {
  const queryClient = useQueryClient();

  useEffect(() => connectSSE(queryClient), [queryClient]);

  return useQuery<Snapshot>({
    queryKey: SNAPSHOT_KEY,
    queryFn: fetchSnapshot,
    // SSE handles updates; only fetch once for initial seed.
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
}

// Suspense variant — throws until the initial snapshot arrives.
// The component calling this must be wrapped in a <Suspense> boundary.
export function useSuspenseSnapshot() {
  const queryClient = useQueryClient();

  useEffect(() => connectSSE(queryClient), [queryClient]);

  return useSuspenseQuery<Snapshot>({
    queryKey: SNAPSHOT_KEY,
    queryFn: fetchSnapshot,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
}

export function useSessionEvents(sessionId: string | undefined) {
  return useQuery<EventLine[]>({
    queryKey: ["sessionEvents", sessionId],
    queryFn: () => fetchSessionEvents(sessionId ?? ""),
    enabled: Boolean(sessionId),
    refetchInterval: 3000,
  });
}

export function useSSEStatus(): SSEStatus {
  const queryClient = useQueryClient();
  const { data } = useQuery<SSEStatus>({
    queryKey: SSE_STATUS_KEY,
    queryFn: () => queryClient.getQueryData(SSE_STATUS_KEY) ?? "connecting",
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
  return data ?? "connecting";
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

export function ActiveChannelProvider({ children }: { children: React.ReactNode }) {
  const [activeChannel, setActiveChannel] = useState<ChannelId>({ type: "scheduler" });
  return createElement(ActiveChannelContext.Provider, { value: { activeChannel, setActiveChannel } }, children);
}

export function useActiveChannel() {
  const ctx = useContext(ActiveChannelContext);
  if (!ctx) throw new Error("useActiveChannel used outside ActiveChannelProvider");
  return ctx;
}

export function useSendControl() {
  const queryClient = useQueryClient();
  return useMutation<ControlAck, Error, ControlCommand>({
    mutationFn: sendControl,
    onSuccess: () => {
      // Invalidate snapshot so next SSE or refetch picks up changes.
      queryClient.invalidateQueries({ queryKey: SNAPSHOT_KEY });
    },
  });
}
