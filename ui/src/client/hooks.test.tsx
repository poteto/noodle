import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ActiveChannelProvider, useActiveChannel, useSendControl } from "./hooks";
import type { ChannelId, ControlAck } from "./types";

const mockSendWSControl = vi.fn();
const mockSendControl = vi.fn();
let wsStatus: "connected" | "connecting" | "disconnected" = "connected";

vi.mock("./ws", () => ({
  SNAPSHOT_KEY: ["snapshot"] as const,
  connectWS: vi.fn(),
  subscribeSession: vi.fn(),
  unsubscribeSession: vi.fn(),
  sendWSControl: (...args: unknown[]) => mockSendWSControl(...args),
  subscribeWSStatus: vi.fn(() => vi.fn()),
  getWSStatus: () => wsStatus,
}));

vi.mock("./api", async () => {
  const actual = await vi.importActual("./api");
  return {
    ...actual,
    sendControl: (...args: unknown[]) => mockSendControl(...args),
  };
});

function makeWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function makeChannelWrapper(channel: ChannelId, onChange = vi.fn()) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <ActiveChannelProvider channel={channel} onChannelChange={onChange}>
        {children}
      </ActiveChannelProvider>
    );
  };
}

function okAck(action: ControlAck["action"]): ControlAck {
  return {
    id: "ack-1",
    action,
    status: "ok",
    at: new Date().toISOString(),
  };
}

describe("useSendControl", () => {
  beforeEach(() => {
    wsStatus = "connected";
    mockSendWSControl.mockReset();
    mockSendControl.mockReset();
  });

  it("does not invalidate snapshot when websocket is connected", async () => {
    mockSendWSControl.mockResolvedValue(okAck("pause"));

    const queryClient = new QueryClient();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const { result } = renderHook(() => useSendControl(), {
      wrapper: makeWrapper(queryClient),
    });

    await result.current.mutateAsync({ action: "pause" });

    expect(mockSendWSControl).toHaveBeenCalledTimes(1);
    expect(invalidateSpy).not.toHaveBeenCalled();
  });

  it("invalidates snapshot when websocket is not connected", async () => {
    wsStatus = "disconnected";
    mockSendWSControl.mockResolvedValue(okAck("pause"));

    const queryClient = new QueryClient();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const { result } = renderHook(() => useSendControl(), {
      wrapper: makeWrapper(queryClient),
    });

    await result.current.mutateAsync({ action: "pause" });

    expect(mockSendWSControl).toHaveBeenCalledTimes(1);
    expect(invalidateSpy).toHaveBeenCalledTimes(1);
  });
});

describe("useActiveChannel", () => {
  it("returns provided channel", () => {
    const wrapper = makeChannelWrapper({ type: "scheduler" });
    const { result } = renderHook(() => useActiveChannel(), { wrapper });
    expect(result.current.activeChannel).toEqual({ type: "scheduler" });
  });

  it("returns agent channel when provided", () => {
    const wrapper = makeChannelWrapper({ type: "agent", sessionId: "sess-1" });
    const { result } = renderHook(() => useActiveChannel(), { wrapper });
    expect(result.current.activeChannel).toEqual({ type: "agent", sessionId: "sess-1" });
  });

  it("calls onChannelChange when setActiveChannel is invoked", () => {
    const onChange = vi.fn();
    const wrapper = makeChannelWrapper({ type: "scheduler" }, onChange);
    const { result } = renderHook(() => useActiveChannel(), { wrapper });
    result.current.setActiveChannel({ type: "agent", sessionId: "sess-1" });
    expect(onChange).toHaveBeenCalledWith({ type: "agent", sessionId: "sess-1" });
  });
});
