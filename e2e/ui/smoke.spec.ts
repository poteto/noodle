import { test, expect } from "@playwright/test";

test.describe("Noodle UI smoke", () => {
  test("channel layout loads with sidebar, feed, and context panel", async ({ page }) => {
    await page.goto("/");

    // NOODLE header visible in sidebar
    await expect(page.getByText("NOODLE")).toBeVisible();

    // Nav links visible
    await expect(page.getByText("DASHBOARD")).toBeVisible();
    await expect(page.getByText("LIVE FEED")).toBeVisible();
    await expect(page.getByText("TREE")).toBeVisible();

    // SCHEDULER section visible
    await expect(page.getByText("SCHEDULER").first()).toBeVisible();

    // Three-column layout present (grid with sidebar, feed, context)
    const grid = page.locator(".grid");
    await expect(grid).toBeVisible();
  });

  test("sidebar shows scheduler channel", async ({ page }) => {
    await page.goto("/");

    // Manager channel item exists under SCHEDULER
    await expect(page.getByText("Manager")).toBeVisible();
    await expect(page.getByText("LLM")).toBeVisible();
  });

  test("dashboard route loads", async ({ page }) => {
    await page.goto("/dashboard");

    // Dashboard header and stats bar render
    await expect(page.getByText("DASHBOARD")).toBeVisible();
    await expect(page.getByTestId("stats-bar")).toBeVisible();
  });

  test("tree route loads", async ({ page }) => {
    await page.goto("/tree");

    // SVG container renders
    const svg = page.locator("svg");
    await expect(svg).toBeVisible();
  });

  test("steer input is functional", async ({ page }) => {
    await page.goto("/");

    // Textarea and SEND button exist in the feed area
    const textarea = page.locator("textarea");
    await expect(textarea).toBeVisible();

    const sendButton = page.getByRole("button", { name: "SEND" });
    await expect(sendButton).toBeVisible();
  });

  test("snapshot API returns valid data", async ({ request }) => {
    const res = await request.get("/api/snapshot");
    expect(res.ok()).toBeTruthy();
    const snapshot = await res.json();
    expect(snapshot).toHaveProperty("loop_state");
    expect(snapshot).toHaveProperty("orders");
    expect(snapshot).toHaveProperty("active_order_ids");
    expect(snapshot).toHaveProperty("total_cost_usd");
    expect(Array.isArray(snapshot.orders)).toBe(true);
  });

  test("config API returns provider and model", async ({ request }) => {
    const res = await request.get("/api/config");
    expect(res.ok()).toBeTruthy();
    const config = await res.json();
    expect(config).toHaveProperty("provider");
    expect(config).toHaveProperty("model");
    expect(config).toHaveProperty("task_types");
  });

  test("events SSE endpoint is reachable", async ({ page, baseURL }) => {
    // Navigate to the app first so we have a valid page context
    await page.goto("/");
    // Use page.evaluate with an AbortController so we can verify the SSE
    // endpoint responds with the right content-type without waiting for
    // the (infinite) stream to complete.
    const contentType = await page.evaluate(async (url) => {
      const controller = new AbortController();
      const res = await fetch(url, {
        headers: { Accept: "text/event-stream" },
        signal: controller.signal,
      });
      const ct = res.headers.get("content-type");
      controller.abort();
      return ct;
    }, `${baseURL}/api/events`);
    expect(contentType).toContain("text/event-stream");
  });
});
