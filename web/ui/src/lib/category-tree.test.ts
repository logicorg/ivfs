import { describe, expect, it } from "vitest"

import type { Category } from "@/api"

import { categoryRows, describeCategoryDeletion, parentChoices, replacementChoices } from "./category-tree"

const category = (id: number, name: string, fields: Partial<Category> = {}): Category => ({
  id,
  name,
  kind: "expense",
  closed: false,
  ...fields,
})

const food = category(1, "Food")
const groceries = category(2, "Groceries", { parentId: 1 })
const bakery = category(3, "Bakery", { parentId: 2 })
const eatingOut = category(4, "Eating out", { parentId: 1 })
const travel = category(5, "Travel")
const salary = category(6, "Salary", { kind: "income" })
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

const all = [food, groceries, bakery, eatingOut, travel, salary, oldHobby]

describe("parentChoices", () => {
  const cases: { name: string; kind: Category["kind"]; moving?: Category; want: string[] }[] = [
    {
      name: "a new category: open ones of its kind",
      kind: "expense",
      want: ["Food", "-Eating out", "-Groceries", "--Bakery", "Travel"],
    },
    {
      name: "another kind",
      kind: "income",
      want: ["Salary"],
    },
    {
      name: "a moved category cannot go under itself or what is under it",
      kind: "expense",
      moving: groceries,
      want: ["Food", "-Eating out", "Travel"],
    },
  ]

  for (const { name, kind, moving, want } of cases) {
    it(name, () => {
      expect(names(parentChoices(all, kind, moving))).toEqual(want)
    })
  }
})

describe("replacementChoices", () => {
  const cases: { name: string; categories: Category[]; deleting: Category; want: string[] }[] = [
    {
      name: "the other open ones of its kind with none under them, named with their parents",
      categories: all,
      deleting: travel,
      want: ["Food › Eating out", "Food › Groceries › Bakery"],
    },
    {
      name: "not the category itself",
      categories: all,
      deleting: bakery,
      want: ["Food › Eating out", "Travel"],
    },
    {
      name: "another kind has its own",
      categories: [...all, category(7, "Gifts", { kind: "income" })],
      deleting: salary,
      want: ["Gifts"],
    },
    { name: "the only category", categories: [travel], deleting: travel, want: [] },
  ]

  for (const { name, categories, deleting, want } of cases) {
    it(name, () => {
      expect(replacementChoices(categories, deleting).map((choice) => choice.label)).toEqual(want)
    })
  }
})

describe("describeCategoryDeletion", () => {
  const cases: { name: string; entries: number; replacement?: string; want: string }[] = [
    { name: "no entries", entries: 0, want: "No journal entry is filed under it." },
    { name: "no entries, with a replacement", entries: 0, replacement: "Travel", want: "No journal entry is filed under it." },
    { name: "one entry", entries: 1, want: "1 journal entry loses its category." },
    { name: "several entries", entries: 3, want: "3 journal entries lose their category." },
    { name: "one entry, with a replacement", entries: 1, replacement: "Travel", want: "1 journal entry moves to Travel." },
    {
      name: "several entries, with a replacement",
      entries: 3,
      replacement: "Food › Groceries",
      want: "3 journal entries move to Food › Groceries.",
    },
  ]

  for (const { name, entries, replacement, want } of cases) {
    it(name, () => {
      expect(describeCategoryDeletion(entries, replacement)).toEqual(want)
    })
  }
})
