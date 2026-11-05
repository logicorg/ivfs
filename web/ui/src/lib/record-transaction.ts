import * as z from "zod"

import type { AccountBalances, Category, ExpenseForm, FinancialAccount, IncomeForm, TransferForm } from "@/api"

import { categoryRows } from "./category-tree"
import { amountText, startOfDay } from "./open-account"

export const transactionTypes = ["expense", "income", "transfer"] as const

export type TransactionType = (typeof transactionTypes)[number]

// recordTransactionForm is the form as the person fills it in. The money
// moves from fromId to toId, both ledger account ids as the selects hold
// them: a balance to a category for an expense, a category to a balance for
// an income, and a balance to another for a transfer.
export const recordTransactionForm = z.object({
  type: z.enum(transactionTypes),
  fromId: z.string().min(1, "Choose one"),
  toId: z.string().min(1, "Choose one"),
  amount: amountText.min(1, "The amount, such as 12.50"),
  toAmount: amountText,
  occurredOn: z.string(),
  description: z.string().trim(),
})

export type RecordTransactionFormValues = z.infer<typeof recordTransactionForm>

// A Choice is a ledger account a picker offers, with its currency. depth
// indents a category inside its tree.
export type Choice = { ledgerAccountId: number; label: string; currencyCode: string; depth?: number }

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
        ledgerAccountId: balance.ledgerAccountId,
        label:
          balance.ledgerAccountName === account.name
            ? `${account.institutionName} · ${account.name}`
            : `${account.institutionName} · ${account.name} · ${balance.ledgerAccountName}`,
        currencyCode: balance.currencyCode,
      })),
  )
}

// categoryChoices lists the open categories of kind in tree order, only those
// in currencyCode when one is given.
export function categoryChoices(categories: Category[], kind: Category["kind"], currencyCode?: string): Choice[] {
  const offered = categories.filter(
    (category) =>
      !category.closed &&
      category.kind === kind &&
      (currencyCode === undefined || category.currencyCode === currencyCode),
  )
  return categoryRows(offered).map(({ category, depth }) => ({
    ledgerAccountId: category.id,
    label: category.name,
    currencyCode: category.currencyCode,
    depth,
  }))
}

// pickers are the two sides the type asks for. A category is offered only in
// the currency of the balance already chosen, since a category takes money in
// its own currency only, and a transfer does not offer its source as its
// destination.
export function pickers(
  type: TransactionType,
  balances: Choice[],
  categories: Category[],
  fromId: string,
  toId: string,
): { from: Picker; to: Picker } {
  const currencyOf = (id: string) => balances.find((choice) => String(choice.ledgerAccountId) === id)?.currencyCode
  switch (type) {
    case "expense":
      return {
        from: { label: "Paid from", choices: balances },
        to: { label: "Category", choices: categoryChoices(categories, "expense", currencyOf(fromId)) },
      }
    case "income":
      return {
        from: { label: "Category", choices: categoryChoices(categories, "income", currencyOf(toId)) },
        to: { label: "Received into", choices: balances },
      }
    case "transfer":
      return {
        from: { label: "From", choices: balances },
        to: { label: "To", choices: balances.filter((choice) => String(choice.ledgerAccountId) !== fromId) },
      }
  }
}

// offers tells whether picker still offers the ledger account id chosen on
// its side, after the other side changed what it offers.
export function offers(picker: Picker, id: string): boolean {
  return picker.choices.some((choice) => String(choice.ledgerAccountId) === id)
}

// crossesCurrencies tells whether a transfer between the chosen balances
// changes currency, and so needs the amount that arrives.
export function crossesCurrencies(values: RecordTransactionFormValues, balances: Choice[]): boolean {
  if (values.type !== "transfer") {
    return false
  }
  const currencyOf = (id: string) => balances.find((choice) => String(choice.ledgerAccountId) === id)?.currencyCode
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
// is chosen. The amount that arrives is sent only between currencies.
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
  }
  const from = Number(values.fromId)
  const to = Number(values.toId)
  switch (values.type) {
    case "expense":
      return { type: "expense", body: { ...common, balanceLedgerAccountId: from, categoryLedgerAccountId: to } }
    case "income":
      return { type: "income", body: { ...common, balanceLedgerAccountId: to, categoryLedgerAccountId: from } }
    case "transfer":
      return {
        type: "transfer",
        body: {
          ...common,
          fromLedgerAccountId: from,
          toLedgerAccountId: to,
          toAmount: crossing ? values.toAmount : undefined,
        },
      }
  }
}
