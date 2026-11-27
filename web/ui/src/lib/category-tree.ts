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
// of a category of kind: open ones of the same kind. When moving an existing
// category, pass it as moving: it cannot go under itself or a category under
// it.
export function parentChoices(categories: Category[], kind: Category["kind"], moving?: Category): CategoryRow[] {
  const excluded = moving ? withDescendants(categories, moving.id) : new Set<number>()
  return categoryRows(
    categories.filter((category) => !category.closed && category.kind === kind && !excluded.has(category.id)),
  )
}

// A Replacement is a category the entries of a deleted one can take, named
// with the categories above it, such as "Food › Groceries".
export type Replacement = { id: number; label: string }

// replacementChoices lists the categories deleting can be merged into: the
// other open ones of its kind with no category under them, sorted by label.
export function replacementChoices(categories: Category[], deleting: Category): Replacement[] {
  const byID = new Map(categories.map((category) => [category.id, category]))
  const parents = new Set(categories.map((category) => category.parentId))
  const label = (category: Category): string => {
    const parent = category.parentId === undefined ? undefined : byID.get(category.parentId)
    return parent ? `${label(parent)} › ${category.name}` : category.name
  }
  return categories
    .filter(
      (category) =>
        !category.closed && category.kind === deleting.kind && category.id !== deleting.id && !parents.has(category.id),
    )
    .map((category) => ({ id: category.id, label: label(category) }))
    .sort((a, b) => a.label.localeCompare(b.label))
}

// describeCategoryDeletion says what deleting a category does to the entries
// filed under it, which take replacement when one is chosen.
export function describeCategoryDeletion(entries: number, replacement?: string): string {
  if (entries === 0) {
    return "No journal entry is filed under it."
  }
  const count = entries === 1 ? "1 journal entry" : `${entries} journal entries`
  if (replacement !== undefined) {
    return `${count} ${entries === 1 ? "moves" : "move"} to ${replacement}.`
  }
  return entries === 1 ? `${count} loses its category.` : `${count} lose their category.`
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
