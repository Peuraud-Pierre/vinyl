/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";
import { resolve } from "node:path";

export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  // Le plugin Wails exige des bindings générés : inutile (et bloquant) sous Vitest.
  plugins: [tailwindcss(), ...(process.env.VITEST ? [] : [wails("./bindings")])],
  build: {
    rollupOptions: {
      input: {
        main: resolve(import.meta.dirname, "index.html"),
        widget: resolve(import.meta.dirname, "widget.html"),
      },
    },
  },
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.ts"],
  },
});
