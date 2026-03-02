import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { VirtualizedFeed } from "./VirtualizedFeed";
import type { EventLine } from "~/client";

vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: ({
    count,
    getItemKey,
  }: {
    count: number;
    getItemKey: (index: number) => string;
  }) => ({
    getVirtualItems: () =>
      Array.from({ length: count }, (_, index) => ({
        index,
        key: getItemKey(index),
        start: index * 40,
      })),
    getTotalSize: () => count * 40,
    measureElement: () => null,
  }),
}));

class ResizeObserverMock {
  static instances: ResizeObserverMock[] = [];
  private connected = true;
  private readonly cb: ResizeObserverCallback;

  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn(() => {
    this.connected = false;
  });

  constructor(cb: ResizeObserverCallback) {
    this.cb = cb;
    ResizeObserverMock.instances.push(this);
  }

  trigger() {
    if (!this.connected) {
      return;
    }
    this.cb([], this as unknown as ResizeObserver);
  }
}

function buildEvent(body: string): EventLine {
  return {
    at: "2026-03-02T20:12:00Z",
    label: "Think",
    body,
    category: "think",
  };
}

function setScrollMetrics(
  el: HTMLDivElement,
  metrics: { scrollHeight: number; clientHeight: number },
) {
  const { scrollHeight: initialScrollHeight } = metrics;
  let scrollTop = initialScrollHeight;
  let scrollHeight = initialScrollHeight;

  Object.defineProperty(el, "scrollTop", {
    configurable: true,
    get: () => scrollTop,
    set: (value: number) => {
      scrollTop = value;
    },
  });
  Object.defineProperty(el, "scrollHeight", {
    configurable: true,
    get: () => scrollHeight,
  });
  Object.defineProperty(el, "clientHeight", {
    configurable: true,
    get: () => metrics.clientHeight,
  });

  return {
    get scrollTop() {
      return scrollTop;
    },
    setScrollTop(value: number) {
      scrollTop = value;
    },
    setScrollHeight(value: number) {
      scrollHeight = value;
    },
  };
}

describe("VirtualizedFeed", () => {
  beforeEach(() => {
    ResizeObserverMock.instances = [];
    vi.stubGlobal("ResizeObserver", ResizeObserverMock);
  });

  it("keeps bottom pin on content resize when auto-scroll is enabled", async () => {
    render(<VirtualizedFeed items={[buildEvent("initial")]} tail={<div>tail</div>} />);

    const scroller = document.querySelector(".feed-content") as HTMLDivElement;
    const metrics = setScrollMetrics(scroller, { scrollHeight: 500, clientHeight: 300 });
    metrics.setScrollTop(500);

    await waitFor(() => {
      expect(ResizeObserverMock.instances.length).toBeGreaterThan(0);
    });

    metrics.setScrollHeight(850);
    for (const instance of ResizeObserverMock.instances) {
      instance.trigger();
    }

    expect(metrics.scrollTop).toBe(850);
  });

  it("does not force-scroll after user scrolls away from bottom", async () => {
    render(<VirtualizedFeed items={[buildEvent("initial")]} tail={<div>tail</div>} />);

    const scroller = document.querySelector(".feed-content") as HTMLDivElement;
    const metrics = setScrollMetrics(scroller, { scrollHeight: 900, clientHeight: 300 });

    await waitFor(() => {
      expect(ResizeObserverMock.instances.length).toBeGreaterThan(0);
    });

    metrics.setScrollTop(120);
    fireEvent.scroll(scroller);

    await waitFor(() => {
      expect(screen.getByText("Scroll to bottom")).toBeInTheDocument();
    });

    metrics.setScrollHeight(1100);
    for (const instance of ResizeObserverMock.instances) {
      instance.trigger();
    }

    expect(metrics.scrollTop).toBe(120);
  });
});
