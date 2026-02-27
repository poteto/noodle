import { describe, expect, it } from "vitest";
import { formatCost, formatDuration, middleTruncate } from "./format";

describe("formatCost", () => {
  it("formats sub-penny amounts", () => {
    expect(formatCost(0)).toBe("<$0.01");
    expect(formatCost(0.005)).toBe("<$0.01");
    expect(formatCost(0.0099)).toBe("<$0.01");
  });

  it("formats normal amounts with two decimal places", () => {
    expect(formatCost(0.01)).toBe("$0.01");
    expect(formatCost(1.5)).toBe("$1.50");
    expect(formatCost(123.456)).toBe("$123.46");
  });
});

describe("formatDuration", () => {
  it("formats seconds only", () => {
    expect(formatDuration(0)).toBe("0s");
    expect(formatDuration(30)).toBe("30s");
    expect(formatDuration(59)).toBe("59s");
  });

  it("formats minutes and seconds", () => {
    expect(formatDuration(60)).toBe("1m");
    expect(formatDuration(90)).toBe("1m 30s");
    expect(formatDuration(3599)).toBe("59m 59s");
  });

  it("formats hours and minutes", () => {
    expect(formatDuration(3600)).toBe("1h 0m");
    expect(formatDuration(3661)).toBe("1h 1m");
    expect(formatDuration(7200)).toBe("2h 0m");
  });
});

describe("middleTruncate", () => {
  it("returns empty for zero maxWidth", () => {
    expect(middleTruncate("hello", 0)).toBe("");
  });

  it("returns original if within limit", () => {
    expect(middleTruncate("hello", 5)).toBe("hello");
    expect(middleTruncate("hi", 10)).toBe("hi");
  });

  it("truncates with ellipsis", () => {
    const result = middleTruncate("abcdefghij", 5);
    expect(result).toHaveLength(5);
    expect(result).toContain("\u2026");
  });

  it("returns just ellipsis for maxWidth 1", () => {
    expect(middleTruncate("hello", 1)).toBe("\u2026");
  });

  it("handles paths by preserving first and last segments", () => {
    const result = middleTruncate("src/components/deeply/nested/Component.tsx", 30);
    expect(result).toContain("src");
    expect(result).toContain("Component.tsx");
    expect(result.length).toBeLessThanOrEqual(30);
  });

  it("handles paths too short for segment preservation", () => {
    const result = middleTruncate("a/b/c/d/e/f/g", 5);
    expect(result).toHaveLength(5);
    expect(result).toContain("\u2026");
  });
});
