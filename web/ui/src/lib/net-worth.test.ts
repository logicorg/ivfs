import { describe, expect, it } from "vitest"

import type { Balance } from "@/api"

import { netWorth, type CurrencyNetWorth } from "./net-worth"

const balance = (kind: Balance["kind"], currencyCode: string, amount: string): Balance => ({
  ledgerAccountId: 1,
  ledgerAccountName: "Cash",
  kind,
  currencyId: 1,
  currencyCode,
  balance: amount,
  closed: false,
})

describe("netWorth", () => {
  const cases: { name: string; balances: Balance[]; want: CurrencyNetWorth[] }[] = [
    { name: "no balances", balances: [], want: [] },
    {
      name: "a card's debt taken from what is held",
      balances: [balance("asset", "EUR", "1000"), balance("asset", "EUR", "200.5"), balance("liability", "EUR", "300")],
      want: [{ currencyCode: "EUR", held: "1200.5", owed: "300", net: "900.5" }],
    },
    {
      name: "currencies kept apart, sorted by code",
      balances: [balance("asset", "USD", "10"), balance("asset", "EUR", "20")],
      want: [
        { currencyCode: "EUR", held: "20", owed: "0", net: "20" },
        { currencyCode: "USD", held: "10", owed: "0", net: "10" },
      ],
    },
    {
      name: "more owed than held",
      balances: [balance("liability", "EUR", "50")],
      want: [{ currencyCode: "EUR", held: "0", owed: "50", net: "-50" }],
    },
    {
      name: "an overdraft lowers what is held",
      balances: [balance("asset", "EUR", "-25"), balance("asset", "EUR", "10")],
      want: [{ currencyCode: "EUR", held: "-15", owed: "0", net: "-15" }],
    },
    {
      name: "categories and equity left out",
      balances: [balance("expense", "EUR", "80"), balance("income", "EUR", "-90"), balance("equity", "EUR", "-1000")],
      want: [],
    },
  ]

  for (const { name, balances, want } of cases) {
    it(name, () => {
      expect(netWorth(balances)).toEqual(want)
    })
  }
})
