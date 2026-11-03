import { describe, expect, it } from "vitest"

import { parseTheme, resolveTheme, type Theme } from "./theme"

describe("parseTheme", () => {
  const cases: { name: string; stored: string | null; want: Theme }[] = [
    { name: "nothing stored", stored: null, want: "system" },
    { name: "light", stored: "light", want: "light" },
    { name: "dark", stored: "dark", want: "dark" },
    { name: "system", stored: "system", want: "system" },
    { name: "an unknown value", stored: "sepia", want: "system" },
    { name: "a value in the wrong case", stored: "Dark", want: "system" },
  ]

  for (const { name, stored, want } of cases) {
    it(name, () => {
      expect(parseTheme(stored)).toBe(want)
    })
  }
})

describe("resolveTheme", () => {
  const cases: { name: string; theme: Theme; prefersDark: boolean; want: "light" | "dark" }[] = [
    { name: "light ignores a dark system", theme: "light", prefersDark: true, want: "light" },
    { name: "dark ignores a light system", theme: "dark", prefersDark: false, want: "dark" },
    { name: "system on a dark system", theme: "system", prefersDark: true, want: "dark" },
    { name: "system on a light system", theme: "system", prefersDark: false, want: "light" },
  ]

  for (const { name, theme, prefersDark, want } of cases) {
    it(name, () => {
      expect(resolveTheme(theme, prefersDark)).toBe(want)
    })
  }
})
