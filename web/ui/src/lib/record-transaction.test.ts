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

function account(id: number, name: string, closed = false, institutionName = "Revolut"): FinancialAccount {
  return { id, name, closed, institutionId: 1, institutionName, institutionClosed: false }
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

function category(id: number, name: string, kind: Category["kind"], extra?: Partial<Category>): Category {
  return { id, name, kind, closed: false, ...extra }
}

const groceries: RecordTransactionFormValues = {
  type: "expense",
  fromId: "100",
  toId: "200",
  amount: "12.50",
  toAmount: "",
  occurredOn: "",
  description: "",
  tagIds: [],
  splits: [],
}

describe("recordTransactionForm", () => {
  const cases: { name: string; values: RecordTransactionFormValues; wantErrors: string[] }[] = [
    { name: "an expense", values: groceries, wantErrors: [] },
    { name: "no source chosen", values: { ...groceries, fromId: "" }, wantErrors: ["fromId"] },
    { name: "no destination chosen", values: { ...groceries, toId: "" }, wantErrors: ["toId"] },
    { name: "no amount", values: { ...groceries, amount: " " }, wantErrors: ["amount"] },
    { name: "an amount with a thousands separator", values: { ...groceries, amount: "1,200" }, wantErrors: ["amount"] },
    {
      name: "an expense split in two",
      values: { ...groceries, splits: [{ categoryId: "201", amount: "4.50" }] },
      wantErrors: [],
    },
    {
      name: "splits that take the whole amount",
      values: { ...groceries, splits: [{ categoryId: "201", amount: "4.50" }, { categoryId: "202", amount: "8" }] },
      wantErrors: ["amount"],
    },
    {
      name: "a split with no category or amount",
      values: { ...groceries, splits: [{ categoryId: "", amount: "" }] },
      wantErrors: ["splits.0.categoryId", "splits.0.amount"],
    },
    {
      name: "a split of zero",
      values: { ...groceries, splits: [{ categoryId: "201", amount: "0" }] },
      wantErrors: ["splits.0.amount"],
    },
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
      accounts: [account(2, "Personal"), account(1, "Joint")],
      balances: [
        { financialAccountId: 2, balances: [balance(102, "Vault", "EUR"), balance(101, "Personal", "EUR")] },
        { financialAccountId: 1, balances: [balance(100, "Joint", "EUR")] },
      ],
      want: [
        { id: 100, label: "Revolut · Joint", currencyCode: "EUR" },
        { id: 101, label: "Revolut · Personal", currencyCode: "EUR" },
        { id: 102, label: "Revolut · Personal · Vault", currencyCode: "EUR" },
      ],
    },
    {
      name: "accounts sharing a name told apart and sorted by institution",
      accounts: [account(1, "Personal", false, "Revolut"), account(2, "Personal", false, "N26")],
      balances: [
        { financialAccountId: 1, balances: [balance(100, "Cash", "EUR")] },
        { financialAccountId: 2, balances: [balance(101, "Cash", "EUR")] },
      ],
      want: [
        { id: 101, label: "N26 · Personal · Cash", currencyCode: "EUR" },
        { id: 100, label: "Revolut · Personal · Cash", currencyCode: "EUR" },
      ],
    },
    {
      name: "a card's balance that is owed",
      accounts: [account(3, "Amex")],
      balances: [{ financialAccountId: 3, balances: [balance(103, "Amex", "USD", { kind: "liability" })] }],
      want: [{ id: 103, label: "Revolut · Amex", currencyCode: "USD" }],
    },
    {
      name: "closed accounts and balances left out",
      accounts: [account(1, "Cash"), account(2, "Old bank", true)],
      balances: [
        { financialAccountId: 1, balances: [balance(100, "Cash", "EUR"), balance(104, "Old vault", "EUR", { closed: true })] },
        { financialAccountId: 2, balances: [balance(105, "Old bank", "EUR")] },
      ],
      want: [{ id: 100, label: "Revolut · Cash", currencyCode: "EUR" }],
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
    category(201, "Restaurants", "expense", { parentId: 200 }),
    category(200, "Food", "expense"),
    category(202, "Travel", "expense"),
    category(203, "Old hobby", "expense", { closed: true }),
    category(210, "Salary", "income"),
  ]
  const cases: { name: string; kind: Category["kind"]; want: Choice[] }[] = [
    {
      name: "open ones of the kind, in tree order",
      kind: "expense",
      want: [
        { id: 200, label: "Food", depth: 0 },
        { id: 201, label: "Restaurants", depth: 1 },
        { id: 202, label: "Travel", depth: 0 },
      ],
    },
    {
      name: "income ones",
      kind: "income",
      want: [{ id: 210, label: "Salary", depth: 0 }],
    },
  ]

  for (const { name, kind, want } of cases) {
    it(name, () => {
      expect(categoryChoices(categories, kind)).toEqual(want)
    })
  }
})

describe("pickers", () => {
  const cash: Choice = { id: 100, label: "Cash", currencyCode: "EUR" }
  const dollars: Choice = { id: 103, label: "Cash (USD)", currencyCode: "USD" }
  const categories = [
    category(200, "Food", "expense"),
    category(202, "Travel", "expense"),
    category(210, "Salary", "income"),
  ]
  const food: Choice = { id: 200, label: "Food", depth: 0 }
  const travel: Choice = { id: 202, label: "Travel", depth: 0 }
  const salary: Choice = { id: 210, label: "Salary", depth: 0 }

  const cases: {
    name: string
    type: RecordTransactionFormValues["type"]
    fromId: string
    want: ReturnType<typeof pickers>
  }[] = [
    {
      name: "an expense, before a balance is chosen",
      type: "expense",
      fromId: "",
      want: { from: { label: "Paid from", choices: [cash, dollars] }, to: { label: "Category", choices: [food, travel] } },
    },
    {
      name: "an expense from a dollar balance, offered every category",
      type: "expense",
      fromId: "103",
      want: { from: { label: "Paid from", choices: [cash, dollars] }, to: { label: "Category", choices: [food, travel] } },
    },
    {
      name: "an income",
      type: "income",
      fromId: "",
      want: { from: { label: "Category", choices: [salary] }, to: { label: "Received into", choices: [cash, dollars] } },
    },
    {
      name: "a transfer, its source not offered as its destination",
      type: "transfer",
      fromId: "100",
      want: { from: { label: "From", choices: [cash, dollars] }, to: { label: "To", choices: [dollars] } },
    },
  ]

  for (const { name, type, fromId, want } of cases) {
    it(name, () => {
      expect(pickers(type, [cash, dollars], categories, fromId)).toEqual(want)
    })
  }
})

describe("offers", () => {
  const picker = {
    label: "Category",
    choices: [
      { id: 200, label: "Food" },
      { id: 201, label: "Rent" },
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
    { id: 100, label: "Cash", currencyCode: "EUR" },
    { id: 101, label: "Revolut", currencyCode: "EUR" },
    { id: 103, label: "Cash (USD)", currencyCode: "USD" },
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
          categoryId: 200,
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
          categoryId: 210,
          amount: "2500",
          occurredAt: new Date(2026, 8, 30).toISOString(),
          description: "September salary",
        },
      },
    },
    {
      name: "an expense with tags",
      values: { ...groceries, tagIds: [5, 3] },
      crossing: false,
      want: {
        type: "expense",
        body: {
          idempotencyKey: "3f2a",
          balanceLedgerAccountId: 100,
          categoryId: 200,
          amount: "12.50",
          occurredAt: "2026-10-04T08:30:00.000Z",
          tagIds: [5, 3],
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
    {
      name: "an expense split in two, its category taking the rest",
      values: { ...groceries, splits: [{ categoryId: "201", amount: "4.5" }, { categoryId: "202", amount: "1.25" }] },
      crossing: false,
      want: {
        type: "expense",
        body: {
          idempotencyKey: "3f2a",
          balanceLedgerAccountId: 100,
          amount: "12.50",
          occurredAt: "2026-10-04T08:30:00.000Z",
          lines: [
            { categoryId: 200, amount: "6.75" },
            { categoryId: 201, amount: "4.5" },
            { categoryId: 202, amount: "1.25" },
          ],
        },
      },
    },
    {
      name: "a mortgage instalment, its interest a charge",
      values: { ...groceries, type: "transfer", fromId: "100", toId: "104", amount: "1200", splits: [{ categoryId: "220", amount: "800" }] },
      crossing: false,
      want: {
        type: "transfer",
        body: {
          idempotencyKey: "3f2a",
          fromLedgerAccountId: 100,
          toLedgerAccountId: 104,
          amount: "1200",
          occurredAt: "2026-10-04T08:30:00.000Z",
          charges: [{ categoryId: 220, amount: "800" }],
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
