import { useQueries, useQuery } from "@tanstack/react-query"

import type { AssetValues, ConversionRates, Currency, ErrorModel, ProviderError } from "@/api"
import {
  getNetWorthSeriesOptions,
  listAssetsOptions,
  listAssetValuesOptions,
  listConversionRatesOptions,
  listCurrenciesOptions,
} from "@/api/@tanstack/react-query.gen"
import type { ProviderChoices } from "@/lib/asset-values"
import {
  chartDates,
  chartInterval,
  heldAssetIds,
  rateCurrencyIds,
  valueNetWorth,
  type NetWorthPoint,
} from "@/lib/net-worth-series"

// NetWorth is the state of a valued series: loading, failed, or its points,
// with the providers that did not answer, which explain some of its flags.
export type NetWorth =
  | { status: "pending" }
  | { status: "error"; error: unknown }
  | { status: "success"; points: NetWorthPoint[]; currency: Currency; providerErrors: ProviderError[] }

// useNetWorth fetches the series, then the values of the assets it holds,
// then the rates of the currencies those and its balances are in, each on the
// chart's dates: one request per asset and per currency whatever the range
// (ADR 0015). Each step waits for the one before, since it names what to ask.
export function useNetWorth(from: string, to: string, currencyId: number, chosen: ProviderChoices): NetWorth {
  const interval = chartInterval(from, to)
  const series = useQuery(getNetWorthSeriesOptions({ query: { from, to } }))
  const assets = useQuery(listAssetsOptions())
  const currencies = useQuery(listCurrenciesOptions())

  const assetIds = series.data ? heldAssetIds(series.data) : []
  const values = useQueries({
    queries: assetIds.map((id) => listAssetValuesOptions({ path: { id }, query: { from, to, interval } })),
  })
  const valuesLoaded = series.isSuccess && values.every((query) => query.isSuccess)
  const valuesByAsset = new Map<number, AssetValues>(
    valuesLoaded ? assetIds.map((id, n) => [id, values[n].data as AssetValues]) : [],
  )

  const currencyIds = valuesLoaded ? rateCurrencyIds(series.data, valuesByAsset, chosen, currencyId) : []
  const rates = useQueries({
    queries: currencyIds.map((id) =>
      listConversionRatesOptions({ path: { id, quoteId: currencyId }, query: { from, to, interval } }),
    ),
  })

  const failed = [series, assets, currencies, ...values, ...rates].find((query) => query.isError)
  if (failed) {
    return { status: "error", error: failed.error }
  }
  const currency = currencies.data?.currencies.find((currency) => currency.id === currencyId)
  if (!valuesLoaded || !assets.isSuccess || !currencies.isSuccess || rates.some((query) => !query.isSuccess)) {
    return { status: "pending" }
  }
  if (!currency) {
    // Shaped as the API's problems are, so it shows as one.
    const problem: ErrorModel = { status: 404, title: "Not Found", detail: `There is no currency ${currencyId}; pick another one.` }
    return { status: "error", error: problem }
  }

  const ratesByCurrency = new Map<number, ConversionRates>(
    currencyIds.map((id, n) => [id, rates[n].data as ConversionRates]),
  )
  const points = valueNetWorth({
    series: series.data,
    dates: chartDates(from, to, interval),
    currencyId,
    currencies: currencies.data.currencies,
    assets: assets.data.assets,
    values: valuesByAsset,
    chosen,
    rates: ratesByCurrency,
  })
  const providerErrors = [...valuesByAsset.values(), ...ratesByCurrency.values()].flatMap((answer) => answer.errors)
  return { status: "success", points, currency, providerErrors }
}
