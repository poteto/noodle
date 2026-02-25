import { useQuery } from "@tanstack/react-query";
import { fetchConfig } from "./api";
import type { ConfigDefaults } from "./types";

export function useConfig() {
  return useQuery<ConfigDefaults>({
    queryKey: ["config"],
    queryFn: fetchConfig,
    staleTime: 60_000,
  });
}
