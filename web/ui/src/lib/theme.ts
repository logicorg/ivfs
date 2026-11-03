export const themes = ["light", "dark", "system"] as const

// Theme is the reader's choice; "system" follows the operating system.
export type Theme = (typeof themes)[number]

// themeStorageKey names the choice in localStorage. index.html reads it too,
// to paint the first frame in the right theme, so change both together.
export const themeStorageKey = "ivfs-theme"

// parseTheme reads a stored choice, falling back to "system" for nothing
// stored or a value an older or newer build left behind.
export function parseTheme(stored: string | null): Theme {
  return themes.find((theme) => theme === stored) ?? "system"
}

// resolveTheme is the palette a choice shows, given whether the operating
// system prefers dark.
export function resolveTheme(theme: Theme, prefersDark: boolean): "light" | "dark" {
  if (theme === "system") {
    return prefersDark ? "dark" : "light"
  }
  return theme
}
