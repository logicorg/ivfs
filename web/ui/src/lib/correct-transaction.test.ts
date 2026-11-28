import { describe, expect, it } from "vitest"

import type { EntryTransaction } from "@/api"

import { correctionOf, formValuesOf, localDay, type TransactionCorrection } from "./correct-transaction"
import type { RecordTransactionFormValues } from "./record-transaction"

// Instants are built in the browser's time zone, so the day they fall on is
// the same wherever the tests run.
const morning = new Date(2026, 9, 3, 10, 0).toISOString()

const groceries: RecordTransactionFormValues = {
  type: "expense",
  fromId: "100",
  toId: "200",
  amount: "12.50",
  toAmount: "",
  occurredOn: "2026-10-03",
  description: "",
  tagIds: [],
  splits: [],
}

describe("localDay", () => {
  const cases = [
    { name: "a morning", instant: morning, want: "2026-10-03" },
    { name: "the first minute of a day", instant: new Date(2026, 0, 1, 0, 0).toISOString(), want: "2026-01-01" },
    { name: "the last minute of a day", instant: new Date(2026, 11, 31, 23, 59).toISOString(), want: "2026-12-31" },
  ]

  for (const { name, instant, want } of cases) {
    it(name, () => {
      expect(localDay(instant)).toEqual(want)
    })
  }
})

describe("formValuesOf", () => {
  const cases: { name: string; transaction: EntryTransaction; want: RecordTransactionFormValues | undefined }[] = [
    {
      name: "an expense",
      transaction: { expense: { balanceLedgerAccountId: 100, categoryId: 200, amount: "12.50", occurredAt: morning } },
      want: groceries,
    },
    {
      name: "an expense in lines, the first its category",
      transaction: {
        expense: {
          balanceLedgerAccountId: 100,
          amount: "12.50",
          occurredAt: morning,
          description: "Esselunga",
          tagIds: [5],
          lines: [
            { categoryId: 200, amount: "8" },
            { categoryId: 201, amount: "4.50" },
          ],
        },
      },
      want: { ...groceries, description: "Esselunga", tagIds: [5], splits: [{ categoryId: "201", amount: "4.50" }] },
    },
    {
      name: "an income",
      transaction: { income: { balanceLedgerAccountId: 100, categoryId: 210, amount: "2500", occurredAt: morning } },
      want: { ...groceries, type: "income", fromId: "210", toId: "100", amount: "2500" },
    },
    {
      name: "an exchange with a fee",
      transaction: {
        transfer: {
          fromLedgerAccountId: 100,
          toLedgerAccountId: 103,
          amount: "300.50",
          toAmount: "324.15",
          occurredAt: morning,
          charges: [{ categoryId: 220, amount: "1.50" }],
        },
      },
      want: {
        ...groceries,
        type: "transfer",
        toId: "103",
        amount: "300.50",
        toAmount: "324.15",
        splits: [{ categoryId: "220", amount: "1.50" }],
      },
    },
    { name: "no transaction", transaction: {}, want: undefined },
  ]

  for (const { name, transaction, want } of cases) {
    it(name, () => {
      expect(formValuesOf(transaction)).toEqual(want)
    })
  }
})

describe("correctionOf", () => {
  const expense = (occurredAt: string, extra?: object): TransactionCorrection => ({
    type: "expense",
    body: { balanceLedgerAccountId: 100, categoryId: 200, amount: "12.50", occurredAt, ...extra },
  })
  const cases: { name: string; values: RecordTransactionFormValues; crossing: boolean; want: TransactionCorrection }[] = [
    { name: "the same day keeps the instant", values: groceries, crossing: false, want: expense(morning) },
    { name: "no day keeps the instant", values: { ...groceries, occurredOn: "" }, crossing: false, want: expense(morning) },
    {
      name: "another day moves it to that day's start",
      values: { ...groceries, occurredOn: "2026-10-05" },
      crossing: false,
      want: expense(new Date(2026, 9, 5).toISOString()),
    },
    {
      name: "an expense split, its category taking the rest",
      values: { ...groceries, splits: [{ categoryId: "201", amount: "4.50" }] },
      crossing: false,
      want: {
        type: "expense",
        body: {
          balanceLedgerAccountId: 100,
          amount: "12.50",
          occurredAt: morning,
          lines: [
            { categoryId: 200, amount: "8" },
            { categoryId: 201, amount: "4.50" },
          ],
        },
      },
    },
    {
      name: "an exchange",
      values: { ...groceries, type: "transfer", toId: "103", amount: "300.50", toAmount: "324.15" },
      crossing: true,
      want: {
        type: "transfer",
        body: { fromLedgerAccountId: 100, toLedgerAccountId: 103, amount: "300.50", toAmount: "324.15", occurredAt: morning },
      },
    },
  ]

  for (const { name, values, crossing, want } of cases) {
    it(name, () => {
      expect(correctionOf(values, crossing, morning)).toEqual(want)
    })
  }
})
