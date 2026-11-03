import type { AccountDeletionPreview, StrayAmount } from "@/api"
import { negate } from "@/lib/decimal"
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
    lines.push(`${amount.name} (${amount.kind}) receives ${formatAmount(strayAmount(amount), amount.currencyCode, undefined, locale)}.`)
  }
  for (const units of preview.strayUnits ?? []) {
    lines.push(`Untracked accounts receives ${units.units} ${units.assetName}.`)
  }
  return lines
}

// strayAmount is what an untracked account receives in the sense its kind
// reads: earned for income, debits minus credits otherwise, so a moved balance
// reads as it did on the account.
function strayAmount(amount: StrayAmount): string {
  return amount.kind === "income" ? negate(amount.amount) : amount.amount
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
