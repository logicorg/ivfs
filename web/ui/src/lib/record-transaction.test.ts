import { describe, expect, it } from "vitest"

import type { AccountBalances, Balance, Category, FinancialAccount } from "@/api"

import {
  balanceChoices,
  categoryChoices,
  crossesCurrencies,
  newTransaction,
  offers,
  pickers,
  recordTransactionForm,
  type Choice,
  type RecordTransactionFormValues,
  type TransactionRequest,
} from "./record-transaction"

function account(id: number, name: string, closed = false): FinancialAccount {
  return { id, name, closed, institutionId: 1, institutionName: "Revolut", institutionClosed: false }
}

function balance(ledgerAccountId: number, ledgerAccountName: string, currencyCode: string, extra?: Partial<Balance>): Balance {
  return {
    ledgerAccountId,
    ledgerAccountName,
    currencyCode,
    currencyId: currencyCode === "EUR" ? 1 : 2,
    kind: "asset",
    balance: "0",
    closed: false,
    ...extra,
  }
}

function category(id: number, name: string, kind: Category["kind"], currencyCode: string, extra?: Partial<Category>): Category {
  return { id, name, kind, currencyCode, currencyId: currencyCode === "EUR" ? 1 : 2, closed: false, ...extra }
}

const groceries: RecordTransactionFormValues = {
  type: "expense",
  fromId: "100",
  toId: "200",
  amount: "12.50",
  toAmount: "",
  occurredOn: "",
  description: "",
}

describe("recordTransactionForm", () => {
  const cases: { name: string; values: RecordTransactionFormValues; wantErrors: string[] }[] = [
    { name: "an expense", values: groceries, wantErrors: [] },
    { name: "no source chosen", values: { ...groceries, fromId: "" }, wantErrors: ["fromId"] },
    { name: "no destination chosen", values: { ...groceries, toId: "" }, wantErrors: ["toId"] },
    { name: "no amount", values: { ...groceries, amount: " " }, wantErrors: ["amount"] },
    { name: "an amount with a thousands separator", values: { ...groceries, amount: "1,200" }, wantErrors: ["amount"] },
    {
      name: "an amount that arrives with a decimal comma",
      values: { ...groceries, type: "transfer", toAmount: "324,15" },
      wantErrors: ["toAmount"],
    },
  ]

  for (const { name, values, wantErrors } of cases) {
    it(name, () => {
      const result = recordTransactionForm.safeParse(values)
      expect(result.error?.issues.map((issue) => issue.path.join(".")) ?? []).toEqual(wantErrors)
    })
  }
})

describe("balanceChoices", () => {
  const cases: { name: string; accounts: FinancialAccount[]; balances: AccountBalances[]; want: Choice[] }[] = [
    {
      name: "sorted by account then balance, a balance named as its account labelled once",
      accounts: [account(2, "Revolut"), account(1, "Cash")],
      balances: [
        { financialAccountId: 2, balances: [balance(102, "Vault", "EUR"), balance(101, "Revolut", "EUR")] },
        { financialAccountId: 1, balances: [balance(100, "Cash", "EUR")] },
      ],
      want: [
        { ledgerAccountId: 100, label: "Cash", currencyCode: "EUR" },
        { ledgerAccountId: 101, label: "Revolut", currencyCode: "EUR" },
        { ledgerAccountId: 102, label: "Revolut · Vault", currencyCode: "EUR" },
      ],
    },
    {
      name: "a card's balance that is owed",
      accounts: [account(3, "Amex")],
      balances: [{ financialAccountId: 3, balances: [balance(103, "Amex", "USD", { kind: "liability" })] }],
      want: [{ ledgerAccountId: 103, label: "Amex", currencyCode: "USD" }],
    },
    {
      name: "closed accounts and balances left out",
      accounts: [account(1, "Cash"), account(2, "Old bank", true)],
      balances: [
        { financialAccountId: 1, balances: [balance(100, "Cash", "EUR"), balance(104, "Old vault", "EUR", { closed: true })] },
        { financialAccountId: 2, balances: [balance(105, "Old bank", "EUR")] },
      ],
      want: [{ ledgerAccountId: 100, label: "Cash", currencyCode: "EUR" }],
    },
    {
      name: "an account whose balances have not loaded",
      accounts: [account(1, "Cash")],
      balances: [],
      want: [],
    },
  ]

  for (const { name, accounts, balances, want } of cases) {
    it(name, () => {
      expect(balanceChoices(accounts, balances)).toEqual(want)
    })
  }
})

describe("categoryChoices", () => {
  const categories = [
    category(201, "Restaurants", "expense", "EUR", { parentId: 200 }),
    category(200, "Food", "expense", "EUR"),
    category(202, "Travel", "expense", "USD"),
    category(203, "Old hobby", "expense", "EUR", { closed: true }),
    category(210, "Salary", "income", "EUR"),
  ]
  const cases: { name: string; kind: Category["kind"]; currencyCode?: string; want: Choice[] }[] = [
    {
      name: "open ones of the kind, in tree order",
      kind: "expense",
      want: [
        { ledgerAccountId: 200, label: "Food", currencyCode: "EUR", depth: 0 },
        { ledgerAccountId: 201, label: "Restaurants", currencyCode: "EUR", depth: 1 },
        { ledgerAccountId: 202, label: "Travel", currencyCode: "USD", depth: 0 },
      ],
    },
    {
      name: "only those in the currency given",
      kind: "expense",
      currencyCode: "USD",
      want: [{ ledgerAccountId: 202, label: "Travel", currencyCode: "USD", depth: 0 }],
    },
    {
      name: "income ones",
      kind: "income",
      want: [{ ledgerAccountId: 210, label: "Salary", currencyCode: "EUR", depth: 0 }],
    },
  ]

  for (const { name, kind, currencyCode, want } of cases) {
    it(name, () => {
      expect(categoryChoices(categories, kind, currencyCode)).toEqual(want)
    })
  }
})

describe("pickers", () => {
  const cash: Choice = { ledgerAccountId: 100, label: "Cash", currencyCode: "EUR" }
  const dollars: Choice = { ledgerAccountId: 103, label: "Cash (USD)", currencyCode: "USD" }
  const categories = [
    category(200, "Food", "expense", "EUR"),
    category(202, "Travel", "expense", "USD"),
    category(210, "Salary", "income", "EUR"),
  ]
  const food: Choice = { ledgerAccountId: 200, label: "Food", currencyCode: "EUR", depth: 0 }
  const travel: Choice = { ledgerAccountId: 202, label: "Travel", currencyCode: "USD", depth: 0 }
  const salary: Choice = { ledgerAccountId: 210, label: "Salary", currencyCode: "EUR", depth: 0 }

  const cases: {
    name: string
    type: RecordTransactionFormValues["type"]
    fromId: string
    toId: string
    want: ReturnType<typeof pickers>
  }[] = [
    {
      name: "an expense, before a balance is chosen",
      type: "expense",
      fromId: "",
      toId: "",
      want: { from: { label: "Paid from", choices: [cash, dollars] }, to: { label: "Category", choices: [food, travel] } },
    },
    {
      name: "an expense from a dollar balance",
      type: "expense",
      fromId: "103",
      toId: "",
      want: { from: { label: "Paid from", choices: [cash, dollars] }, to: { label: "Category", choices: [travel] } },
    },
    {
      name: "an income into a euro balance",
      type: "income",
      fromId: "",
      toId: "100",
      want: { from: { label: "Category", choices: [salary] }, to: { label: "Received into", choices: [cash, dollars] } },
    },
    {
      name: "a transfer, its source not offered as its destination",
      type: "transfer",
      fromId: "100",
      toId: "",
      want: { from: { label: "From", choices: [cash, dollars] }, to: { label: "To", choices: [dollars] } },
    },
  ]

  for (const { name, type, fromId, toId, want } of cases) {
    it(name, () => {
      expect(pickers(type, [cash, dollars], categories, fromId, toId)).toEqual(want)
    })
  }
})

describe("offers", () => {
  const picker = {
    label: "Category",
    choices: [
      { ledgerAccountId: 200, label: "Food", currencyCode: "EUR" },
      { ledgerAccountId: 201, label: "Rent", currencyCode: "EUR" },
    ],
  }
  const cases: { name: string; id: string; want: boolean }[] = [
    { name: "a choice it offers", id: "201", want: true },
    { name: "a choice it no longer offers", id: "202", want: false },
    { name: "no choice", id: "", want: false },
  ]

  for (const { name, id, want } of cases) {
    it(name, () => {
      expect(offers(picker, id)).toBe(want)
    })
  }
})

describe("crossesCurrencies", () => {
  const balances: Choice[] = [
    { ledgerAccountId: 100, label: "Cash", currencyCode: "EUR" },
    { ledgerAccountId: 101, label: "Revolut", currencyCode: "EUR" },
    { ledgerAccountId: 103, label: "Cash (USD)", currencyCode: "USD" },
  ]
  const cases: { name: string; values: RecordTransactionFormValues; want: boolean }[] = [
    { name: "a transfer into dollars", values: { ...groceries, type: "transfer", fromId: "100", toId: "103" }, want: true },
    { name: "a transfer in euros", values: { ...groceries, type: "transfer", fromId: "100", toId: "101" }, want: false },
    { name: "a transfer with no destination yet", values: { ...groceries, type: "transfer", fromId: "100", toId: "" }, want: false },
    { name: "an expense", values: { ...groceries, fromId: "103", toId: "200" }, want: false },
  ]

  for (const { name, values, want } of cases) {
    it(name, () => {
      expect(crossesCurrencies(values, balances)).toBe(want)
    })
  }
})

describe("newTransaction", () => {
  const now = new Date("2026-10-04T08:30:00Z")
  const cases: { name: string; values: RecordTransactionFormValues; crossing: boolean; want: TransactionRequest }[] = [
    {
      name: "an expense, now, with no note",
      values: groceries,
      crossing: false,
      want: {
        type: "expense",
        body: {
          idempotencyKey: "3f2a",
          balanceLedgerAccountId: 100,
          categoryLedgerAccountId: 200,
          amount: "12.50",
          occurredAt: "2026-10-04T08:30:00.000Z",
        },
      },
    },
    {
      name: "an income on a day, with a note",
      values: { ...groceries, type: "income", fromId: "210", toId: "100", amount: "2500", occurredOn: "2026-09-30", description: "September salary" },
      crossing: false,
      want: {
        type: "income",
        body: {
          idempotencyKey: "3f2a",
          balanceLedgerAccountId: 100,
          categoryLedgerAccountId: 210,
          amount: "2500",
          occurredAt: new Date(2026, 8, 30).toISOString(),
          description: "September salary",
        },
      },
    },
    {
      name: "a transfer in one currency, an amount that arrives left out",
      values: { ...groceries, type: "transfer", fromId: "100", toId: "101", amount: "300.50", toAmount: "300.50" },
      crossing: false,
      want: {
        type: "transfer",
        body: {
          idempotencyKey: "3f2a",
          fromLedgerAccountId: 100,
          toLedgerAccountId: 101,
          amount: "300.50",
          occurredAt: "2026-10-04T08:30:00.000Z",
        },
      },
    },
    {
      name: "a transfer into dollars",
      values: { ...groceries, type: "transfer", fromId: "100", toId: "103", amount: "300.50", toAmount: "324.15" },
      crossing: true,
      want: {
        type: "transfer",
        body: {
          idempotencyKey: "3f2a",
          fromLedgerAccountId: 100,
          toLedgerAccountId: 103,
          amount: "300.50",
          toAmount: "324.15",
          occurredAt: "2026-10-04T08:30:00.000Z",
        },
      },
    },
  ]

  for (const { name, values, crossing, want } of cases) {
    it(name, () => {
      expect(newTransaction(values, "3f2a", crossing, now)).toEqual(want)
    })
  }
})
