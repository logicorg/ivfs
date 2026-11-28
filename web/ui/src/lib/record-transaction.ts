import * as z from "zod"

import type { AccountBalances, Category, ExpenseForm, FinancialAccount, IncomeForm, SplitForm, TransferForm } from "@/api"

import { categoryRows } from "./category-tree"
import { addDecimals, compareDecimals, negate } from "./decimal"
import { amountText, startOfDay } from "./open-account"
import { sentTagIds, tagIdsField } from "./tags"
import { decimal, positiveText } from "./trade"

export const transactionTypes = ["expense", "income", "transfer"] as const

export type TransactionType = (typeof transactionTypes)[number]

// recordTransactionForm is the form as the person fills it in. The money
// moves from fromId to toId, ids as the selects hold them: a balance's ledger
// account to a category for an expense, a category to a balance for an
// income, and a balance to another for a transfer. splits are parts of the
// amount booked on expense categories of their own, the other lines of an
// expense or the charges of a transfer, and toId gets the rest, so they take
// less than the amount. The comparison waits for every amount to be a number;
// their own checks report them otherwise.
export const recordTransactionForm = z
  .object({
    type: z.enum(transactionTypes),
    fromId: z.string().min(1, "Choose one"),
    toId: z.string().min(1, "Choose one"),
    amount: amountText.min(1, "The amount, such as 12.50"),
    toAmount: amountText,
    occurredOn: z.string(),
    description: z.string().trim(),
    tagIds: tagIdsField,
    splits: z.array(z.object({ categoryId: z.string().min(1, "Choose one"), amount: positiveText("4.50") })),
  })
  .superRefine((form, context) => {
    const amounts = [form.amount, ...form.splits.map((split) => split.amount)]
    if (form.splits.length > 0 && amounts.every((amount) => decimal.test(amount)) && compareDecimals(splitsTotal(form.splits), form.amount) >= 0) {
      context.addIssue({ code: "custom", message: "More than the parts split off it", path: ["amount"] })
    }
  })

// A SplitValues is a split as its row holds it.
export type SplitValues = { categoryId: string; amount: string }

// splitsTotal is what the splits take of the amount.
function splitsTotal(splits: SplitValues[]): string {
  return splits.reduce((total, split) => addDecimals(total, split.amount), "0")
}

// splitForms are the splits as the gateway takes them.
function splitForms(splits: SplitValues[]): SplitForm[] {
  return splits.map((split) => ({ categoryId: Number(split.categoryId), amount: split.amount }))
}

export type RecordTransactionFormValues = z.infer<typeof recordTransactionForm>

// A Choice is what a picker offers: a balance, by its ledger account's id and
// with its currency, or a category, by its own id and with none. depth
// indents a category inside its tree.
export type Choice = { id: number; label: string; currencyCode?: string; depth?: number }

// A Picker is one side of the transaction: what it is called and what it
// offers.
export type Picker = { label: string; choices: Choice[] }

// balanceChoices lists the open balances of the open financial accounts,
// sorted by institution, account, then balance. balances holds what each
// account's balances endpoint returned, in any order. The institution leads the
// label because accounts at different institutions can share a name. A balance
// named as its account is labelled once.
export function balanceChoices(accounts: FinancialAccount[], balances: AccountBalances[]): Choice[] {
  const byAccount = new Map(balances.map((account) => [account.financialAccountId, account.balances]))
  const open = accounts
    .filter((account) => !account.closed)
    .sort((a, b) => a.institutionName.localeCompare(b.institutionName) || a.name.localeCompare(b.name))
  return open.flatMap((account) =>
    (byAccount.get(account.id) ?? [])
      .filter((balance) => !balance.closed && (balance.kind === "asset" || balance.kind === "liability"))
      .sort((a, b) => a.ledgerAccountName.localeCompare(b.ledgerAccountName))
      .map((balance) => ({
        id: balance.ledgerAccountId,
        label:
          balance.ledgerAccountName === account.name
            ? `${account.institutionName} · ${account.name}`
            : `${account.institutionName} · ${account.name} · ${balance.ledgerAccountName}`,
        currencyCode: balance.currencyCode,
      })),
  )
}

// categoryChoices lists the open categories of kind in tree order.
export function categoryChoices(categories: Category[], kind: Category["kind"]): Choice[] {
  const offered = categories.filter((category) => !category.closed && category.kind === kind)
  return categoryRows(offered).map(({ category, depth }) => ({ id: category.id, label: category.name, depth }))
}

// pickers are the two sides the type asks for. A category is offered whatever
// the balance's currency, and a transfer does not offer its source as its
// destination.
export function pickers(
  type: TransactionType,
  balances: Choice[],
  categories: Category[],
  fromId: string,
): { from: Picker; to: Picker } {
  switch (type) {
    case "expense":
      return {
        from: { label: "Paid from", choices: balances },
        to: { label: "Category", choices: categoryChoices(categories, "expense") },
      }
    case "income":
      return {
        from: { label: "Category", choices: categoryChoices(categories, "income") },
        to: { label: "Received into", choices: balances },
      }
    case "transfer":
      return {
        from: { label: "From", choices: balances },
        to: { label: "To", choices: balances.filter((choice) => String(choice.id) !== fromId) },
      }
  }
}

// offers tells whether picker still offers the id chosen on its side, after
// the other side changed what it offers.
export function offers(picker: Picker, id: string): boolean {
  return picker.choices.some((choice) => String(choice.id) === id)
}

// crossesCurrencies tells whether a transfer between the chosen balances
// changes currency, and so needs the amount that arrives.
export function crossesCurrencies(values: RecordTransactionFormValues, balances: Choice[]): boolean {
  if (values.type !== "transfer") {
    return false
  }
  const currencyOf = (id: string) => balances.find((choice) => String(choice.id) === id)?.currencyCode
  const from = currencyOf(values.fromId)
  const to = currencyOf(values.toId)
  return from !== undefined && to !== undefined && from !== to
}

// A TransactionRequest is the endpoint to call, by type, and its body.
export type TransactionRequest =
  | { type: "expense"; body: ExpenseForm }
  | { type: "income"; body: IncomeForm }
  | { type: "transfer"; body: TransferForm }

// newTransaction is values as the gateway takes them. The money moves at the
// start of the chosen day, in the browser's time zone, or at now when no day
// is chosen. The amount that arrives is sent only between currencies. An
// expense with splits is sent as lines, its category's first with the rest of
// the amount; a transfer's splits are its charges; an income takes none.
export function newTransaction(
  values: RecordTransactionFormValues,
  idempotencyKey: string,
  crossing: boolean,
  now: Date,
): TransactionRequest {
  const common = {
    idempotencyKey,
    amount: values.amount,
    occurredAt: startOfDay(values.occurredOn) ?? now.toISOString(),
    description: values.description || undefined,
    tagIds: sentTagIds(values.tagIds),
  }
  const from = Number(values.fromId)
  const to = Number(values.toId)
  switch (values.type) {
    case "expense":
      if (values.splits.length > 0) {
        const rest = addDecimals(values.amount, negate(splitsTotal(values.splits)))
        return {
          type: "expense",
          body: { ...common, balanceLedgerAccountId: from, lines: [{ categoryId: to, amount: rest }, ...splitForms(values.splits)] },
        }
      }
      return { type: "expense", body: { ...common, balanceLedgerAccountId: from, categoryId: to } }
    case "income":
      return { type: "income", body: { ...common, balanceLedgerAccountId: to, categoryId: from } }
    case "transfer":
      return {
        type: "transfer",
        body: {
          ...common,
          fromLedgerAccountId: from,
          toLedgerAccountId: to,
          toAmount: crossing ? values.toAmount : undefined,
          charges: values.splits.length > 0 ? splitForms(values.splits) : undefined,
        },
      }
  }
}
