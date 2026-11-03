import type { Asset, CostBasis, Currency, FinancialAccount, Holding } from "@/api"

import { compareDecimals } from "./decimal"

// Money is an exact decimal amount in a currency, with the decimal places to
// show it with when the currency is known.
export type Money = {
  amount: string
  currencyCode: string
  precision?: number
}

// HoldingRow is a holding with its account, asset and currencies named, as the
// investments screen lists it.
export type HoldingRow = {
  financialAccountId: number
  financialAccountName: string
  assetId: number
  assetName: string
  ticker?: string
  units: string
  costBasis: Money[]
  averageCost?: Money
  unknownCostBasis: boolean
}

// holdingRows names each holding's account, asset and currencies, sorted by
// account and then by asset. The API returns only ids, so an id missing from a
// list, such as an account deleted since, is shown as one rather than dropped.
// A holding sold to zero is left out unless includeSold is set.
export function holdingRows(
  holdings: Holding[],
  accounts: FinancialAccount[],
  assets: Asset[],
  currencies: Currency[],
  includeSold = false,
): HoldingRow[] {
  const accountNames = new Map(accounts.map((account) => [account.id, account.name]))
  const assetsById = new Map(assets.map((asset) => [asset.id, asset]))
  const currenciesById = new Map(currencies.map((currency) => [currency.id, currency]))
  const money = ({ amount, currencyId }: CostBasis): Money => {
    const currency = currenciesById.get(currencyId)
    return { amount, currencyCode: currency?.code ?? `currency ${currencyId}`, precision: currency?.precision }
  }

  return holdings
    .filter((holding) => includeSold || compareDecimals(holding.units, "0") !== 0)
    .map((holding) => {
      const asset = assetsById.get(holding.assetId)
      return {
        financialAccountId: holding.financialAccountId,
        financialAccountName: accountNames.get(holding.financialAccountId) ?? `account ${holding.financialAccountId}`,
        assetId: holding.assetId,
        assetName: asset?.name ?? `asset ${holding.assetId}`,
        ticker: asset?.ticker,
        units: holding.units,
        costBasis: holding.costBasis.map(money),
        averageCost: holding.averageCost && money(holding.averageCost),
        unknownCostBasis: holding.unknownCostBasis,
      }
    })
    .sort((a, b) => a.financialAccountName.localeCompare(b.financialAccountName) || a.assetName.localeCompare(b.assetName))
}

// formatUnits shows an exact number of units, such as "12.5", in the reader's
// locale unless one is given. Every fractional digit is kept: Intl would round
// a fractional share to three places by default.
export function formatUnits(units: string, locale?: string): string {
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 20 }).format(units as Intl.StringNumericLiteral)
}
