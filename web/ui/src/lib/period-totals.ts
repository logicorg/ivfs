import type { Category, CategoryTotal } from "@/api"

import { addDecimals, compareDecimals, negate } from "./decimal"

// A PeriodRow is a top-level category's total, with the totals of every
// category under it rolled up into it, or, without an id, the total of what
// has no category.
export type PeriodRow = { id?: number; name: string; kind: Category["kind"]; total: string }

// uncategorizedName names the row of what has no category.
export const uncategorizedName = "Uncategorized"

// CurrencyPeriod is what was earned and spent in one currency over a period,
// and the difference, as exact decimal strings.
export type CurrencyPeriod = {
  currencyCode: string
  income: string
  expenses: string
  net: string
  // rows lists income before expenses, the largest total first in each.
  rows: PeriodRow[]
}

// periodTotals sums each category's total into its top-level category, and
// those by currency, sorted by code. A category whose parent is not listed is
// its own top level, and one that is not listed at all is shown by its id.
// What has no category is a row of its own in each kind.
export function periodTotals(totals: CategoryTotal[], categories: Category[]): CurrencyPeriod[] {
  const byID = new Map(categories.map((category) => [category.id, category]))
  const topLevel = (id: number | undefined): { id?: number; name: string } => {
    if (id === undefined) {
      return { name: uncategorizedName }
    }
    let category = byID.get(id)
    if (!category) {
      return { id, name: `Category ${id}` }
    }
    for (let parent = byID.get(category.parentId ?? -1); parent; parent = byID.get(parent.parentId ?? -1)) {
      category = parent
    }
    return category
  }

  const periods = new Map<string, Map<string, PeriodRow>>()
  for (const { categoryId, kind, currencyCode, total } of totals) {
    const rows = periods.get(currencyCode) ?? new Map<string, PeriodRow>()
    const root = topLevel(categoryId)
    // The kind is part of the key for the rows without a category, one of
    // each kind.
    const key = `${kind}:${root.id ?? ""}`
    const row = rows.get(key) ?? { id: root.id, name: root.name, kind, total: "0" }
    row.total = addDecimals(row.total, total)
    rows.set(key, row)
    periods.set(currencyCode, rows)
  }

  return [...periods.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([currencyCode, byRoot]) => {
      const rows = [...byRoot.values()].sort(
        (a, b) => (a.kind === b.kind ? 0 : a.kind === "income" ? -1 : 1) || compareDecimals(b.total, a.total),
      )
      const sum = (kind: Category["kind"]) =>
        rows.filter((row) => row.kind === kind).reduce((total, row) => addDecimals(total, row.total), "0")
      const income = sum("income")
      const expenses = sum("expense")
      return { currencyCode, income, expenses, net: addDecimals(income, negate(expenses)), rows }
    })
}

// A Month is a calendar month, such as "2026-01".
export type Month = `${number}-${string}`

// monthOf is the month date falls in, in the browser's time zone.
export function monthOf(date: Date): Month {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`
}

// isMonth is whether text is a month such as "2026-01".
export function isMonth(text: unknown): text is Month {
  return typeof text === "string" && /^\d{4}-(0[1-9]|1[0-2])$/.test(text)
}

// addMonths is month moved by count months, back when count is negative.
export function addMonths(month: Month, count: number): Month {
  const [year, index] = month.split("-").map(Number)
  return monthOf(new Date(year, index - 1 + count, 1))
}

// monthRange is the period a month covers, from the start of its first day
// to the start of the next month's, in the browser's time zone.
export function monthRange(month: Month): { from: string; to: string } {
  const [year, index] = month.split("-").map(Number)
  return { from: new Date(year, index - 1, 1).toISOString(), to: new Date(year, index, 1).toISOString() }
}

// formatMonth shows a month, such as "January 2026", in the reader's locale
// unless one is given.
export function formatMonth(month: Month, locale?: string): string {
  const [year, index] = month.split("-").map(Number)
  return new Intl.DateTimeFormat(locale, { month: "long", year: "numeric" }).format(new Date(year, index - 1, 1))
}
