import { describe, expect, it } from "vitest"

import type { AssetValues, Currency } from "@/api"

import { marketValue, parseProviderChoices, type MarketValue, type ProviderChoices } from "./asset-values"

describe("parseProviderChoices", () => {
  const cases: { name: string; stored: string | null; want: ProviderChoices }[] = [
    { name: "nothing stored", stored: null, want: {} },
    { name: "a choice for each of two assets", stored: '{"7":"yahoo","9":"manual"}', want: { "7": "yahoo", "9": "manual" } },
    { name: "not JSON", stored: "yahoo", want: {} },
    { name: "a list, not choices", stored: '["yahoo"]', want: {} },
    { name: "null", stored: "null", want: {} },
    { name: "a key that is not an id, left out", stored: '{"apple":"yahoo","7":"yahoo"}', want: { "7": "yahoo" } },
    { name: "a choice that is not a name, left out", stored: '{"7":1,"9":"manual"}', want: { "9": "manual" } },
  ]

  for (const { name, stored, want } of cases) {
    it(name, () => {
      expect(parseProviderChoices(stored)).toEqual(want)
    })
  }
})

describe("marketValue", () => {
  const currencies: Currency[] = [
    { id: 1, code: "EUR", name: "Euro", precision: 2 },
    { id: 2, code: "USD", name: "US dollar", precision: 2 },
  ]
  const values = (...providers: [string, string, number][]): AssetValues => ({
    values: providers.map(([provider, value, currencyId]) => ({
      provider,
      values: [
        { date: "2026-10-08", value: "1", currencyId, observedAt: "2026-10-08T20:00:00Z" },
        { date: "2026-10-09", value, currencyId, observedAt: "2026-10-09T20:00:00Z" },
      ],
    })),
    errors: [],
  })
  const worth = (provider: string, providers: string[], price: string, value: string, currencyCode: string): MarketValue => ({
    providers,
    provider,
    price: { amount: price, currencyCode, precision: 2 },
    value: { amount: value, currencyCode, precision: 2 },
    observedAt: "2026-10-09T20:00:00Z",
  })

  const cases: { name: string; values?: AssetValues; chosen?: string; want?: MarketValue }[] = [
    { name: "not loaded yet" },
    { name: "no provider has a value", values: { values: [], errors: [{ provider: "yahoo", detail: "no answer within 5s" }] } },
    {
      name: "no choice, so the first provider, at its latest value",
      values: values(["manual", "230", 2], ["yahoo", "231.45", 2]),
      want: worth("manual", ["manual", "yahoo"], "230", "2875", "USD"),
    },
    {
      name: "the chosen provider",
      values: values(["manual", "230", 2], ["yahoo", "231.45", 1]),
      chosen: "yahoo",
      want: worth("yahoo", ["manual", "yahoo"], "231.45", "2893.125", "EUR"),
    },
    {
      name: "a chosen provider with no value, so the first",
      values: values(["manual", "230", 2]),
      chosen: "yahoo",
      want: worth("manual", ["manual"], "230", "2875", "USD"),
    },
    {
      name: "a provider with an empty series, not offered",
      values: { values: [{ provider: "manual", values: [] }, ...values(["yahoo", "231.45", 2]).values], errors: [] },
      want: worth("yahoo", ["yahoo"], "231.45", "2893.125", "USD"),
    },
    { name: "a currency not listed", values: values(["yahoo", "231.45", 9]) },
  ]

  for (const { name, values, chosen, want } of cases) {
    it(name, () => {
      expect(marketValue("12.5", values, chosen, currencies)).toEqual(want)
    })
  }
})
