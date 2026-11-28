import type { EntryTransaction, Expense, Income, Transfer } from "@/api"

import { newTransaction, type RecordTransactionFormValues } from "./record-transaction"

// localDay is the day an instant falls on in the browser's time zone, such as
// 2026-10-03, as a date input holds it.
export function localDay(instant: string): string {
  const date = new Date(instant)
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

// formValuesOf is an entry's transaction as the form shows it: an expense's
// first line is its category and the others its splits, and a transfer's
// charges are its splits.
export function formValuesOf(transaction: EntryTransaction): RecordTransactionFormValues | undefined {
  const common = (t: Expense | Income | Transfer) => ({
    amount: t.amount,
    toAmount: "",
    occurredOn: localDay(t.occurredAt),
    description: t.description ?? "",
    tagIds: t.tagIds ?? [],
    splits: [],
  })
  const splits = (forms: { categoryId: number; amount: string }[] = []) =>
    forms.map((form) => ({ categoryId: String(form.categoryId), amount: form.amount }))

  if (transaction.expense) {
    const { expense } = transaction
    const [first, ...others] = expense.lines ?? []
    return {
      ...common(expense),
      type: "expense",
      fromId: String(expense.balanceLedgerAccountId),
      toId: String(expense.categoryId ?? first?.categoryId ?? ""),
      splits: splits(others),
    }
  }
  if (transaction.income) {
    const { income } = transaction
    return {
      ...common(income),
      type: "income",
      fromId: String(income.categoryId),
      toId: String(income.balanceLedgerAccountId),
    }
  }
  if (transaction.transfer) {
    const { transfer } = transaction
    return {
      ...common(transfer),
      type: "transfer",
      fromId: String(transfer.fromLedgerAccountId),
      toId: String(transfer.toLedgerAccountId),
      toAmount: transfer.toAmount ?? "",
      splits: splits(transfer.charges),
    }
  }
  return undefined
}

// A TransactionCorrection is the endpoint that corrects an entry, by type,
// and its body.
export type TransactionCorrection =
  | { type: "expense"; body: Expense }
  | { type: "income"; body: Income }
  | { type: "transfer"; body: Transfer }

// correctionOf is values as the endpoints that correct an entry take them.
// The entry keeps the instant it occurred at unless its day is changed, which
// moves it to the start of the new day, as recording on a day does.
export function correctionOf(values: RecordTransactionFormValues, crossing: boolean, occurredAt: string): TransactionCorrection {
  const sameDay = values.occurredOn === "" || values.occurredOn === localDay(occurredAt)
  const kept = new Date(occurredAt)
  const request = newTransaction({ ...values, occurredOn: sameDay ? "" : values.occurredOn }, "", crossing, kept)
  const { idempotencyKey: _key, ...body } = request.body
  return { type: request.type, body } as TransactionCorrection
}
