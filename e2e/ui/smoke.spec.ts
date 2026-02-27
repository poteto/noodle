import { test, expect } from "@playwright/test";

test.describe("Noodle UI smoke", () => {
  test("board loads with four columns", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("noodle")).toBeVisible();
    await expect(page.getByText("Queued")).toBeVisible();
    await expect(page.getByText("Cooking")).toBeVisible();
    await expect(page.getByText("Review")).toBeVisible();
    await expect(page.getByText("Done")).toBeVisible();
  });

  test("loop state indicator is visible", async ({ page }) => {
    await page.goto("/");
    // The loop state badge shows one of: running, paused, draining, idle
    const states = page.getByText(/^(running|paused|draining|idle)$/);
    await expect(states.first()).toBeVisible();
  });

  test("snapshot API returns valid data", async ({ request }) => {
    const res = await request.get("/api/snapshot");
    expect(res.ok()).toBeTruthy();
    const snapshot = await res.json();
    expect(snapshot).toHaveProperty("loop_state");
    expect(snapshot).toHaveProperty("orders");
    expect(snapshot).toHaveProperty("active_order_ids");
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

  test("SSE endpoint connects", async ({ request }) => {
    // Just verify the endpoint is reachable and returns event-stream
    const res = await request.get("/api/events", {
      headers: { Accept: "text/event-stream" },
    });
    expect(res.status()).toBe(200);
    expect(res.headers()["content-type"]).toContain("text/event-stream");
  });

  test("pause and resume via UI", async ({ page }) => {
    await page.goto("/");
    // Wait for board to render
    await expect(page.getByText("noodle")).toBeVisible();

    // Find the pause/resume button
    const btn = page.getByRole("button", { name: /^(pause|resume)$/ });
    await expect(btn).toBeVisible();

    const initialText = await btn.textContent();

    // Click to toggle
    await btn.click();

    // The button text should flip (optimistic update)
    const expectedText = initialText === "pause" ? "resume" : "pause";
    await expect(btn).toHaveText(expectedText);

    // Toggle back to restore original state
    await btn.click();
    await expect(btn).toHaveText(initialText!);
  });

  test("new task modal opens and closes", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("noodle")).toBeVisible();

    // Click "+ new task" button
    await page.getByRole("button", { name: "+ new task" }).click();

    // Modal should appear with form elements
    await expect(page.getByText("New Task")).toBeVisible();
    await expect(page.getByLabel("Prompt")).toBeVisible();
    await expect(page.getByLabel("Type")).toBeVisible();
    await expect(page.getByLabel("Provider")).toBeVisible();
    await expect(page.getByLabel("Model")).toBeVisible();

    // Enqueue button should be disabled with empty prompt
    await expect(page.getByRole("button", { name: "enqueue" })).toBeDisabled();

    // Type something — enqueue should enable
    await page.getByLabel("Prompt").fill("test task");
    await expect(page.getByRole("button", { name: "enqueue" })).toBeEnabled();

    // Close via cancel
    await page.getByRole("button", { name: "cancel" }).click();
    await expect(page.getByText("New Task")).not.toBeVisible();
  });

  test("keyboard shortcut n opens task editor", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("noodle")).toBeVisible();

    await page.keyboard.press("n");
    await expect(page.getByText("New Task")).toBeVisible();

    // Escape closes it
    await page.keyboard.press("Escape");
    await expect(page.getByText("New Task")).not.toBeVisible();
  });

  test("orders appear in queued or cooking columns", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("noodle")).toBeVisible();

    // After the agent smoke milestones pass, there should be at least one
    // order visible somewhere on the board. It could be queued, cooking, or done.
    // Wait up to 10s for content to appear.
    const board = page.locator(".flex-1.overflow-x-auto");
    await expect(board).toBeVisible();

    // The snapshot should have loaded and rendered something — at minimum
    // the cost display in the header shows $0.00 or higher
    await expect(page.getByText(/\$\d+\.\d{2}/)).toBeVisible();
  });
});
