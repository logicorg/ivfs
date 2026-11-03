import path from "node:path"

import tailwindcss from "@tailwindcss/vite"
import { tanstackRouter } from "@tanstack/router-plugin/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// The gateway `make web` runs. During development Vite serves the UI and
// forwards /api and /auth to it. The proxy keeps the browser's Host and
// Origin, so the gateway must be told it is served from Vite
// (IVFS_PUBLIC_URL=http://localhost:5173): its guard then accepts them, and
// the provider sends the browser back here after sign-in (ADR 0013).
const gateway = "http://127.0.0.1:8080"

export default defineConfig({
  plugins: [tanstackRouter({ target: "react", autoCodeSplitting: true }), react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(import.meta.dirname, "src") },
  },
  build: {
    // The gateway embeds dist, where a committed placeholder keeps go build
    // working before the UI is built, so the build goes one level down.
    outDir: "dist/app",
    emptyOutDir: true,
  },
  server: {
    // Spelled out because the string shorthand sets changeOrigin, which
    // would swap the Host for the gateway's.
    proxy: {
      "/api": { target: gateway },
      "/auth": { target: gateway },
    },
  },
})
