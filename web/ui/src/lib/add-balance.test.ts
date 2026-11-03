import { describe, expect, it } from "vitest"

import type { BalanceForm } from "@/api"

import { addBalanceForm, newBalanceForm, type AddBalanceFormValues } from "./add-balance"

const vault: AddBalanceFormValues = {
  name: "Savings vault",
  currencyCode: "EUR",
  openingBalance: "300.50",
  openingBalanceAt: "",
}

describe("addBalanceForm", () => {
  const cases: { name: string; values: AddBalanceFormValues; wantErrors: string[] }[] = [
    { name: "a vault", values: vault, wantErrors: [] },
    { name: "no opening balance", values: { ...vault, openingBalance: "" }, wantErrors: [] },
    { name: "a blank name", values: { ...vault, name: " " }, wantErrors: ["name"] },
    { name: "a currency not chosen", values: { ...vault, currencyCode: "" }, wantErrors: ["currencyCode"] },
    { name: "an amount with a thousands separator", values: { ...vault, openingBalance: "1,200" }, wantErrors: ["openingBalance"] },
  ]

  for (const { name, values, wantErrors } of cases) {
    it(name, () => {
      const result = addBalanceForm.safeParse(values)
      expect(result.error?.issues.map((issue) => issue.path.join(".")) ?? []).toEqual(wantErrors)
    })
  }
})

describe("newBalanceForm", () => {
  const cases: { name: string; values: AddBalanceFormValues; want: BalanceForm }[] = [
    {
      name: "the empty fields left out",
      values: { ...vault, openingBalance: "" },
      want: { name: "Savings vault", currencyCode: "EUR" },
    },
    {
      name: "the day as its start in the browser's time zone",
      values: { ...vault, openingBalanceAt: "2026-01-31" },
      want: {
        name: "Savings vault",
        currencyCode: "EUR",
        openingBalance: "300.50",
        openingBalanceAt: new Date(2026, 0, 31).toISOString(),
      },
    },
  ]

  for (const { name, values, want } of cases) {
    it(name, () => {
      expect(newBalanceForm(values)).toEqual(want)
    })
  }
})
