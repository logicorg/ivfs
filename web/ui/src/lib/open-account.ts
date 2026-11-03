import * as z from "zod"

import type { AccountForm } from "@/api"

// amountText is an amount as typed: empty, or a decimal with no thousands
// separator.
export const amountText = z
  .string()
  .trim()
  .regex(/^(-?[0-9]+(\.[0-9]+)?)?$/, "A number such as 1200.50")

// startOfDay is the instant a day, such as 2026-01-31, starts in the browser's
// time zone, or undefined for no day.
export function startOfDay(day: string): string | undefined {
  return day ? new Date(`${day}T00:00`).toISOString() : undefined
}

// openAccountForm is the form as the person fills it in: an empty input is
// an empty string, and the date is a day, not an instant.
export const openAccountForm = z
  .object({
    type: z.enum(["bank", "card", "brokerage", "cash"]),
    institutionName: z.string().trim(),
    accountName: z.string().trim(),
    currencies: z
      .array(
        z.object({
          code: z.string().min(1, "Choose a currency"),
          openingBalance: amountText,
        }),
      )
      .min(1, "Add a currency"),
    openingBalancesAt: z.string(),
  })
  .refine((form) => form.type === "cash" || form.institutionName !== "", {
    message: "The institution that holds the account",
    path: ["institutionName"],
  })

export type OpenAccountFormValues = z.infer<typeof openAccountForm>

// newAccountForm is values as the gateway takes them. The opening balances
// hold from the start of the chosen day, in the browser's time zone.
export function newAccountForm(values: OpenAccountFormValues): AccountForm {
  return {
    type: values.type,
    institutionName: values.type === "cash" ? undefined : values.institutionName,
    accountName: values.accountName || undefined,
    currencies: values.currencies.map((currency) => ({
      code: currency.code,
      openingBalance: currency.openingBalance || undefined,
    })),
    openingBalancesAt: startOfDay(values.openingBalancesAt),
  }
}
