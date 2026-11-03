import type { AssetValues, Currency } from "@/api"

import { multiplyDecimals } from "./decimal"
import type { Money } from "./holdings"

// valueProvidersStorageKey names the reader's provider choices in
// localStorage. They stay in the browser (ADR 0014): which source to trust is
// how this reader looks at an asset, not a fact of the ledger.
export const valueProvidersStorageKey = "ivfs-value-providers"

// ProviderChoices are the provider the reader picked for each asset, by the
// asset's id.
export type ProviderChoices = Record<string, string>

// parseProviderChoices reads stored choices, keeping only well-formed ones:
// anything else, such as what an older build left behind, counts as no
// choice, so the default provider shows.
export function parseProviderChoices(stored: string | null): ProviderChoices {
  let parsed: unknown
  try {
    parsed = JSON.parse(stored ?? "{}")
  } catch {
    return {}
  }
  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
    return {}
  }
  return Object.fromEntries(
    Object.entries(parsed).filter(
      (entry): entry is [string, string] => /^[1-9]\d*$/.test(entry[0]) && typeof entry[1] === "string",
    ),
  )
}

// MarketValue is what a holding is worth by one provider's latest value.
export type MarketValue = {
  // providers are those with a value, in the configured order, to pick from.
  providers: string[]
  provider: string
  price: Money
  value: Money
  observedAt: string
}

// marketValue values units at the latest value of the chosen provider, or of
// the first with one, which is the configured default, when the chosen one
// has none or none was chosen. It is undefined when no provider has a value,
// or the value's currency is not in currencies.
export function marketValue(
  units: string,
  values: AssetValues | undefined,
  chosen: string | undefined,
  currencies: Currency[],
): MarketValue | undefined {
  const answered = (values?.values ?? []).filter((provider) => provider.values.length > 0)
  const shown = answered.find((provider) => provider.provider === chosen) ?? answered[0]
  const latest = shown?.values.at(-1)
  const currency = currencies.find((currency) => currency.id === latest?.currencyId)
  if (!shown || !latest || !currency) {
    return undefined
  }
  const money = (amount: string): Money => ({ amount, currencyCode: currency.code, precision: currency.precision })
  return {
    providers: answered.map((provider) => provider.provider),
    provider: shown.provider,
    price: money(latest.value),
    value: money(multiplyDecimals(units, latest.value)),
    observedAt: latest.observedAt,
  }
}
