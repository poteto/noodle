import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useActiveChannel } from "./hooks";

describe("useActiveChannel", () => {
  it("defaults to scheduler channel", () => {
    const { result } = renderHook(() => useActiveChannel());
    expect(result.current.activeChannel).toEqual({ type: "scheduler" });
  });

  it("can switch to an agent channel", () => {
    const { result } = renderHook(() => useActiveChannel());
    act(() => {
      result.current.setActiveChannel({ type: "agent", sessionId: "sess-1" });
    });
    expect(result.current.activeChannel).toEqual({ type: "agent", sessionId: "sess-1" });
  });

  it("can switch back to scheduler", () => {
    const { result } = renderHook(() => useActiveChannel());
    act(() => {
      result.current.setActiveChannel({ type: "agent", sessionId: "sess-1" });
    });
    act(() => {
      result.current.setActiveChannel({ type: "scheduler" });
    });
    expect(result.current.activeChannel).toEqual({ type: "scheduler" });
  });
});
