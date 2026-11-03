import { defineConfig } from "@hey-api/openapi-ts"

// Generates the API client from the gateway's OpenAPI document (ADR 0007):
// its types, an SDK function per operation, zod schemas, and TanStack Query
// options. `make api` runs it after regenerating the document.
export default defineConfig({
  input: "../api/openapi.json",
  output: "src/api",
  plugins: ["@hey-api/client-fetch", "@hey-api/typescript", "@hey-api/sdk", "zod", "@tanstack/react-query"],
})
