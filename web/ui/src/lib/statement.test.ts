import { describe, expect, it } from "vitest"

import type { Balance, Entry, Posting } from "@/api"

import { signedAmount, statementLines, type StatementLine } from "./statement"

// posting is a posting of amount on the ledger account id, named after it.
function posting(lineNumber: number, id: number, name: string, side: Posting["side"], amount: string): Posting {
  return { lineNumber, ledgerAccountId: id, ledgerAccountName: name, kind: "asset", currencyId: 1, side, amount }
}

describe("statementLines", () => {
  const shopping: Entry = {
    id: 7,
    occurredAt: "2026-01-31T10:00:00Z",
    description: "Supermarket",
    status: "posted",
    postings: [posting(1, 200, "Groceries", "debit", "12.50"), posting(2, 100, "Cash", "credit", "12.50")],
  }
  const reversal: Entry = {
    id: 8,
    occurredAt: "2026-01-31T11:00:00Z",
    status: "posted",
    reversalOfEntryId: 7,
    postings: [posting(1, 200, "Groceries", "credit", "12.50"), posting(2, 100, "Cash", "debit", "12.50")],
  }
  const split: Entry = {
    id: 9,
    occurredAt: "2026-02-01T10:00:00Z",
    description: "Two cards, one shop",
    status: "posted",
    postings: [
      posting(1, 200, "Groceries", "debit", "30"),
      posting(2, 100, "Cash", "credit", "10"),
      posting(3, 100, "Cash", "credit", "20"),
    ],
  }

  const cases: { name: string; entries: Entry[]; id: number; kind: Balance["kind"]; want: StatementLine[] }[] = [
    {
      name: "a payment and its reversal, seen from the asset that paid",
      entries: [reversal, shopping],
      id: 100,
      kind: "asset",
      want: [
        {
          key: "8-2",
          entryId: 8,
          occurredAt: "2026-01-31T11:00:00Z",
          description: undefined,
          reversalOfEntryId: 7,
          otherAccounts: ["Groceries"],
          amount: "12.50",
        },
        {
          key: "7-2",
          entryId: 7,
          occurredAt: "2026-01-31T10:00:00Z",
          description: "Supermarket",
          reversalOfEntryId: undefined,
          otherAccounts: ["Groceries"],
          amount: "-12.50",
        },
      ],
    },
    {
      name: "the same payment, seen from the expense",
      entries: [shopping],
      id: 200,
      kind: "expense",
      want: [
        {
          key: "7-1",
          entryId: 7,
          occurredAt: "2026-01-31T10:00:00Z",
          description: "Supermarket",
          reversalOfEntryId: undefined,
          otherAccounts: ["Cash"],
          amount: "12.50",
        },
      ],
    },
    {
      name: "an entry that posts to the account twice",
      entries: [split],
      id: 100,
      kind: "asset",
      want: [
        {
          key: "9-2",
          entryId: 9,
          occurredAt: "2026-02-01T10:00:00Z",
          description: "Two cards, one shop",
          reversalOfEntryId: undefined,
          otherAccounts: ["Groceries"],
          amount: "-10",
        },
        {
          key: "9-3",
          entryId: 9,
          occurredAt: "2026-02-01T10:00:00Z",
          description: "Two cards, one shop",
          reversalOfEntryId: undefined,
          otherAccounts: ["Groceries"],
          amount: "-20",
        },
      ],
    },
    { name: "no entries", entries: [], id: 100, kind: "asset", want: [] },
  ]

  for (const { name, entries, id, kind, want } of cases) {
    it(name, () => {
      expect(statementLines(entries, id, kind)).toEqual(want)
    })
  }
})

describe("signedAmount", () => {
  const cases: { name: string; side: Posting["side"]; amount: string; kind: Balance["kind"]; want: string }[] = [
    { name: "a debit to an asset", side: "debit", amount: "5", kind: "asset", want: "5" },
    { name: "a credit to an asset", side: "credit", amount: "5", kind: "asset", want: "-5" },
    { name: "a debit to an expense", side: "debit", amount: "5", kind: "expense", want: "5" },
    { name: "a credit to a liability", side: "credit", amount: "5", kind: "liability", want: "5" },
    { name: "a debit to a liability", side: "debit", amount: "5", kind: "liability", want: "-5" },
    { name: "a credit to income", side: "credit", amount: "5", kind: "income", want: "5" },
    { name: "a debit to equity", side: "debit", amount: "5", kind: "equity", want: "-5" },
    {
      name: "more digits than a JavaScript number keeps",
      side: "credit",
      amount: "12345678901234567.89",
      kind: "asset",
      want: "-12345678901234567.89",
    },
    { name: "zero stays unsigned", side: "credit", amount: "0.00", kind: "asset", want: "0.00" },
  ]

  for (const { name, side, amount, kind, want } of cases) {
    it(name, () => {
      expect(signedAmount({ side, amount }, kind)).toBe(want)
    })
  }
})
