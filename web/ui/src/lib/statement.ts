import type { Balance, Entry, Posting } from "@/api"

type Kind = Balance["kind"]

// A StatementLine is one posting on the statement's ledger account, with the
// entry it belongs to.
export type StatementLine = {
  // key is unique within a statement, since an entry may post to the same
  // ledger account more than once.
  key: string
  entryId: number
  occurredAt: string
  description?: string
  reversalOfEntryId?: number
  // otherAccounts names the ledger accounts on the entry's other side, such
  // as Groceries for a payment from Cash.
  otherAccounts: string[]
  // amount is what the posting adds to the account's balance in its natural
  // direction, as the balance itself is shown: negative when it takes away.
  amount: string
}

// statementLines lists, for each entry in order, its postings on the ledger
// account ledgerAccountId of the given kind.
export function statementLines(entries: Entry[], ledgerAccountId: number, kind: Kind): StatementLine[] {
  return entries.flatMap((entry) => {
    const others = entry.postings.filter((posting) => posting.ledgerAccountId !== ledgerAccountId)
    const otherAccounts = [...new Set(others.map((posting) => posting.ledgerAccountName))]
    return entry.postings
      .filter((posting) => posting.ledgerAccountId === ledgerAccountId)
      .map((posting) => ({
        key: `${entry.id}-${posting.lineNumber}`,
        entryId: entry.id,
        occurredAt: entry.occurredAt,
        description: entry.description,
        reversalOfEntryId: entry.reversalOfEntryId,
        otherAccounts,
        amount: signedAmount(posting, kind),
      }))
  })
}

// signedAmount is what posting adds to the natural balance of an account of
// kind: an asset or an expense grows with a debit, the others with a credit,
// as the gateway's balances count them.
export function signedAmount(posting: Pick<Posting, "amount" | "side">, kind: Kind): string {
  const debitNormal = kind === "asset" || kind === "expense"
  const grows = (posting.side === "debit") === debitNormal
  return grows ? posting.amount : negate(posting.amount)
}

// negate negates a decimal string without turning it into a number.
function negate(amount: string): string {
  if (amount.startsWith("-")) {
    return amount.slice(1)
  }
  return /^[0.]+$/.test(amount) ? amount : `-${amount}`
}
