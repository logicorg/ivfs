import { describe, expect, it } from "vitest"

import type { Asset, Currency, FinancialAccount, Holding } from "@/api"

import { formatUnits, holdingRows, type HoldingRow } from "./holdings"

const account = (id: number, name: string): FinancialAccount => ({
  id,
  name,
  institutionId: 1,
  institutionName: "IBKR",
  closed: false,
  institutionClosed: false,
})
const accounts = [account(3, "Broker"), account(4, "Pension")]
const assets: Asset[] = [
  { id: 7, name: "Vanguard FTSE All-World", ticker: "VWCE", kind: "etf" },
  { id: 8, name: "Apple", kind: "stock" },
]
const currencies: Currency[] = [
  { id: 1, code: "EUR", name: "Euro", precision: 2 },
  { id: 2, code: "USD", name: "US dollar", precision: 2 },
]
const holding = (financialAccountId: number, assetId: number, units: string, extra: Partial<Holding> = {}): Holding => ({
  financialAccountId,
  assetId,
  units,
  costBasis: [],
  unknownCostBasis: false,
  ...extra,
})
const row = (financialAccountId: number, assetId: number, units: string, extra: Partial<HoldingRow> = {}): HoldingRow => ({
  financialAccountId,
  financialAccountName: accounts.find((a) => a.id === financialAccountId)?.name ?? `account ${financialAccountId}`,
  assetId,
  assetName: assets.find((a) => a.id === assetId)?.name ?? `asset ${assetId}`,
  ticker: assets.find((a) => a.id === assetId)?.ticker,
  units,
  costBasis: [],
  averageCost: undefined,
  unknownCostBasis: false,
  ...extra,
})

describe("holdingRows", () => {
  const cases: { name: string; holdings: Holding[]; includeSold?: boolean; want: HoldingRow[] }[] = [
    { name: "no holdings", holdings: [], want: [] },
    {
      name: "a holding with its cost and average cost named",
      holdings: [
        holding(3, 7, "12.5", {
          costBasis: [{ currencyId: 1, amount: "1250.25" }],
          averageCost: { currencyId: 1, amount: "100.02" },
        }),
      ],
      want: [
        row(3, 7, "12.5", {
          costBasis: [{ amount: "1250.25", currencyCode: "EUR", precision: 2 }],
          averageCost: { amount: "100.02", currencyCode: "EUR", precision: 2 },
        }),
      ],
    },
    {
      name: "a cost in several currencies, part of it unknown",
      holdings: [
        holding(3, 8, "4", {
          costBasis: [
            { currencyId: 1, amount: "10" },
            { currencyId: 2, amount: "20" },
          ],
          unknownCostBasis: true,
        }),
      ],
      want: [
        row(3, 8, "4", {
          costBasis: [
            { amount: "10", currencyCode: "EUR", precision: 2 },
            { amount: "20", currencyCode: "USD", precision: 2 },
          ],
          unknownCostBasis: true,
        }),
      ],
    },
    {
      name: "sorted by account and then by asset",
      holdings: [holding(4, 7, "1"), holding(3, 7, "2"), holding(3, 8, "3")],
      want: [row(3, 8, "3"), row(3, 7, "2"), row(4, 7, "1")],
    },
    {
      name: "a holding sold to zero left out",
      holdings: [holding(3, 7, "0"), holding(3, 8, "0.000"), holding(4, 7, "1")],
      want: [row(4, 7, "1")],
    },
    {
      name: "a holding sold to zero listed when asked",
      holdings: [holding(3, 7, "0")],
      includeSold: true,
      want: [row(3, 7, "0")],
    },
    {
      name: "a short holding is not sold",
      holdings: [holding(3, 7, "-2")],
      want: [row(3, 7, "-2")],
    },
    {
      name: "ids missing from the lists shown as ids",
      holdings: [holding(9, 99, "1", { costBasis: [{ currencyId: 5, amount: "3" }] })],
      want: [
        row(9, 99, "1", { costBasis: [{ amount: "3", currencyCode: "currency 5", precision: undefined }] }),
      ],
    },
  ]

  for (const { name, holdings, includeSold, want } of cases) {
    it(name, () => {
      expect(holdingRows(holdings, accounts, assets, currencies, includeSold)).toEqual(want)
    })
  }
})

describe("formatUnits", () => {
  const cases: { name: string; units: string; want: string }[] = [
    { name: "whole units", units: "12", want: "12" },
    { name: "a fractional share", units: "12.5", want: "12.5" },
    { name: "every fractional digit kept", units: "0.123456789", want: "0.123456789" },
    { name: "thousands grouped, no digit lost", units: "1234567.000000001", want: "1,234,567.000000001" },
    { name: "a short holding", units: "-2.5", want: "-2.5" },
  ]

  for (const { name, units, want } of cases) {
    it(name, () => {
      expect(formatUnits(units, "en-US")).toBe(want)
    })
  }
})
