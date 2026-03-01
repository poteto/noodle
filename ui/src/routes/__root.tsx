import { Suspense, useMemo, useCallback } from "react";
import { createRootRoute, Outlet, useLocation, useNavigate } from "@tanstack/react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ActiveChannelProvider } from "~/client";
import type { ChannelId } from "~/client";
import { Sidebar } from "~/components/Sidebar";
import "~/app.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: Infinity,
      refetchOnWindowFocus: false,
    },
  },
});

function RootComponent() {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const isOrchestratorRoute = pathname === "/orchestrator";

  const activeChannel: ChannelId = useMemo(() => {
    if (pathname.startsWith("/actor/")) {
      return { type: "agent", sessionId: pathname.slice("/actor/".length) };
    }
    return { type: "scheduler" };
  }, [pathname]);

  const setActiveChannel = useCallback(
    (channel: ChannelId) => {
      if (channel.type === "scheduler") {
        navigate({ to: "/" });
      } else {
        navigate({ to: "/actor/$id", params: { id: channel.sessionId } });
      }
    },
    [navigate],
  );

  return (
    <ActiveChannelProvider channel={activeChannel} onChannelChange={setActiveChannel}>
      <Suspense fallback={<div className="h-screen bg-bg-depth" />}>
        {isOrchestratorRoute ? (
          <Outlet />
        ) : (
          <div className="app-layout h-screen">
            <Sidebar />
            <Outlet />
          </div>
        )}
      </Suspense>
    </ActiveChannelProvider>
  );
}

export const Route = createRootRoute({
  component: () => (
    <QueryClientProvider client={queryClient}>
      <RootComponent />
    </QueryClientProvider>
  ),
});
