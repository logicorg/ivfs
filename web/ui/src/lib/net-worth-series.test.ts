import { describe, expect, it } from "vitest"

import type { Asset, AssetValues, ConversionRates, Currency, NetWorthSeries } from "@/api"

import type { ProviderChoices } from "./asset-values"
import {
  chartDates,
  chartInterval,
  flaggedStretches,
  heldAssetIds,
  monthsBefore,
  rateCurrencyIds,
  valueNetWorth,
  type Interval,
  type NetWorthPoint,
  type Stretch,
} from "./net-worth-series"

describe("chartInterval", () => {
  const cases: { name: string; from: string; to: string; want: Interval }[] = [
    { name: "a month, daily", from: "2026-01-01", to: "2026-01-31", want: "day" },
    { name: "a year, weekly", from: "2026-01-01", to: "2026-12-31", want: "week" },
    { name: "five years, monthly", from: "2022-01-01", to: "2026-12-31", want: "month" },
  ]

  for (const { name, from, to, want } of cases) {
    it(name, () => {
      expect(chartInterval(from, to)).toBe(want)
    })
  }
})

describe("chartDates", () => {
  const cases: { name: string; from: string; to: string; interval: Interval; want: string[] }[] = [
    { name: "every day", from: "2026-01-30", to: "2026-02-02", interval: "day", want: ["2026-01-30", "2026-01-31", "2026-02-01", "2026-02-02"] },
    { name: "a single day", from: "2026-01-30", to: "2026-01-30", interval: "day", want: ["2026-01-30"] },
    {
      name: "the Sundays, then to",
      from: "2026-10-01",
      to: "2026-10-14",
      interval: "week",
      want: ["2026-10-04", "2026-10-11", "2026-10-14"],
    },
    { name: "a range ending on a Sunday", from: "2026-10-05", to: "2026-10-11", interval: "week", want: ["2026-10-11"] },
    {
      name: "the months' last days, February's included",
      from: "2026-01-15",
      to: "2026-03-31",
      interval: "month",
      want: ["2026-01-31", "2026-02-28", "2026-03-31"],
    },
    { name: "a range within a month", from: "2026-02-03", to: "2026-02-10", interval: "month", want: ["2026-02-10"] },
  ]

  for (const { name, from, to, interval, want } of cases) {
    it(name, () => {
      expect(chartDates(from, to, interval)).toEqual(want)
    })
  }
})

const euro = 1
const dollar = 2
const pound = 3
const apple = 7
const bitcoin = 9
const currencies: Currency[] = [
  { id: euro, code: "EUR", name: "Euro", precision: 2 },
  { id: dollar, code: "USD", name: "US dollar", precision: 2 },
  { id: pound, code: "GBP", name: "Pound sterling", precision: 2 },
]
const assets: Asset[] = [
  { id: apple, name: "Apple", ticker: "AAPL", kind: "stock" },
  { id: bitcoin, name: "Bitcoin", kind: "crypto" },
]

const appleValues: AssetValues = {
  values: [
    { provider: "manual", values: [] },
    {
      provider: "yahoo",
      values: [
        { date: "2026-01-10", value: "200", currencyId: dollar, observedAt: "2026-01-09T21:00:00Z" },
        { date: "2026-01-20", value: "210.5", currencyId: dollar, observedAt: "2026-01-20T21:00:00Z" },
      ],
    },
    {
      provider: "stooq",
      values: [{ date: "2026-01-01", value: "180", currencyId: euro, observedAt: "2026-01-01T21:00:00Z" }],
    },
  ],
  errors: [],
}
const dollarRates: ConversionRates = {
  rates: [
    { provider: "ecb", rates: [] },
    {
      provider: "frankfurter",
      rates: [
        { date: "2026-01-01", rate: "0.9", observedAt: "2026-01-01T16:00:00Z" },
        { date: "2026-01-20", rate: "0.8", observedAt: "2026-01-20T16:00:00Z" },
      ],
    },
  ],
  errors: [],
}

describe("valueNetWorth", () => {
  const series: NetWorthSeries = {
    balances: [
      { date: "2026-01-01", currencyId: euro, assets: "1000", liabilities: "50" },
      { date: "2026-01-05", currencyId: dollar, assets: "200", liabilities: "0" },
      { date: "2026-01-15", currencyId: euro, assets: "700", liabilities: "0" },
      { date: "2026-01-15", currencyId: pound, assets: "10", liabilities: "0" },
    ],
    units: [
      { date: "2026-01-01", assetId: apple, units: "2.5" },
      { date: "2026-01-12", assetId: bitcoin, units: "0.1" },
      { date: "2026-01-25", assetId: bitcoin, units: "0" },
    ],
  }
  const point = (
    date: string,
    balances: string,
    investments: string,
    liabilities: string,
    netWorth: string,
    ...flags: string[]
  ): NetWorthPoint => ({ date, balances, investments, liabilities, netWorth, flags })

  const withDollar = new Map([[dollar, dollarRates]])

  const cases: {
    name: string
    series: NetWorthSeries
    dates: string[]
    chosen: ProviderChoices
    rates: Map<number, ConversionRates>
    want: NetWorthPoint[]
  }[] = [
    {
      name: "carried forward, converted, and what has no value or rate flagged",
      series,
      dates: ["2026-01-01", "2026-01-10", "2026-01-14", "2026-01-20", "2026-01-31"],
      chosen: {},
      rates: withDollar,
      want: [
        point("2026-01-01", "1000", "0", "50", "950", "no value for AAPL"),
        point("2026-01-10", "1180", "450", "50", "1580"),
        point("2026-01-14", "1180", "450", "50", "1580", "no value for Bitcoin"),
        point("2026-01-20", "860", "421", "0", "1281", "no GBP rate", "no value for Bitcoin"),
        point("2026-01-31", "860", "421", "0", "1281", "no GBP rate"),
      ],
    },
    {
      name: "the chosen provider, in the reporting currency itself",
      series,
      dates: ["2026-01-01"],
      chosen: { [String(apple)]: "stooq" },
      rates: withDollar,
      want: [point("2026-01-01", "1000", "450", "50", "1400")],
    },
    {
      name: "a short position is worth less than nothing",
      series: { balances: [], units: [{ date: "2026-01-10", assetId: apple, units: "-1" }] },
      dates: ["2026-01-10"],
      chosen: {},
      rates: withDollar,
      want: [point("2026-01-10", "0", "-180", "0", "-180")],
    },
    {
      name: "an asset valued in a currency without a rate",
      series: { balances: [], units: [{ date: "2026-01-10", assetId: apple, units: "1" }] },
      dates: ["2026-01-10"],
      chosen: { [String(apple)]: "yahoo" },
      rates: new Map(),
      want: [point("2026-01-10", "0", "0", "0", "0", "no USD rate for AAPL")],
    },
    {
      name: "before anything was held",
      series: { balances: [], units: [] },
      dates: ["2026-01-01"],
      chosen: {},
      rates: withDollar,
      want: [point("2026-01-01", "0", "0", "0", "0")],
    },
  ]

  for (const { name, series, dates, chosen, rates, want } of cases) {
    it(name, () => {
      const values = new Map([[apple, appleValues]])
      expect(valueNetWorth({ series, dates, currencyId: euro, currencies, assets, values, chosen, rates })).toEqual(want)
    })
  }
})

describe("heldAssetIds", () => {
  it("each asset once, by id", () => {
    const series: NetWorthSeries = {
      balances: [],
      units: [
        { date: "2026-01-01", assetId: bitcoin, units: "1" },
        { date: "2026-01-01", assetId: apple, units: "1" },
        { date: "2026-01-02", assetId: bitcoin, units: "0" },
      ],
    }
    expect(heldAssetIds(series)).toEqual([apple, bitcoin])
  })
})

describe("rateCurrencyIds", () => {
  const series: NetWorthSeries = {
    balances: [
      { date: "2026-01-01", currencyId: pound, assets: "1", liabilities: "0" },
      { date: "2026-01-01", currencyId: euro, assets: "1", liabilities: "0" },
    ],
    units: [],
  }
  const cases: { name: string; chosen: ProviderChoices; want: number[] }[] = [
    { name: "the balances' and the default provider's currencies, less the reporting one", chosen: {}, want: [dollar, pound] },
    { name: "the chosen provider's currency", chosen: { [String(apple)]: "stooq" }, want: [pound] },
  ]

  for (const { name, chosen, want } of cases) {
    it(name, () => {
      expect(rateCurrencyIds(series, new Map([[apple, appleValues]]), chosen, euro)).toEqual(want)
    })
  }
})

describe("flaggedStretches", () => {
  const at = (date: string, ...flags: string[]): NetWorthPoint => ({
    date,
    balances: "0",
    investments: "0",
    liabilities: "0",
    netWorth: "0",
    flags,
  })
  const cases: { name: string; points: NetWorthPoint[]; want: Stretch[] }[] = [
    { name: "nothing flagged", points: [at("2026-01-01"), at("2026-01-02")], want: [] },
    {
      name: "two runs, one a single day",
      points: [at("2026-01-01", "no GBP rate"), at("2026-01-02", "no GBP rate"), at("2026-01-03"), at("2026-01-04", "no value for AAPL")],
      want: [
        { from: "2026-01-01", to: "2026-01-02" },
        { from: "2026-01-04", to: "2026-01-04" },
      ],
    },
  ]

  for (const { name, points, want } of cases) {
    it(name, () => {
      expect(flaggedStretches(points)).toEqual(want)
    })
  }
})

describe("monthsBefore", () => {
  const cases: { name: string; date: string; months: number; want: string }[] = [
    { name: "a year", date: "2026-10-31", months: 12, want: "2025-10-31" },
    { name: "into a shorter month", date: "2026-03-31", months: 1, want: "2026-02-28" },
    { name: "across a year", date: "2026-01-15", months: 3, want: "2025-10-15" },
    { name: "to a leap day", date: "2028-03-30", months: 1, want: "2028-02-29" },
  ]

  for (const { name, date, months, want } of cases) {
    it(name, () => {
      expect(monthsBefore(date, months)).toBe(want)
    })
  }
})
