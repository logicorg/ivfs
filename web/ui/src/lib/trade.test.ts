import { describe, expect, it } from "vitest"

import type { Asset, AssetForm, Balance, BuyForm, Holding, SellForm } from "@/api"

import {
  assetLabel,
  heldUnits,
  newAsset,
  newAssetForm,
  newBuy,
  newSell,
  saleAssets,
  saleForm,
  tradeCurrencies,
  tradeForm,
  type TradeFormValues,
} from "./trade"

const buy: TradeFormValues = {
  financialAccountId: "3",
  currencyCode: "EUR",
  assetId: "7",
  assetName: "",
  assetTicker: "",
  assetKind: "",
  units: "12.5",
  amount: "1250.25",
  fee: "",
  occurredOn: "",
  description: "",
}

const buyNew: TradeFormValues = { ...buy, assetId: newAsset, assetName: "Vanguard FTSE All-World", assetKind: "etf" }

describe("tradeForm", () => {
  const cases: { name: string; values: TradeFormValues; wantErrors: string[] }[] = [
    { name: "a buy of an asset that exists", values: buy, wantErrors: [] },
    { name: "a buy of a new asset", values: buyNew, wantErrors: [] },
    { name: "a fee", values: { ...buy, fee: "1.50" }, wantErrors: [] },
    {
      name: "nothing chosen",
      values: { ...buy, financialAccountId: "", currencyCode: "", assetId: "" },
      wantErrors: ["financialAccountId", "currencyCode", "assetId"],
    },
    { name: "no units", values: { ...buy, units: " " }, wantErrors: ["units"] },
    { name: "zero units", values: { ...buy, units: "0.000" }, wantErrors: ["units"] },
    { name: "negative units", values: { ...buy, units: "-1" }, wantErrors: ["units"] },
    { name: "an amount with a thousands separator", values: { ...buy, amount: "1,250.25" }, wantErrors: ["amount"] },
    { name: "a zero amount", values: { ...buy, amount: "0" }, wantErrors: ["amount"] },
    { name: "a negative fee", values: { ...buy, fee: "-1" }, wantErrors: ["fee"] },
    {
      name: "a new asset with no name or kind",
      values: { ...buyNew, assetName: " ", assetKind: "" },
      wantErrors: ["assetName", "assetKind"],
    },
    { name: "a new asset of a kind that is not one", values: { ...buyNew, assetKind: "gift" }, wantErrors: ["assetKind"] },
    { name: "an asset that exists needs no name", values: { ...buy, assetKind: "gift" }, wantErrors: [] },
  ]

  for (const { name, values, wantErrors } of cases) {
    it(name, () => {
      const result = tradeForm.safeParse(values)
      expect(result.error?.issues.map((issue) => issue.path.join(".")) ?? []).toEqual(wantErrors)
    })
  }
})

describe("saleForm", () => {
  const cases: { name: string; values: TradeFormValues; wantErrors: string[] }[] = [
    { name: "a sale with no fee", values: buy, wantErrors: [] },
    { name: "a fee less than the price", values: { ...buy, amount: "100", fee: "99.99" }, wantErrors: [] },
    { name: "a fee equal to the price", values: { ...buy, amount: "100", fee: "100.00" }, wantErrors: ["fee"] },
    { name: "a fee more than the price", values: { ...buy, amount: "100", fee: "250" }, wantErrors: ["fee"] },
    { name: "a price that is not a number", values: { ...buy, amount: "1,00", fee: "2" }, wantErrors: ["amount"] },
    { name: "a fee that is not a number", values: { ...buy, fee: "1,50" }, wantErrors: ["fee"] },
  ]

  for (const { name, values, wantErrors } of cases) {
    it(name, () => {
      const result = saleForm.safeParse(values)
      expect(result.error?.issues.map((issue) => issue.path.join(".")) ?? []).toEqual(wantErrors)
    })
  }
})

const holding = (assetId: number, units: string): Holding => ({
  financialAccountId: 3,
  assetId,
  units,
  costBasis: [],
  unknownCostBasis: false,
})

describe("heldUnits", () => {
  const holdings = [holding(7, "12.5"), holding(8, "0")]
  const cases: { name: string; assetId: string; want: string | undefined }[] = [
    { name: "an asset held", assetId: "7", want: "12.5" },
    { name: "an asset sold to zero", assetId: "8", want: "0" },
    { name: "an asset never held", assetId: "9", want: undefined },
  ]

  for (const { name, assetId, want } of cases) {
    it(name, () => {
      expect(heldUnits(holdings, assetId)).toBe(want)
    })
  }
})

describe("saleAssets", () => {
  const assets: Asset[] = [
    { id: 7, name: "Vanguard FTSE All-World", ticker: "VWCE", kind: "etf" },
    { id: 8, name: "Apple", kind: "stock" },
    { id: 9, name: "Bitcoin", kind: "crypto" },
  ]
  const cases: { name: string; holdings: Holding[]; want: { value: string; label: string }[] }[] = [
    {
      name: "nothing held, by name",
      holdings: [],
      want: [
        { value: "8", label: "Apple" },
        { value: "9", label: "Bitcoin" },
        { value: "7", label: "Vanguard FTSE All-World · VWCE" },
      ],
    },
    {
      name: "those held first, with their units",
      holdings: [holding(7, "1234.5"), holding(9, "0.001")],
      want: [
        { value: "9", label: "Bitcoin (0.001 held)" },
        { value: "7", label: "Vanguard FTSE All-World · VWCE (1,234.5 held)" },
        { value: "8", label: "Apple" },
      ],
    },
    {
      name: "sold to zero or short counts as not held",
      holdings: [holding(8, "0"), holding(9, "-2")],
      want: [
        { value: "8", label: "Apple" },
        { value: "9", label: "Bitcoin" },
        { value: "7", label: "Vanguard FTSE All-World · VWCE" },
      ],
    },
  ]

  for (const { name, holdings, want } of cases) {
    it(name, () => {
      expect(saleAssets(assets, holdings, "en-US")).toEqual(want)
    })
  }
})

describe("tradeCurrencies", () => {
  const balance = (currencyCode: string, extra?: Partial<Balance>): Balance => ({
    ledgerAccountId: 1,
    ledgerAccountName: "Cash",
    currencyCode,
    currencyId: 1,
    kind: "asset",
    balance: "0",
    closed: false,
    ...extra,
  })
  const cases: { name: string; balances: Balance[]; want: string[] }[] = [
    { name: "no balances", balances: [], want: [] },
    {
      name: "each currency once, sorted",
      balances: [balance("USD"), balance("EUR"), balance("EUR", { ledgerAccountName: "Investments" })],
      want: ["EUR", "USD"],
    },
    {
      name: "closed balances and debts left out",
      balances: [balance("GBP", { closed: true }), balance("CHF", { kind: "liability" }), balance("EUR")],
      want: ["EUR"],
    },
  ]

  for (const { name, balances, want } of cases) {
    it(name, () => {
      expect(tradeCurrencies(balances)).toEqual(want)
    })
  }
})

describe("assetLabel", () => {
  const cases: { name: string; asset: Asset; want: string }[] = [
    { name: "with a ticker", asset: { id: 7, name: "Vanguard FTSE All-World", ticker: "VWCE", kind: "etf" }, want: "Vanguard FTSE All-World · VWCE" },
    { name: "with none", asset: { id: 8, name: "Gold bar", kind: "commodity" }, want: "Gold bar" },
  ]

  for (const { name, asset, want } of cases) {
    it(name, () => {
      expect(assetLabel(asset)).toBe(want)
    })
  }
})

describe("newAssetForm", () => {
  const cases: { name: string; values: TradeFormValues; want: AssetForm }[] = [
    {
      name: "with a ticker",
      values: { ...buyNew, assetTicker: "VWCE" },
      want: { name: "Vanguard FTSE All-World", ticker: "VWCE", kind: "etf" },
    },
    { name: "with none", values: buyNew, want: { name: "Vanguard FTSE All-World", ticker: undefined, kind: "etf" } },
  ]

  for (const { name, values, want } of cases) {
    it(name, () => {
      expect(newAssetForm(values)).toEqual(want)
    })
  }
})

describe("newBuy", () => {
  const now = new Date("2026-02-01T09:30:00Z")
  const cases: { name: string; values: TradeFormValues; assetId: number; want: BuyForm }[] = [
    {
      name: "now, with no fee or note",
      values: buy,
      assetId: 7,
      want: {
        idempotencyKey: "key",
        financialAccountId: 3,
        assetId: 7,
        currencyCode: "EUR",
        units: "12.5",
        amount: "1250.25",
        fee: undefined,
        occurredAt: "2026-02-01T09:30:00.000Z",
        description: undefined,
      },
    },
    {
      name: "on a day, with a fee and a note, of the asset just created",
      values: { ...buyNew, fee: "1.50", occurredOn: "2026-01-31", description: "Monthly buy" },
      assetId: 9,
      want: {
        idempotencyKey: "key",
        financialAccountId: 3,
        assetId: 9,
        currencyCode: "EUR",
        units: "12.5",
        amount: "1250.25",
        fee: "1.50",
        occurredAt: new Date("2026-01-31T00:00").toISOString(),
        description: "Monthly buy",
      },
    },
  ]

  for (const { name, values, assetId, want } of cases) {
    it(name, () => {
      expect(newBuy(values, assetId, "key", now)).toEqual(want)
    })
  }
})

describe("newSell", () => {
  const now = new Date("2026-02-01T09:30:00Z")
  const cases: { name: string; values: TradeFormValues; want: SellForm }[] = [
    {
      name: "now, with no fee or note",
      values: buy,
      want: {
        idempotencyKey: "key",
        financialAccountId: 3,
        assetId: 7,
        currencyCode: "EUR",
        units: "12.5",
        amount: "1250.25",
        fee: undefined,
        occurredAt: "2026-02-01T09:30:00.000Z",
        description: undefined,
      },
    },
    {
      name: "on a day, with a fee and a note",
      values: { ...buy, fee: "1.50", occurredOn: "2026-01-31", description: "Rebalance" },
      want: {
        idempotencyKey: "key",
        financialAccountId: 3,
        assetId: 7,
        currencyCode: "EUR",
        units: "12.5",
        amount: "1250.25",
        fee: "1.50",
        occurredAt: new Date("2026-01-31T00:00").toISOString(),
        description: "Rebalance",
      },
    },
  ]

  for (const { name, values, want } of cases) {
    it(name, () => {
      expect(newSell(values, "key", now)).toEqual(want)
    })
  }
})
