import { defineConfig } from "@playwright/test";

// NOODLE_BASE_URL is set by the Go test harness to point at the running
// noodle server (e.g. "http://127.0.0.1:3000").
const baseURL = process.env.NOODLE_BASE_URL ?? "http://127.0.0.1:3000";

export default defineConfig({
  testDir: ".",
  testMatch: "*.spec.ts",
  timeout: 30_000,
  retries: 1,
  use: {
    baseURL,
    headless: true,
  },
  projects: [{ name: "chromium", use: { browserName: "chromium" } }],
  reporter: [["list"]],
});
