import { describe, expect, it } from "vitest"

import { formatAmount } from "./money"

describe("formatAmount", () => {
  const cases: { name: string; amount: string; code: string; precision?: number; want: string }[] = [
    { name: "cents padded to the currency's precision", amount: "1200.5", code: "EUR", precision: 2, want: "€1,200.50" },
    { name: "a currency without cents", amount: "1200", code: "JPY", precision: 0, want: "¥1,200" },
    { name: "a negative amount", amount: "-0.01", code: "USD", precision: 2, want: "-$0.01" },
    {
      name: "more digits than a JavaScript number keeps",
      amount: "12345678901234567.89",
      code: "USD",
      precision: 2,
      want: "$12,345,678,901,234,567.89",
    },
    { name: "Intl's precision when the currency's is unknown", amount: "3", code: "EUR", want: "€3.00" },
    { name: "a code Intl does not accept as a currency", amount: "1200.5", code: "USDT", precision: 2, want: "1,200.50 USDT" },
  ]

  for (const { name, amount, code, precision, want } of cases) {
    it(name, () => {
      expect(formatAmount(amount, code, precision, "en-US")).toBe(want)
    })
  }
})
