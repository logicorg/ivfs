import * as z from "zod"

import type { Asset, AssetForm, Balance, BuyForm, Holding, SellForm } from "@/api"

import { compareDecimals } from "./decimal"
import { formatUnits } from "./holdings"
import { startOfDay } from "./open-account"

// newAsset is the asset choice that creates one from the name, ticker and kind
// typed in, rather than picking one that exists.
export const newAsset = "new"

export const assetKinds: { value: AssetForm["kind"]; label: string }[] = [
  { value: "etf", label: "ETF" },
  { value: "stock", label: "Stock" },
  { value: "equity", label: "Equity" },
  { value: "fund", label: "Fund" },
  { value: "bond", label: "Bond" },
  { value: "crypto", label: "Crypto" },
  { value: "commodity", label: "Commodity" },
  { value: "other", label: "Other" },
]

// decimal is a number as typed: no sign, no thousands separator.
export const decimal = /^[0-9]+(\.[0-9]+)?$/

// positiveText is a decimal greater than zero. One check, not a regex then a
// refine: zod runs both, and comparing text that is not a number would throw.
export const positiveText = (example: string) =>
  z
    .string()
    .trim()
    .superRefine((text, context) => {
      if (!decimal.test(text)) {
        context.addIssue({ code: "custom", message: `A number such as ${example}` })
      } else if (compareDecimals(text, "0") <= 0) {
        context.addIssue({ code: "custom", message: "More than zero" })
      }
    })

// optionalText is a decimal, or empty for none.
export const optionalText = (example: string) =>
  z
    .string()
    .trim()
    .regex(/^([0-9]+(\.[0-9]+)?)?$/, `A number such as ${example}`)

// NewAssetValues are the fields of a form that can create the asset it trades.
export type NewAssetValues = { assetId: string; assetName: string; assetTicker: string; assetKind: string }

// checkNewAsset asks for the new asset's name and kind, and only when assetId
// is newAsset.
export function checkNewAsset(form: NewAssetValues, context: z.RefinementCtx) {
  if (form.assetId !== newAsset) {
    return
  }
  if (form.assetName === "") {
    context.addIssue({ code: "custom", message: "The asset's name, such as Vanguard FTSE All-World", path: ["assetName"] })
  }
  if (!assetKinds.some((kind) => kind.value === form.assetKind)) {
    context.addIssue({ code: "custom", message: "Choose one", path: ["assetKind"] })
  }
}

// tradeForm is a buy or a sale as the person fills it in: ids as the selects
// hold them, an empty input as an empty string, and the date as a day. The
// new asset's fields count only when assetId is newAsset.
export const tradeForm = z
  .object({
    financialAccountId: z.string().min(1, "Choose one"),
    currencyCode: z.string().min(1, "Choose one"),
    assetId: z.string().min(1, "Choose one"),
    assetName: z.string().trim(),
    assetTicker: z.string().trim(),
    assetKind: z.string(),
    units: positiveText("12.5"),
    amount: positiveText("1200.50"),
    fee: optionalText("1.50"),
    occurredOn: z.string(),
    description: z.string().trim(),
  })
  .superRefine(checkNewAsset)

export type TradeFormValues = z.infer<typeof tradeForm>

// saleForm is tradeForm with the fee taken from the proceeds, so less than
// them. The comparison waits for both to be numbers; their own checks report
// them otherwise.
export const saleForm = tradeForm.superRefine((form, context) => {
  if (decimal.test(form.amount) && decimal.test(form.fee) && compareDecimals(form.fee, form.amount) >= 0) {
    context.addIssue({ code: "custom", message: "Less than the price received", path: ["fee"] })
  }
})

// tradeCurrencies are the currencies an account can trade in: those of its
// open asset balances, sorted. The gateway checks the account has both a Cash
// and an Investments balance in the one chosen.
export function tradeCurrencies(balances: Balance[]): string[] {
  const codes = balances.filter((balance) => !balance.closed && balance.kind === "asset").map((b) => b.currencyCode)
  return [...new Set(codes)].sort()
}

// assetLabel names an asset with its ticker, when it has one.
export function assetLabel(asset: Asset): string {
  return asset.ticker ? `${asset.name} · ${asset.ticker}` : asset.name
}

// newAssetForm is the new asset as the gateway takes it.
export function newAssetForm(values: NewAssetValues): AssetForm {
  return {
    name: values.assetName,
    ticker: values.assetTicker || undefined,
    kind: values.assetKind as AssetForm["kind"],
  }
}

// heldUnits is the units of assetId among an account's holdings, or undefined
// when the account has never held it.
export function heldUnits(holdings: Holding[], assetId: string): string | undefined {
  return holdings.find((holding) => String(holding.assetId) === assetId)?.units
}

// saleAssets are the choices of asset to sell from an account: those it holds
// first, with their units, then the rest. A sale of an asset not held is still
// recorded, for units bought before the ledger began, and the gateway warns
// about it.
export function saleAssets(
  assets: Asset[],
  holdings: Holding[],
  locale?: string,
): { value: string; label: string }[] {
  const held = (asset: Asset) => {
    const units = heldUnits(holdings, String(asset.id))
    return units !== undefined && compareDecimals(units, "0") > 0 ? units : undefined
  }
  return assets
    .map((asset) => ({ asset, units: held(asset) }))
    .sort((a, b) => Number(b.units !== undefined) - Number(a.units !== undefined) || a.asset.name.localeCompare(b.asset.name))
    .map(({ asset, units }) => ({
      value: String(asset.id),
      label: units ? `${assetLabel(asset)} (${formatUnits(units, locale)} held)` : assetLabel(asset),
    }))
}

// newBuy is values as the gateway takes them, for assetId, which is the new
// asset's once it is created.
export function newBuy(values: TradeFormValues, assetId: number, idempotencyKey: string, now: Date): BuyForm {
  return trade(values, assetId, idempotencyKey, now)
}

// newSell is values as the gateway takes them.
export function newSell(values: TradeFormValues, idempotencyKey: string, now: Date): SellForm {
  return trade(values, Number(values.assetId), idempotencyKey, now)
}

// trade happens at the start of the chosen day, in the browser's time zone,
// or at now when no day is chosen.
function trade(values: TradeFormValues, assetId: number, idempotencyKey: string, now: Date): BuyForm & SellForm {
  return {
    idempotencyKey,
    financialAccountId: Number(values.financialAccountId),
    assetId,
    currencyCode: values.currencyCode,
    units: values.units,
    amount: values.amount,
    fee: values.fee || undefined,
    occurredAt: startOfDay(values.occurredOn) ?? now.toISOString(),
    description: values.description || undefined,
  }
}
