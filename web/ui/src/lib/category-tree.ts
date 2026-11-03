import type { Category } from "@/api"

// A CategoryRow is a category in tree order, with how deep it is.
export type CategoryRow = { category: Category; depth: number }

// categoryRows lists categories in tree order: each one is followed by the
// categories under it, and siblings are sorted by name. A category whose
// parent is not in the list, such as one under a closed category that is
// left out, is listed as top level.
export function categoryRows(categories: Category[]): CategoryRow[] {
  const ids = new Set(categories.map((category) => category.id))
  const children = new Map<number | undefined, Category[]>()
  for (const category of categories) {
    const parent = category.parentId !== undefined && ids.has(category.parentId) ? category.parentId : undefined
    children.set(parent, [...(children.get(parent) ?? []), category])
  }

  const rows: CategoryRow[] = []
  const visit = (parent: number | undefined, depth: number) => {
    const siblings = [...(children.get(parent) ?? [])].sort((a, b) => a.name.localeCompare(b.name))
    for (const category of siblings) {
      rows.push({ category, depth })
      visit(category.id, depth + 1)
    }
  }
  visit(undefined, 0)
  return rows
}

// parentChoices lists, in tree order, the categories that can be the parent
// of a category of kind and currencyId: open ones of the same kind and
// currency. When moving an existing category, pass it as moving: it cannot go
// under itself or a category under it.
export function parentChoices(
  categories: Category[],
  kind: Category["kind"],
  currencyId: number,
  moving?: Category,
): CategoryRow[] {
  const excluded = moving ? withDescendants(categories, moving.id) : new Set<number>()
  return categoryRows(
    categories.filter(
      (category) =>
        !category.closed && category.kind === kind && category.currencyId === currencyId && !excluded.has(category.id),
    ),
  )
}

function withDescendants(categories: Category[], id: number): Set<number> {
  const found = new Set([id])
  let grew = true
  while (grew) {
    grew = false
    for (const category of categories) {
      if (category.parentId !== undefined && found.has(category.parentId) && !found.has(category.id)) {
        found.add(category.id)
        grew = true
      }
    }
  }
  return found
}
