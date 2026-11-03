import { describe, expect, it } from "vitest"

import type { Entry, Posting } from "@/api"

import { entryAmounts, type EntryAmount } from "./entry-amounts"

const posting = (lineNumber: number, side: Posting["side"], amount: string, currencyId = 1): Posting => ({
  lineNumber,
  ledgerAccountId: lineNumber,
  ledgerAccountName: `Account ${lineNumber}`,
  kind: "asset",
  currencyId,
  amount,
  side,
})

const entry = (...postings: Posting[]): Entry => ({ id: 1, occurredAt: "2026-01-31T10:00:00Z", status: "posted", postings })

describe("entryAmounts", () => {
  const cases: { name: string; entry: Entry; want: EntryAmount[] }[] = [
    { name: "an expense", entry: entry(posting(1, "debit", "12.5"), posting(2, "credit", "12.5")), want: [{ currencyId: 1, amount: "12.5" }] },
    {
      name: "a split across categories",
      entry: entry(posting(1, "debit", "10"), posting(2, "debit", "2.25"), posting(3, "credit", "12.25")),
      want: [{ currencyId: 1, amount: "12.25" }],
    },
    {
      name: "an exchange, one amount for each currency in the order they appear",
      entry: entry(posting(1, "credit", "100", 2), posting(2, "debit", "100", 2), posting(3, "debit", "92", 1), posting(4, "credit", "92", 1)),
      want: [
        { currencyId: 2, amount: "100" },
        { currencyId: 1, amount: "92" },
      ],
    },
  ]

  for (const { name, entry, want } of cases) {
    it(name, () => {
      expect(entryAmounts(entry)).toEqual(want)
    })
  }
})
