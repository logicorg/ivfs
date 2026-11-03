import { describe, expect, it } from "vitest"

import type { Category, CategoryTotal } from "@/api"

import { addMonths, formatMonth, isMonth, monthOf, monthRange, periodTotals, type CurrencyPeriod, type Month } from "./period-totals"

const category = (id: number, name: string, fields: Partial<Category> = {}): Category => ({
  id,
  name,
  kind: "expense",
  currencyId: 1,
  currencyCode: "EUR",
  closed: false,
  ...fields,
})

const food = category(1, "Food")
const groceries = category(2, "Groceries", { parentId: 1 })
const bakery = category(3, "Bakery", { parentId: 2 })
const travel = category(4, "Travel")
const salary = category(5, "Salary", { kind: "income" })
const gifts = category(6, "Gifts", { kind: "income" })
const dollarTravel = category(7, "Travel", { currencyId: 2, currencyCode: "USD" })
const categories = [food, groceries, bakery, travel, salary, gifts, dollarTravel]

const total = ({ id, kind, currencyId, currencyCode }: Category, amount: string): CategoryTotal => ({
  id,
  kind,
  currencyId,
  currencyCode,
  total: amount,
})

describe("periodTotals", () => {
  const cases: { name: string; totals: CategoryTotal[]; categories: Category[]; want: CurrencyPeriod[] }[] = [
    { name: "no totals", totals: [], categories, want: [] },
    {
      name: "subcategories rolled up into their top-level category",
      totals: [total(food, "10"), total(groceries, "20.5"), total(bakery, "4.5")],
      categories,
      want: [
        {
          currencyCode: "EUR",
          income: "0",
          expenses: "35",
          net: "-35",
          rows: [{ id: 1, name: "Food", kind: "expense", total: "35" }],
        },
      ],
    },
    {
      name: "income first, then expenses, the largest first in each",
      totals: [total(travel, "100"), total(gifts, "50"), total(groceries, "300"), total(salary, "2000")],
      categories,
      want: [
        {
          currencyCode: "EUR",
          income: "2050",
          expenses: "400",
          net: "1650",
          rows: [
            { id: 5, name: "Salary", kind: "income", total: "2000" },
            { id: 6, name: "Gifts", kind: "income", total: "50" },
            { id: 1, name: "Food", kind: "expense", total: "300" },
            { id: 4, name: "Travel", kind: "expense", total: "100" },
          ],
        },
      ],
    },
    {
      name: "a refund lowering the expenses",
      totals: [total(travel, "100"), total(food, "-20")],
      categories,
      want: [
        {
          currencyCode: "EUR",
          income: "0",
          expenses: "80",
          net: "-80",
          rows: [
            { id: 4, name: "Travel", kind: "expense", total: "100" },
            { id: 1, name: "Food", kind: "expense", total: "-20" },
          ],
        },
      ],
    },
    {
      name: "currencies kept apart, sorted by code",
      totals: [total(dollarTravel, "30"), total(travel, "10")],
      categories,
      want: [
        { currencyCode: "EUR", income: "0", expenses: "10", net: "-10", rows: [{ id: 4, name: "Travel", kind: "expense", total: "10" }] },
        { currencyCode: "USD", income: "0", expenses: "30", net: "-30", rows: [{ id: 7, name: "Travel", kind: "expense", total: "30" }] },
      ],
    },
    {
      name: "a category whose parent is not listed is its own top level",
      totals: [total(bakery, "4.5")],
      categories: [bakery],
      want: [
        { currencyCode: "EUR", income: "0", expenses: "4.5", net: "-4.5", rows: [{ id: 3, name: "Bakery", kind: "expense", total: "4.5" }] },
      ],
    },
    {
      name: "a category not listed at all shown by its id",
      totals: [total(travel, "10")],
      categories: [],
      want: [
        { currencyCode: "EUR", income: "0", expenses: "10", net: "-10", rows: [{ id: 4, name: "Category 4", kind: "expense", total: "10" }] },
      ],
    },
  ]

  for (const { name, totals, categories, want } of cases) {
    it(name, () => {
      expect(periodTotals(totals, categories)).toEqual(want)
    })
  }
})

describe("months", () => {
  it("monthOf is the month of a date in the browser's time zone", () => {
    expect(monthOf(new Date(2026, 0, 31, 23, 59))).toBe("2026-01")
  })

  const isMonthCases: { text: unknown; want: boolean }[] = [
    { text: "2026-01", want: true },
    { text: "2026-12", want: true },
    { text: "2026-13", want: false },
    { text: "2026-1", want: false },
    { text: 202601, want: false },
  ]
  for (const { text, want } of isMonthCases) {
    it(`isMonth(${JSON.stringify(text)}) is ${want}`, () => {
      expect(isMonth(text)).toBe(want)
    })
  }

  const addMonthsCases: { month: Month; count: number; want: Month }[] = [
    { month: "2026-01", count: 1, want: "2026-02" },
    { month: "2026-12", count: 1, want: "2027-01" },
    { month: "2026-01", count: -1, want: "2025-12" },
  ]
  for (const { month, count, want } of addMonthsCases) {
    it(`addMonths(${month}, ${count}) is ${want}`, () => {
      expect(addMonths(month, count)).toBe(want)
    })
  }

  it("monthRange runs from the month's first day to the next month's", () => {
    expect(monthRange("2026-12")).toEqual({
      from: new Date(2026, 11, 1).toISOString(),
      to: new Date(2027, 0, 1).toISOString(),
    })
  })

  it("formatMonth names the month", () => {
    expect(formatMonth("2026-01", "en-US")).toBe("January 2026")
  })
})
