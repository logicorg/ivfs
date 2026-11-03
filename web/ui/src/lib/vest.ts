import * as z from "zod"

import type { SellForm, VestForm } from "@/api"

import { compareDecimals } from "./decimal"
import { startOfDay } from "./open-account"
import { checkNewAsset, decimal, optionalText, positiveText } from "./trade"

// vestForm is a vest as the person fills it in, with the sale that covers its
// tax when sellToCover is on: ids as the selects hold them, an empty input as
// an empty string, and the date as a day. The sale's fields count only when
// sellToCover is on, the tax category only when units are withheld.
export const vestForm = z
  .object({
    financialAccountId: z.string().min(1, "Choose one"),
    currencyCode: z.string().min(1, "Choose one"),
    assetId: z.string().min(1, "Choose one"),
    assetName: z.string().trim(),
    assetTicker: z.string().trim(),
    assetKind: z.string(),
    unitsVested: positiveText("100"),
    unitsWithheld: optionalText("40"),
    fairMarketValue: positiveText("182.40"),
    incomeCategoryId: z.string().min(1, "Choose one"),
    taxCategoryId: z.string(),
    occurredOn: z.string(),
    description: z.string().trim(),
    sellToCover: z.boolean(),
    soldUnits: z.string().trim(),
    soldAmount: z.string().trim(),
    soldFee: optionalText("1.50"),
  })
  .superRefine(checkNewAsset)
  .superRefine((form, context) => {
    const withheld = decimal.test(form.unitsWithheld) && compareDecimals(form.unitsWithheld, "0") > 0
    if (withheld && decimal.test(form.unitsVested) && compareDecimals(form.unitsWithheld, form.unitsVested) >= 0) {
      context.addIssue({ code: "custom", message: "Fewer than the units vested", path: ["unitsWithheld"] })
    }
    if (withheld && form.taxCategoryId === "") {
      context.addIssue({ code: "custom", message: "Choose one, as units are withheld", path: ["taxCategoryId"] })
    }
    if (!form.sellToCover) {
      return
    }
    for (const [path, example] of [
      ["soldUnits", "12"],
      ["soldAmount", "2188.80"],
    ] as const) {
      const issues = positiveText(example).safeParse(form[path]).error?.issues ?? []
      for (const issue of issues) {
        context.addIssue({ code: "custom", message: issue.message, path: [path] })
      }
    }
    if (decimal.test(form.soldAmount) && decimal.test(form.soldFee) && compareDecimals(form.soldFee, form.soldAmount) >= 0) {
      context.addIssue({ code: "custom", message: "Less than the price received", path: ["soldFee"] })
    }
  })

export type VestFormValues = z.infer<typeof vestForm>

// newVest is values as the gateway takes them, for assetId, which is the new
// asset's once it is created. A tax category is sent only with units withheld,
// which the gateway would otherwise ignore.
export function newVest(values: VestFormValues, assetId: number, idempotencyKey: string, now: Date): VestForm {
  const withheld = values.unitsWithheld !== "" && compareDecimals(values.unitsWithheld, "0") > 0
  return {
    idempotencyKey,
    financialAccountId: Number(values.financialAccountId),
    assetId,
    currencyCode: values.currencyCode,
    unitsVested: values.unitsVested,
    unitsWithheld: withheld ? values.unitsWithheld : undefined,
    fairMarketValue: values.fairMarketValue,
    incomeCategoryLedgerAccountId: Number(values.incomeCategoryId),
    taxCategoryLedgerAccountId: withheld ? Number(values.taxCategoryId) : undefined,
    occurredAt: vestedAt(values, now),
    description: values.description || undefined,
  }
}

// newSellToCover is the sale of some of the units delivered, recorded after
// the vest with a key of its own (ADR 0009), at the same instant: on a day
// chosen, both start it, so the sale cannot come before the units it sells.
export function newSellToCover(values: VestFormValues, assetId: number, idempotencyKey: string, now: Date): SellForm {
  return {
    idempotencyKey,
    financialAccountId: Number(values.financialAccountId),
    assetId,
    currencyCode: values.currencyCode,
    units: values.soldUnits,
    amount: values.soldAmount,
    fee: values.soldFee || undefined,
    occurredAt: vestedAt(values, now),
    description: "Sold to cover the vest's tax",
  }
}

// vestedAt is the start of the chosen day, in the browser's time zone, or now
// when no day is chosen.
function vestedAt(values: VestFormValues, now: Date): string {
  return startOfDay(values.occurredOn) ?? now.toISOString()
}
