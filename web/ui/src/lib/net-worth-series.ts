import type { Asset, AssetValue, AssetValues, ConversionRate, ConversionRates, Currency, NetWorthSeries } from "@/api"

import type { ProviderChoices } from "./asset-values"
import { addDecimals, compareDecimals, multiplyDecimals, negate } from "./decimal"

// Interval is how far apart a chart's dates are, as the values and rates
// endpoints take it.
export type Interval = "day" | "week" | "month"

// chartInterval picks the step for a range, so a chart has from a few dozen
// to about a hundred and fifty points whatever its length.
export function chartInterval(from: string, to: string): Interval {
  const days = (Date.parse(to) - Date.parse(from)) / 86_400_000
  return days <= 120 ? "day" : days <= 1_000 ? "week" : "month"
}

// chartDates are the dates of a range with the rules of the API's
// Range.Dates: the last day of each interval from from to to, weeks ending
// on Sunday, and to itself. Values and rates come back on the same dates.
export function chartDates(from: string, to: string, interval: Interval): string[] {
  const last = Date.parse(to)
  const end = (day: Date): Date => {
    if (interval === "week") {
      return addDays(day, (7 - day.getUTCDay()) % 7)
    }
    if (interval === "month") {
      return new Date(Date.UTC(day.getUTCFullYear(), day.getUTCMonth() + 1, 0))
    }
    return day
  }
  const dates: string[] = []
  for (let date = end(new Date(from)); date.getTime() <= last; date = end(addDays(date, 1))) {
    dates.push(isoDate(date))
  }
  if (dates.at(-1) !== to) {
    dates.push(to)
  }
  return dates
}

function addDays(day: Date, days: number): Date {
  return new Date(day.getTime() + days * 86_400_000)
}

function isoDate(day: Date): string {
  return day.toISOString().slice(0, 10)
}

// NetWorthPoint is net worth on one date in the reporting currency, as exact
// decimal strings: balances are what the ledger's asset accounts hold,
// investments the market value of the units, and liabilities what is owed.
export type NetWorthPoint = {
  date: string
  balances: string
  investments: string
  liabilities: string
  netWorth: string
  // flags name what the point leaves out for want of a value or a rate.
  flags: string[]
}

// Valuation is what valueNetWorth needs: the series, the dates to chart, and
// the values and rates fetched for them.
export type Valuation = {
  series: NetWorthSeries
  dates: string[]
  // currencyId is the reporting currency.
  currencyId: number
  currencies: Currency[]
  assets: Asset[]
  // values are by asset id; chosen picks a provider among them.
  values: Map<number, AssetValues>
  chosen: ProviderChoices
  // rates are by base currency id, quoted in the reporting currency.
  rates: Map<number, ConversionRates>
}

// valueNetWorth values the series on each date (ADR 0015). Balances, units,
// values and rates are each carried forward from their latest row on or
// before the date. Gaps follow the exclude policy: a holding without a value,
// or anything in a currency without a rate, is left out of the point and
// named in its flags, never guessed.
export function valueNetWorth({ series, dates, currencyId, currencies, assets, values, chosen, rates }: Valuation): NetWorthPoint[] {
  const codes = new Map(currencies.map((currency) => [currency.id, currency.code]))
  const code = (id: number) => codes.get(id) ?? `currency ${id}`
  const names = new Map(assets.map((asset) => [asset.id, asset.ticker ?? asset.name]))

  const balances = carried(groupBy(series.balances, (row) => row.currencyId))
  const units = carried(groupBy(series.units, (row) => row.assetId))
  const prices = carried(
    new Map([...values].map(([assetId, answer]) => [assetId, chosenValues(answer, chosen[String(assetId)])])),
  )
  const quotes = carried(new Map([...rates].map(([baseId, answer]) => [baseId, firstRates(answer)])))
  const rate = (id: number, date: string): string | undefined =>
    id === currencyId ? "1" : quotes.get(id)?.(date)?.rate

  return dates.map((date) => {
    const flags: string[] = []
    let held = "0"
    let owed = "0"
    let invested = "0"
    for (const [id, at] of balances) {
      const balance = at(date)
      if (!balance || (isZero(balance.assets) && isZero(balance.liabilities))) {
        continue
      }
      const r = rate(id, date)
      if (r === undefined) {
        flags.push(`no ${code(id)} rate`)
        continue
      }
      held = addDecimals(held, multiplyDecimals(balance.assets, r))
      owed = addDecimals(owed, multiplyDecimals(balance.liabilities, r))
    }
    for (const [id, at] of units) {
      const holding = at(date)
      if (!holding || isZero(holding.units)) {
        continue
      }
      const name = names.get(id) ?? `asset ${id}`
      const price = prices.get(id)?.(date)
      if (!price) {
        flags.push(`no value for ${name}`)
        continue
      }
      const r = rate(price.currencyId, date)
      if (r === undefined) {
        flags.push(`no ${code(price.currencyId)} rate for ${name}`)
        continue
      }
      invested = addDecimals(invested, multiplyDecimals(multiplyDecimals(holding.units, price.value), r))
    }
    return {
      date,
      balances: held,
      investments: invested,
      liabilities: owed,
      netWorth: addDecimals(addDecimals(held, invested), negate(owed)),
      flags,
    }
  })
}

// chosenValues are the chosen provider's values, or the first provider's
// with any, as the investments screen picks them.
function chosenValues(answer: AssetValues, chosen: string | undefined): AssetValue[] {
  const answered = answer.values.filter((provider) => provider.values.length > 0)
  return (answered.find((provider) => provider.provider === chosen) ?? answered[0])?.values ?? []
}

// firstRates are the first provider's rates with any: rates have no choice,
// so the configured order decides.
function firstRates(answer: ConversionRates): ConversionRate[] {
  return answer.rates.find((provider) => provider.rates.length > 0)?.rates ?? []
}

function groupBy<T>(rows: T[], key: (row: T) => number): Map<number, T[]> {
  const groups = new Map<number, T[]>()
  for (const row of rows) {
    groups.set(key(row), [...(groups.get(key(row)) ?? []), row])
  }
  return groups
}

// carried turns each list, oldest first, into a lookup of its latest row on
// or before a date, undefined before its first.
function carried<T extends { date: string }>(lists: Map<number, T[]>): Map<number, (date: string) => T | undefined> {
  return new Map(
    [...lists].map(([id, rows]) => [id, (date: string) => rows.findLast((row) => row.date.slice(0, 10) <= date)]),
  )
}

function isZero(amount: string): boolean {
  return compareDecimals(amount, "0") === 0
}

// heldAssetIds are the assets with units in the series, to fetch values for.
export function heldAssetIds(series: NetWorthSeries): number[] {
  return [...new Set(series.units.map((row) => row.assetId))].sort((a, b) => a - b)
}

// rateCurrencyIds are the currencies other than the reporting one that the
// balances, or the chosen values of the held assets, are in: those to fetch
// rates for.
export function rateCurrencyIds(
  series: NetWorthSeries,
  values: Map<number, AssetValues>,
  chosen: ProviderChoices,
  currencyId: number,
): number[] {
  const ids = new Set(series.balances.map((row) => row.currencyId))
  for (const [assetId, answer] of values) {
    for (const value of chosenValues(answer, chosen[String(assetId)])) {
      ids.add(value.currencyId)
    }
  }
  ids.delete(currencyId)
  return [...ids].sort((a, b) => a - b)
}

// A Stretch is a run of chart dates, first and last included.
export type Stretch = { from: string; to: string }

// flaggedStretches are the runs of consecutive points with flags, which the
// chart shades: their totals leave something out.
export function flaggedStretches(points: NetWorthPoint[]): Stretch[] {
  const stretches: Stretch[] = []
  let open: Stretch | undefined
  for (const { date, flags } of points) {
    if (flags.length === 0) {
      open = undefined
    } else if (open) {
      open.to = date
    } else {
      open = { from: date, to: date }
      stretches.push(open)
    }
  }
  return stretches
}

// monthsBefore is the date months before date, such as "2025-10-31" for a
// year before "2026-10-31". A day the month lacks becomes its last, so a
// month before March 31 is February 28, not March 3.
export function monthsBefore(date: string, months: number): string {
  const [year, month, day] = date.split("-").map(Number)
  const last = new Date(Date.UTC(year, month - 1 - months + 1, 0)).getUTCDate()
  return isoDate(new Date(Date.UTC(year, month - 1 - months, Math.min(day, last))))
}
