import type { InsertCurrencyParams, Preset, PresetCategory, PresetTag } from "@/api"

// Each row of the preset form has a key for React and a box the user can
// untick. An unticked row stays on the form, so ticking it again restores
// what was typed.
type Row = { key: string; included: boolean }

export type CurrencyRow = Row & InsertCurrencyParams
export type TagRow = Row & PresetTag
export type CategoryRow = Row & { name: string; kind?: PresetCategory["kind"]; children: CategoryRow[] }

export type PresetForm = {
  currencies: CurrencyRow[]
  tags: TagRow[]
  categories: CategoryRow[]
}

let lastKey = 0

// newKey is a key no other row has.
export function newKey(): string {
  lastKey += 1
  return `row-${lastKey}`
}

// presetForm is a preset as the form shows it, every row ticked.
export function presetForm(preset: Preset): PresetForm {
  return {
    currencies: preset.currencies.map((currency) => ({ ...currency, key: newKey(), included: true })),
    tags: preset.tags.map((tag) => ({ ...tag, key: newKey(), included: true })),
    categories: preset.categories.map(categoryRow),
  }
}

function categoryRow(category: PresetCategory): CategoryRow {
  return {
    key: newKey(),
    included: true,
    name: category.name,
    kind: category.kind,
    children: (category.children ?? []).map(categoryRow),
  }
}

// presetOf is what the form applies: the ticked rows with their names
// trimmed. Unticking a category leaves its children out too, since they
// cannot exist without it.
export function presetOf(form: PresetForm): Preset {
  return {
    currencies: form.currencies
      .filter((row) => row.included)
      .map(({ code, name, symbol, precision }) => ({
        code: code.trim(),
        name: name.trim(),
        ...(symbol?.trim() ? { symbol: symbol.trim() } : {}),
        ...(precision !== undefined ? { precision } : {}),
      })),
    tags: form.tags
      .filter((row) => row.included)
      .map(({ name, description }) => ({ name: name.trim(), ...(description ? { description } : {}) })),
    categories: categoriesOf(form.categories),
  }
}

function categoriesOf(rows: CategoryRow[]): PresetCategory[] {
  return rows
    .filter((row) => row.included)
    .map((row) => {
      const children = categoriesOf(row.children)
      return {
        name: row.name.trim(),
        ...(row.kind ? { kind: row.kind } : {}),
        ...(children.length > 0 ? { children } : {}),
      }
    })
}

// countIncluded is how many items applying the form creates.
export function countIncluded(form: PresetForm): number {
  const preset = presetOf(form)
  const categories = (list: PresetCategory[]): number =>
    list.reduce((n, category) => n + 1 + categories(category.children ?? []), 0)
  return preset.currencies.length + preset.tags.length + categories(preset.categories)
}

// updateCategory returns rows with the category of the given key changed by
// change, wherever it is in the tree.
export function updateCategory(
  rows: CategoryRow[],
  key: string,
  change: (row: CategoryRow) => CategoryRow,
): CategoryRow[] {
  return rows.map((row) =>
    row.key === key ? change(row) : { ...row, children: updateCategory(row.children, key, change) },
  )
}

// offersPreset says whether the user has nothing yet, which is when the
// preset is offered. It is false until all three lists have loaded.
export function offersPreset(
  currencies: unknown[] | undefined,
  categories: unknown[] | undefined,
  tags: unknown[] | undefined,
): boolean {
  return currencies?.length === 0 && categories?.length === 0 && tags?.length === 0
}
