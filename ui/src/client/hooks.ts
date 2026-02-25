import { useEffect } from "react";
import {
  useQuery,
  useQueryClient,
  useMutation,
  type QueryClient,
} from "@tanstack/react-query";
import { connectSSE, SNAPSHOT_KEY } from "./sse";
import { fetchSnapshot, fetchSessionEvents, sendControl } from "./api";
import type { Snapshot, EventLine, ControlCommand, ControlAck } from "./types";

// Connects SSE on mount, seeds cache with initial fetch, and keeps
// snapshot data live via server-sent events.
export function useSnapshot() {
  const queryClient = useQueryClient();

  useEffect(() => {
    return connectSSE(queryClient);
  }, [queryClient]);

  return useQuery<Snapshot>({
    queryKey: [...SNAPSHOT_KEY],
    queryFn: fetchSnapshot,
    // SSE handles updates; only fetch once for initial seed.
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
}

export function useSessionEvents(sessionId: string | undefined) {
  return useQuery<EventLine[]>({
    queryKey: ["sessionEvents", sessionId],
    queryFn: () => fetchSessionEvents(sessionId!),
    enabled: !!sessionId,
    refetchInterval: 3000,
  });
}

export function useSendControl() {
  const queryClient = useQueryClient();
  return useMutation<ControlAck, Error, ControlCommand>({
    mutationFn: sendControl,
    onSuccess: () => {
      // Invalidate snapshot so next SSE or refetch picks up changes.
      queryClient.invalidateQueries({ queryKey: [...SNAPSHOT_KEY] });
    },
  });
}
