import type { NegativeHolding } from "@/api"
import { formatDate } from "@/lib/dates"

// describeNegativeHolding says which position a warning is about and where it
// first goes negative, such as "ACME in IBKR is -90 units on Mar 3, 2026". The
// date is in the reader's locale and time zone unless they are given.
export function describeNegativeHolding(warning: NegativeHolding, locale?: string, timeZone?: string): string {
  return `${warning.assetName} in ${warning.financialAccountName} is ${warning.units} units on ${formatDate(warning.at, false, locale, timeZone)}`
}
