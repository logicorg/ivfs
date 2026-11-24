import { describe, expect, it } from "vitest"

import type { Preset } from "@/api"

import { countIncluded, offersPreset, presetForm, presetOf, updateCategory, type PresetForm } from "./preset"

const preset: Preset = {
  currencies: [
    { code: "EUR", name: "Euro", symbol: "€", precision: 2 },
    { code: "JPY", name: "Japanese Yen", symbol: "¥", precision: 0 },
  ],
  tags: [{ name: "vacation" }, { name: "recurring" }],
  categories: [
    { name: "Food", kind: "expense", children: [{ name: "Groceries" }, { name: "Eating out" }] },
    { name: "Salary", kind: "income" },
  ],
}

// edit is the form of preset changed by change.
const edit = (change: (form: PresetForm) => PresetForm) => presetOf(change(presetForm(preset)))

describe("presetOf", () => {
  const cases: { name: string; change: (form: PresetForm) => PresetForm; want: Preset }[] = [
    { name: "an untouched form is the preset", change: (form) => form, want: preset },
    {
      name: "an unticked currency and tag are left out",
      change: (form) => ({
        ...form,
        currencies: form.currencies.map((row) => (row.code === "JPY" ? { ...row, included: false } : row)),
        tags: form.tags.map((row) => (row.name === "vacation" ? { ...row, included: false } : row)),
      }),
      want: { ...preset, currencies: [preset.currencies[0]], tags: [{ name: "recurring" }] },
    },
    {
      name: "an unticked category leaves its children out",
      change: (form) => ({
        ...form,
        categories: updateCategory(form.categories, form.categories[0].key, (row) => ({ ...row, included: false })),
      }),
      want: { ...preset, categories: [{ name: "Salary", kind: "income" }] },
    },
    {
      name: "unticking every child leaves the parent without children",
      change: (form) => ({
        ...form,
        categories: updateCategory(form.categories, form.categories[0].key, (row) => ({
          ...row,
          children: row.children.map((child) => ({ ...child, included: false })),
        })),
      }),
      want: { ...preset, categories: [{ name: "Food", kind: "expense" }, { name: "Salary", kind: "income" }] },
    },
    {
      name: "a renamed child is trimmed",
      change: (form) => ({
        ...form,
        categories: updateCategory(form.categories, form.categories[0].children[1].key, (row) => ({
          ...row,
          name: "  Restaurants ",
        })),
      }),
      want: {
        ...preset,
        categories: [
          { name: "Food", kind: "expense", children: [{ name: "Groceries" }, { name: "Restaurants" }] },
          { name: "Salary", kind: "income" },
        ],
      },
    },
    {
      name: "a currency without a symbol sends none",
      change: (form) => ({ ...form, currencies: form.currencies.map((row) => ({ ...row, symbol: " " })) }),
      want: {
        ...preset,
        currencies: [
          { code: "EUR", name: "Euro", precision: 2 },
          { code: "JPY", name: "Japanese Yen", precision: 0 },
        ],
      },
    },
  ]

  for (const { name, change, want } of cases) {
    it(name, () => expect(edit(change)).toEqual(want))
  }
})

describe("countIncluded", () => {
  it("counts every ticked item, children included", () => {
    expect(countIncluded(presetForm(preset))).toBe(2 + 2 + 4)
  })
})

describe("offersPreset", () => {
  const cases: { name: string; lists: [unknown[] | undefined, unknown[] | undefined, unknown[] | undefined]; want: boolean }[] = [
    { name: "all three empty", lists: [[], [], []], want: true },
    { name: "a currency exists", lists: [[{}], [], []], want: false },
    { name: "a category exists", lists: [[], [{}], []], want: false },
    { name: "a tag exists", lists: [[], [], [{}]], want: false },
    { name: "a list is still loading", lists: [[], undefined, []], want: false },
  ]

  for (const { name, lists, want } of cases) {
    it(name, () => expect(offersPreset(...lists)).toBe(want))
  }
})
