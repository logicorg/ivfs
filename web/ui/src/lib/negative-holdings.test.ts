import { describe, expect, it } from "vitest"

import type { NegativeHolding } from "@/api"

import { describeNegativeHolding } from "./negative-holdings"

describe("describeNegativeHolding", () => {
  const warning: NegativeHolding = {
    financialAccountId: 3,
    financialAccountName: "IBKR",
    assetId: 12,
    assetName: "ACME",
    at: "2026-03-03T10:00:00Z",
    units: "-90",
  }
  const cases: { name: string; warning: NegativeHolding; timeZone: string; want: string }[] = [
    { name: "a whole number of units", warning, timeZone: "UTC", want: "ACME in IBKR is -90 units on Mar 3, 2026" },
    {
      name: "a fraction of a unit, kept exact",
      warning: { ...warning, units: "-0.000123" },
      timeZone: "UTC",
      want: "ACME in IBKR is -0.000123 units on Mar 3, 2026",
    },
    {
      name: "the date in the reader's time zone, not UTC's",
      warning: { ...warning, at: "2026-03-03T23:30:00Z" },
      timeZone: "Europe/Rome",
      want: "ACME in IBKR is -90 units on Mar 4, 2026",
    },
  ]

  for (const { name, warning, timeZone, want } of cases) {
    it(name, () => {
      expect(describeNegativeHolding(warning, "en-US", timeZone)).toBe(want)
    })
  }
})
