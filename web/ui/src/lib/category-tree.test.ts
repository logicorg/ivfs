import { describe, expect, it } from "vitest"

import type { Category } from "@/api"

import { categoryRows, parentChoices } from "./category-tree"

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
const eatingOut = category(4, "Eating out", { parentId: 1 })
const travel = category(5, "Travel")
const salary = category(6, "Salary", { kind: "income" })
const dollarFood = category(7, "Food", { currencyId: 2, currencyCode: "USD" })
const oldHobby = category(8, "Old hobby", { closed: true })
const orphan = category(9, "Orphan", { parentId: 99 })

const names = (rows: { category: Category; depth: number }[]) =>
  rows.map(({ category, depth }) => `${"-".repeat(depth)}${category.name}`)

describe("categoryRows", () => {
  const cases: { name: string; categories: Category[]; want: string[] }[] = [
    {
      name: "children follow their parent, siblings by name",
      categories: [travel, bakery, groceries, eatingOut, food],
      want: ["Food", "-Eating out", "-Groceries", "--Bakery", "Travel"],
    },
    {
      name: "a category whose parent is not listed is top level",
      categories: [orphan, travel],
      want: ["Orphan", "Travel"],
    },
    { name: "no categories", categories: [], want: [] },
  ]

  for (const { name, categories, want } of cases) {
    it(name, () => {
      expect(names(categoryRows(categories))).toEqual(want)
    })
  }
})

describe("parentChoices", () => {
  const all = [food, groceries, bakery, eatingOut, travel, salary, dollarFood, oldHobby]

  const cases: { name: string; kind: Category["kind"]; currencyId: number; moving?: Category; want: string[] }[] = [
    {
      name: "a new category: open ones of its kind and currency",
      kind: "expense",
      currencyId: 1,
      want: ["Food", "-Eating out", "-Groceries", "--Bakery", "Travel"],
    },
    {
      name: "another currency",
      kind: "expense",
      currencyId: 2,
      want: ["Food"],
    },
    {
      name: "another kind",
      kind: "income",
      currencyId: 1,
      want: ["Salary"],
    },
    {
      name: "a moved category cannot go under itself or what is under it",
      kind: "expense",
      currencyId: 1,
      moving: groceries,
      want: ["Food", "-Eating out", "Travel"],
    },
  ]

  for (const { name, kind, currencyId, moving, want } of cases) {
    it(name, () => {
      expect(names(parentChoices(all, kind, currencyId, moving))).toEqual(want)
    })
  }
})
