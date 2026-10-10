import { configDefaults, defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

// The build output is served by the product's own delivery (RUN.json `serve_dir`).
// No resource is fetched from a third party: fonts and scripts are bundled locally.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  test: {
    environment: "jsdom",
    globals: true,
    css: true,
    // The office's Playwright smoke test lives under e2e/ and loads
    // @playwright/test; it is not a unit test and must not be collected here.
    exclude: [...configDefaults.exclude, "e2e/**"],
  },
});
