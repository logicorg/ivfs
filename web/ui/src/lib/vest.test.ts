import { describe, expect, it } from "vitest"

import type { SellForm, VestForm } from "@/api"

import { newAsset } from "./trade"
import { newSellToCover, newVest, vestForm, type VestFormValues } from "./vest"

const vest: VestFormValues = {
  financialAccountId: "3",
  currencyCode: "USD",
  assetId: "8",
  assetName: "",
  assetTicker: "",
  assetKind: "",
  unitsVested: "100",
  unitsWithheld: "",
  fairMarketValue: "182.40",
  incomeCategoryId: "600",
  taxCategoryId: "",
  occurredOn: "",
  description: "",
  sellToCover: false,
  soldUnits: "",
  soldAmount: "",
  soldFee: "",
}

const withheld: VestFormValues = { ...vest, unitsWithheld: "40", taxCategoryId: "601" }
const covered: VestFormValues = { ...vest, sellToCover: true, soldUnits: "12", soldAmount: "2188.80" }

describe("vestForm", () => {
  const cases: { name: string; values: VestFormValues; wantErrors: string[] }[] = [
    { name: "a vest", values: vest, wantErrors: [] },
    { name: "with units withheld", values: withheld, wantErrors: [] },
    { name: "zero withheld needs no tax category", values: { ...vest, unitsWithheld: "0" }, wantErrors: [] },
    { name: "with a sale to cover", values: { ...covered, soldFee: "1.50" }, wantErrors: [] },
    {
      name: "nothing chosen",
      values: { ...vest, financialAccountId: "", currencyCode: "", assetId: "", incomeCategoryId: "" },
      wantErrors: ["financialAccountId", "currencyCode", "assetId", "incomeCategoryId"],
    },
    { name: "a new asset with no name or kind", values: { ...vest, assetId: newAsset }, wantErrors: ["assetName", "assetKind"] },
    { name: "zero units vested", values: { ...vest, unitsVested: "0" }, wantErrors: ["unitsVested"] },
    { name: "a value that is not a number", values: { ...vest, fairMarketValue: "1,82" }, wantErrors: ["fairMarketValue"] },
    { name: "units withheld that are not a number", values: { ...vest, unitsWithheld: "-1" }, wantErrors: ["unitsWithheld"] },
    { name: "units withheld with no tax category", values: { ...withheld, taxCategoryId: "" }, wantErrors: ["taxCategoryId"] },
    { name: "every unit withheld", values: { ...withheld, unitsWithheld: "100.0" }, wantErrors: ["unitsWithheld"] },
    { name: "the sale's fields ignored when off", values: { ...vest, soldUnits: "x", soldFee: "" }, wantErrors: [] },
    {
      name: "a sale to cover with no units or price",
      values: { ...covered, soldUnits: "", soldAmount: "0" },
      wantErrors: ["soldUnits", "soldAmount"],
    },
    { name: "a sale's fee equal to its price", values: { ...covered, soldFee: "2188.8" }, wantErrors: ["soldFee"] },
  ]

  for (const { name, values, wantErrors } of cases) {
    it(name, () => {
      const result = vestForm.safeParse(values)
      expect(result.error?.issues.map((issue) => issue.path.join(".")) ?? []).toEqual(wantErrors)
    })
  }
})

describe("newVest", () => {
  const now = new Date("2026-02-01T09:30:00Z")
  const base: VestForm = {
    idempotencyKey: "key",
    financialAccountId: 3,
    assetId: 8,
    currencyCode: "USD",
    unitsVested: "100",
    unitsWithheld: undefined,
    fairMarketValue: "182.40",
    incomeCategoryLedgerAccountId: 600,
    taxCategoryLedgerAccountId: undefined,
    occurredAt: "2026-02-01T09:30:00.000Z",
    description: undefined,
  }
  const cases: { name: string; values: VestFormValues; assetId: number; want: VestForm }[] = [
    { name: "now, with nothing withheld", values: vest, assetId: 8, want: base },
    {
      name: "a tax category left from zero withheld is not sent",
      values: { ...vest, unitsWithheld: "0", taxCategoryId: "601" },
      assetId: 8,
      want: base,
    },
    {
      name: "on a day, with units withheld and a note, of the asset just created",
      values: { ...withheld, occurredOn: "2026-01-31", description: "Q1 vest" },
      assetId: 9,
      want: {
        ...base,
        assetId: 9,
        unitsWithheld: "40",
        taxCategoryLedgerAccountId: 601,
        occurredAt: new Date("2026-01-31T00:00").toISOString(),
        description: "Q1 vest",
      },
    },
  ]

  for (const { name, values, assetId, want } of cases) {
    it(name, () => {
      expect(newVest(values, assetId, "key", now)).toEqual(want)
    })
  }
})

describe("newSellToCover", () => {
  const now = new Date("2026-02-01T09:30:00Z")
  const base: SellForm = {
    idempotencyKey: "key",
    financialAccountId: 3,
    assetId: 8,
    currencyCode: "USD",
    units: "12",
    amount: "2188.80",
    fee: undefined,
    occurredAt: "2026-02-01T09:30:00.000Z",
    description: "Sold to cover the vest's tax",
  }
  const cases: { name: string; values: VestFormValues; assetId: number; want: SellForm }[] = [
    { name: "now, with no fee", values: covered, assetId: 8, want: base },
    {
      name: "on the vest's day, with a fee, of the asset just created",
      values: { ...covered, soldFee: "1.50", occurredOn: "2026-01-31", description: "Q1 vest" },
      assetId: 9,
      want: { ...base, assetId: 9, fee: "1.50", occurredAt: new Date("2026-01-31T00:00").toISOString() },
    },
  ]

  for (const { name, values, assetId, want } of cases) {
    it(name, () => {
      expect(newSellToCover(values, assetId, "key", now)).toEqual(want)
    })
  }
})
