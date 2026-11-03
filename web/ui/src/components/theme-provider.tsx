import { createContext, useContext, useEffect, useState, type ReactNode } from "react"

import { parseTheme, resolveTheme, themeStorageKey, type Theme } from "@/lib/theme"

const darkQuery = "(prefers-color-scheme: dark)"

const ThemeContext = createContext<{ theme: Theme; setTheme: (theme: Theme) => void } | null>(null)

// ThemeProvider keeps the .dark class on <html> in line with the reader's
// choice, and with the operating system while that choice is "system".
export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => parseTheme(readStored()))

  useEffect(() => {
    const media = window.matchMedia(darkQuery)
    const apply = () => {
      const dark = resolveTheme(theme, media.matches) === "dark"
      document.documentElement.classList.toggle("dark", dark)
    }
    apply()
    if (theme !== "system") {
      return
    }
    media.addEventListener("change", apply)
    return () => media.removeEventListener("change", apply)
  }, [theme])

  const setTheme = (next: Theme) => {
    try {
      localStorage.setItem(themeStorageKey, next)
    } catch {
      // Storage can be off, as in some private windows; the choice then lasts
      // until the page reloads.
    }
    setThemeState(next)
  }

  return <ThemeContext value={{ theme, setTheme }}>{children}</ThemeContext>
}

export function useTheme() {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error("useTheme needs a ThemeProvider above it; main.tsx renders one around the app.")
  }
  return context
}

function readStored(): string | null {
  try {
    return localStorage.getItem(themeStorageKey)
  } catch {
    return null
  }
}
