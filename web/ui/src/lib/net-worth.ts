import type { Balance } from "@/api"

import { addDecimals, negate } from "./decimal"

// CurrencyNetWorth is what is held and owed in one currency, and their
// difference, as exact decimal strings.
export type CurrencyNetWorth = {
  currencyCode: string
  held: string
  owed: string
  net: string
}

// netWorth totals balances by currency: an asset's balance is held, a
// liability's is owed. Any other kind, such as a category's, is not money a
// person has, so it is left out. Currencies are sorted by code.
export function netWorth(balances: Balance[]): CurrencyNetWorth[] {
  const totals = new Map<string, { held: string; owed: string }>()
  for (const balance of balances) {
    if (balance.kind !== "asset" && balance.kind !== "liability") {
      continue
    }
    const total = totals.get(balance.currencyCode) ?? { held: "0", owed: "0" }
    if (balance.kind === "asset") {
      total.held = addDecimals(total.held, balance.balance)
    } else {
      total.owed = addDecimals(total.owed, balance.balance)
    }
    totals.set(balance.currencyCode, total)
  }
  return [...totals.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([currencyCode, { held, owed }]) => ({ currencyCode, held, owed, net: addDecimals(held, negate(owed)) }))
}
