import { defineConfig, mergeConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import viteTsConfigPaths from "vite-tsconfig-paths";

// We intentionally do NOT extend the main vite.config.ts because it includes
// TanStackRouterVite which scans src/routes/ and causes noisy output in tests.
// Instead we recreate the minimal plugin set needed for tests.
export default defineConfig({
  plugins: [react(), viteTsConfigPaths()],
  test: {
    environment: "jsdom",
    setupFiles: ["src/test-setup.ts"],
    include: ["src/**/*.test.{ts,tsx}"],
  },
});
