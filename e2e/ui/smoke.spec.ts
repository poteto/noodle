import { test, expect, type APIRequestContext } from "@playwright/test";

async function pollSnapshotForActiveWorktreeAgent(
  request: APIRequestContext,
): Promise<{ id: string }> {
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    const res = await request.get("/api/snapshot");
    if (!res.ok()) {
      await new Promise((resolve) => setTimeout(resolve, 500));
      continue;
    }
    const snapshot = await res.json();
    const sessions = Array.isArray(snapshot?.active) ? snapshot.active : [];
    const match = sessions.find((session: unknown) => {
      const record = (session ?? {}) as Record<string, unknown>;
      const taskKey = String(record.task_key ?? "").trim().toLowerCase();
      const worktree = String(record.worktree_name ?? "").trim();
      const id = String(record.id ?? "").trim();
      return taskKey !== "schedule" && worktree !== "" && id !== "";
    }) as Record<string, unknown> | undefined;
    if (match?.id) {
      return { id: String(match.id) };
    }
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw new Error("no active non-schedule session with worktree_name found in snapshot within 30s");
}

test.describe("Noodle UI smoke", () => {
  test("channel layout loads with sidebar, feed, and context panel", async ({ page }) => {
    await page.goto("/");
    const sidebar = page.locator("aside.sidebar");
    await expect(sidebar).toBeVisible();

    // NOODLE header visible in sidebar
    await expect(sidebar.getByText("NOODLE", { exact: true })).toBeVisible();

    // Nav links visible
    await expect(sidebar.getByRole("link", { name: "Dashboard" })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: "Live Feed" })).toBeVisible();
    await expect(sidebar.getByRole("link", { name: "Topology" })).toBeVisible();

    // Agents section visible
    await expect(sidebar.getByText("Agents")).toBeVisible();

    // Three-column layout present (grid with sidebar, feed, context)
    const grid = page.locator(".grid");
    await expect(grid).toBeVisible();
  });

  test("sidebar shows scheduler channel", async ({ page }) => {
    await page.goto("/");
    const sidebar = page.locator("aside.sidebar");

    // Scheduler channel item exists under Agents
    await expect(sidebar.getByText("Scheduler", { exact: true })).toBeVisible();
    await expect(sidebar.locator(".agent-meta-line")).toContainText(/IDLE|MONITORING THE SITUATION/);
  });

  test("dashboard route loads", async ({ page }) => {
    await page.goto("/dashboard");

    // Dashboard header and stats bar render
    await expect(page.locator(".feed-title")).toContainText("Dashboard");
    await expect(page.getByTestId("stats-bar")).toBeVisible();
  });

  test("topology route loads", async ({ page }) => {
    await page.goto("/topology");

    // Tree visualization SVG renders (the large one, not nav icons)
    const svg = page.locator("svg.w-full");
    await expect(svg).toBeVisible();
  });

  test("steer input is functional", async ({ page }) => {
    await page.goto("/");

    // Textarea and SEND button exist in the feed area
    const textarea = page.locator("textarea");
    await expect(textarea).toBeVisible();

    const sendButton = page.getByRole("button", { name: "Send" });
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

  test("WebSocket endpoint is reachable", async ({ page, baseURL }) => {
    await page.goto("/");
    // Verify we can open a WebSocket connection and receive the initial snapshot.
    const received = await page.evaluate(async (url) => {
      const wsURL = url!.replace(/^http/, "ws") + "/api/ws";
      return new Promise<boolean>((resolve) => {
        const ws = new WebSocket(wsURL);
        const timeout = setTimeout(() => {
          ws.close();
          resolve(false);
        }, 5000);
        ws.addEventListener("message", (event) => {
          clearTimeout(timeout);
          ws.close();
          try {
            const msg = JSON.parse(event.data);
            resolve(msg.type === "snapshot");
          } catch {
            resolve(false);
          }
        });
        ws.addEventListener("error", () => {
          clearTimeout(timeout);
          resolve(false);
        });
      });
    }, baseURL);
    expect(received).toBe(true);
  });

  test("active agent context shows worktree name", async ({ page, request }) => {
    const agent = await pollSnapshotForActiveWorktreeAgent(request);
    await page.goto(`/actor/${agent.id}`);

    await expect(page.getByText("Worktree")).toBeVisible();
    await expect(page.getByText("Not available", { exact: true })).not.toBeVisible();
  });
});
