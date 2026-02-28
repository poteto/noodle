import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { useActiveChannel, ActiveChannelProvider } from "./hooks";
import type { ChannelId } from "./types";

function createWrapper(channel: ChannelId, onChange = vi.fn()) {
  return ({ children }: { children: React.ReactNode }) =>
    ActiveChannelProvider({ channel, onChannelChange: onChange, children });
}

describe("useActiveChannel", () => {
  it("returns provided channel", () => {
    const wrapper = createWrapper({ type: "scheduler" });
    const { result } = renderHook(() => useActiveChannel(), { wrapper });
    expect(result.current.activeChannel).toEqual({ type: "scheduler" });
  });

  it("returns agent channel when provided", () => {
    const wrapper = createWrapper({ type: "agent", sessionId: "sess-1" });
    const { result } = renderHook(() => useActiveChannel(), { wrapper });
    expect(result.current.activeChannel).toEqual({ type: "agent", sessionId: "sess-1" });
  });

  it("calls onChannelChange when setActiveChannel is invoked", () => {
    const onChange = vi.fn();
    const wrapper = createWrapper({ type: "scheduler" }, onChange);
    const { result } = renderHook(() => useActiveChannel(), { wrapper });
    result.current.setActiveChannel({ type: "agent", sessionId: "sess-1" });
    expect(onChange).toHaveBeenCalledWith({ type: "agent", sessionId: "sess-1" });
  });
});
