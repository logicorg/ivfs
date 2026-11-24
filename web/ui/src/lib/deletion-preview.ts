import type { AccountDeletionPreview } from "@/api"
import { formatAmount } from "@/lib/money"

// describeDeletionPreview says what deleting an account would do, a sentence
// a line, such as "Deletes 2 journal entries." The conflicts that would
// refuse it are not among them.
export function describeDeletionPreview(preview: AccountDeletionPreview, locale?: string): string[] {
  const deleted = countOf(preview.deletedEntries, preview.deletedMovementGroups)
  const kept = countOf(preview.keptEntries, preview.keptMovementGroups)
  if (!deleted && !kept) {
    return ["Nothing is recorded on it."]
  }

  const lines: string[] = []
  if (deleted) {
    lines.push(`Deletes ${deleted}.`)
  }
  if (kept) {
    lines.push(`Keeps ${kept}: each also touches an account that stays, so only this side moves to the untracked accounts.`)
  }
  for (const amount of preview.strayAmounts ?? []) {
    lines.push(`Untracked accounts receives ${formatAmount(amount.amount, amount.currencyCode, undefined, locale)}.`)
  }
  for (const units of preview.strayUnits ?? []) {
    lines.push(`Untracked accounts receives ${units.units} ${units.assetName}.`)
  }
  return lines
}

function countOf(entries: number, groups: number): string {
  const parts: string[] = []
  if (entries > 0) {
    parts.push(entries === 1 ? "1 journal entry" : `${entries} journal entries`)
  }
  if (groups > 0) {
    parts.push(groups === 1 ? "1 movement group" : `${groups} movement groups`)
  }
  return parts.join(" and ")
}
