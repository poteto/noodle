import { describe, expect, it, vi, beforeEach } from "vitest";
import { fetchSnapshot, fetchConfig, fetchReviewDiff, sendControl } from "./api";

const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

beforeEach(() => {
  mockFetch.mockReset();
});

function jsonResponse(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("fetchSnapshot", () => {
  it("fetches and returns snapshot data", async () => {
    const data = { loop_state: "running", sessions: [] };
    mockFetch.mockResolvedValue(jsonResponse(data));

    const result = await fetchSnapshot();
    // normalizeSnapshot fills in missing array fields with defaults.
    expect(result).toMatchObject(data);
    expect(mockFetch).toHaveBeenCalledWith("/api/snapshot");
  });

  it("throws on non-OK response", async () => {
    mockFetch.mockResolvedValue(jsonResponse({}, 500));
    await expect(fetchSnapshot()).rejects.toThrow("fetchSnapshot: 500");
  });
});

describe("fetchConfig", () => {
  it("fetches and returns config", async () => {
    const data = { provider: "claude", model: "opus", mode: "supervised", task_types: [] };
    mockFetch.mockResolvedValue(jsonResponse(data));

    const result = await fetchConfig();
    expect(result).toEqual(data);
    expect(mockFetch).toHaveBeenCalledWith("/api/config");
  });

  it("throws on non-OK response", async () => {
    mockFetch.mockResolvedValue(jsonResponse({}, 403));
    await expect(fetchConfig()).rejects.toThrow("fetchConfig: 403");
  });
});

describe("fetchReviewDiff", () => {
  it("fetches diff for a review", async () => {
    const data = { diff: "--- a/file\n+++ b/file", stat: "1 file changed" };
    mockFetch.mockResolvedValue(jsonResponse(data));

    const result = await fetchReviewDiff("order-1");
    expect(result).toEqual(data);
    expect(mockFetch).toHaveBeenCalledWith("/api/reviews/order-1/diff");
  });

  it("encodes the review id", async () => {
    mockFetch.mockResolvedValue(jsonResponse({ diff: "", stat: "" }));
    await fetchReviewDiff("order/1");
    expect(mockFetch).toHaveBeenCalledWith("/api/reviews/order%2F1/diff");
  });
});

describe("sendControl", () => {
  it("sends a control command via POST", async () => {
    const ack = { id: "1", action: "pause", status: "ok", at: "2025-01-01T00:00:00Z" };
    mockFetch.mockResolvedValue(jsonResponse(ack));

    const result = await sendControl({ action: "pause" });
    expect(result).toEqual(ack);
    expect(mockFetch).toHaveBeenCalledWith("/api/control", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "pause" }),
    });
  });

  it("throws on non-OK response", async () => {
    mockFetch.mockResolvedValue(jsonResponse({}, 400));
    await expect(sendControl({ action: "pause" })).rejects.toThrow("sendControl: 400");
  });
});
