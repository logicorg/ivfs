import { describe, expect, it } from "vitest"

import type { AccountForm } from "@/api"

import { newAccountForm, openAccountForm, type OpenAccountFormValues } from "./open-account"

const revolut: OpenAccountFormValues = {
  type: "bank",
  institutionName: "Revolut",
  accountName: "",
  currencies: [{ code: "EUR", openingBalance: "1200.50" }],
  openingBalancesAt: "",
}

describe("openAccountForm", () => {
  const cases: { name: string; values: OpenAccountFormValues; wantErrors: string[] }[] = [
    { name: "a bank account", values: revolut, wantErrors: [] },
    { name: "a wallet, with no institution", values: { ...revolut, type: "cash", institutionName: "" }, wantErrors: [] },
    { name: "an overdraft", values: { ...revolut, currencies: [{ code: "EUR", openingBalance: "-30" }] }, wantErrors: [] },
    { name: "no opening balance", values: { ...revolut, currencies: [{ code: "EUR", openingBalance: "" }] }, wantErrors: [] },
    { name: "a bank account with no institution", values: { ...revolut, institutionName: " " }, wantErrors: ["institutionName"] },
    { name: "a currency not chosen", values: { ...revolut, currencies: [{ code: "", openingBalance: "" }] }, wantErrors: ["currencies.0.code"] },
    {
      name: "an amount with a thousands separator",
      values: { ...revolut, currencies: [{ code: "EUR", openingBalance: "1,200" }] },
      wantErrors: ["currencies.0.openingBalance"],
    },
    { name: "no currencies", values: { ...revolut, currencies: [] }, wantErrors: ["currencies"] },
  ]

  for (const { name, values, wantErrors } of cases) {
    it(name, () => {
      const result = openAccountForm.safeParse(values)
      expect(result.error?.issues.map((issue) => issue.path.join(".")) ?? []).toEqual(wantErrors)
    })
  }
})

describe("newAccountForm", () => {
  const cases: { name: string; values: OpenAccountFormValues; want: AccountForm }[] = [
    {
      name: "the empty fields left out",
      values: revolut,
      want: { type: "bank", institutionName: "Revolut", currencies: [{ code: "EUR", openingBalance: "1200.50" }] },
    },
    {
      name: "a wallet drops the institution it hid",
      values: { ...revolut, type: "cash", accountName: "Pocket", currencies: [{ code: "EUR", openingBalance: "" }] },
      want: { type: "cash", accountName: "Pocket", currencies: [{ code: "EUR" }] },
    },
    {
      name: "the day as its start in the browser's time zone",
      values: { ...revolut, openingBalancesAt: "2026-01-31" },
      want: {
        type: "bank",
        institutionName: "Revolut",
        currencies: [{ code: "EUR", openingBalance: "1200.50" }],
        openingBalancesAt: new Date(2026, 0, 31).toISOString(),
      },
    },
  ]

  for (const { name, values, want } of cases) {
    it(name, () => {
      expect(newAccountForm(values)).toEqual(want)
    })
  }
})
