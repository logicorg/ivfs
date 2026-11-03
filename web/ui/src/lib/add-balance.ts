import * as z from "zod"

import type { BalanceForm } from "@/api"

import { amountText, startOfDay } from "./open-account"

// addBalanceForm is the form as the person fills it in: an empty input is an
// empty string, and the date is a day, not an instant.
export const addBalanceForm = z.object({
  name: z.string().trim().min(1, "A name such as Savings vault"),
  currencyCode: z.string().min(1, "Choose a currency"),
  openingBalance: amountText,
  openingBalanceAt: z.string(),
})

export type AddBalanceFormValues = z.infer<typeof addBalanceForm>

// newBalanceForm is values as the gateway takes them. The opening balance
// holds from the start of the chosen day, in the browser's time zone.
export function newBalanceForm(values: AddBalanceFormValues): BalanceForm {
  return {
    name: values.name,
    currencyCode: values.currencyCode,
    openingBalance: values.openingBalance || undefined,
    openingBalanceAt: startOfDay(values.openingBalanceAt),
  }
}
