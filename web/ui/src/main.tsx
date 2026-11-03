import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { RouterProvider, createRouter } from "@tanstack/react-router"

import { client } from "@/api/client.gen"
import { ThemeProvider } from "@/components/theme-provider"
import { TooltipProvider } from "@/components/ui/tooltip"
import { signInRedirect } from "@/lib/session"
import { routeTree } from "./routeTree.gen"
import "./index.css"

// One place for every API call, so no screen shows "unauthorized" as if it
// were its own error.
client.interceptors.response.use((response) => {
  const target = signInRedirect(response.status)
  if (target) {
    window.location.assign(target)
  }
  return response
})

const queryClient = new QueryClient()
const router = createRouter({ routeTree })

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>
          <RouterProvider router={router} />
        </TooltipProvider>
      </QueryClientProvider>
    </ThemeProvider>
  </StrictMode>,
)
