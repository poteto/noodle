import { test, expect } from "@playwright/test";

test.describe("Noodle UI smoke", () => {
  test("board loads with columns", async ({ page }) => {
    await page.goto("/");

    // Header renders with the app name
    await expect(page.getByText("noodle")).toBeVisible();

    // Four kanban columns render with their exact title text
    for (const col of ["Queued", "Cooking", "Review", "Done"]) {
      await expect(page.getByText(col, { exact: true })).toBeVisible();
    }
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
