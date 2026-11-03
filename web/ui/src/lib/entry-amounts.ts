import type { Entry } from "@/api"

import { addDecimals } from "./decimal"

// An EntryAmount is how much an entry moves in one currency.
export type EntryAmount = { currencyId: number; amount: string }

// entryAmounts is how much an entry moves in each currency: the sum of its
// debits in it, which a balanced entry matches with its credits. An entry
// between currencies, such as an exchange, has one amount for each, in the
// order its postings first use them.
export function entryAmounts(entry: Entry): EntryAmount[] {
  const totals = new Map<number, string>()
  for (const posting of entry.postings) {
    const total = totals.get(posting.currencyId) ?? "0"
    totals.set(posting.currencyId, posting.side === "debit" ? addDecimals(total, posting.amount) : total)
  }
  return [...totals.entries()].map(([currencyId, amount]) => ({ currencyId, amount }))
}
