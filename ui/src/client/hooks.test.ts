import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useActiveChannel, ActiveChannelProvider } from "./hooks";

const wrapper = ({ children }: { children: React.ReactNode }) =>
  ActiveChannelProvider({ children });

describe("useActiveChannel", () => {
  it("defaults to scheduler channel", () => {
    const { result } = renderHook(() => useActiveChannel(), { wrapper });
    expect(result.current.activeChannel).toEqual({ type: "scheduler" });
  });

  it("can switch to an agent channel", () => {
    const { result } = renderHook(() => useActiveChannel(), { wrapper });
    act(() => {
      result.current.setActiveChannel({ type: "agent", sessionId: "sess-1" });
    });
    expect(result.current.activeChannel).toEqual({ type: "agent", sessionId: "sess-1" });
  });

  it("can switch back to scheduler", () => {
    const { result } = renderHook(() => useActiveChannel(), { wrapper });
    act(() => {
      result.current.setActiveChannel({ type: "agent", sessionId: "sess-1" });
    });
    act(() => {
      result.current.setActiveChannel({ type: "scheduler" });
    });
    expect(result.current.activeChannel).toEqual({ type: "scheduler" });
  });
});
