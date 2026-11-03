# ADR 0008: Intent-level transactions

- Status: Implemented
- Date: 2026-10-03
- Depends on: [ADR 0001](0001-double-entry-ledger.md), [ADR 0003](0003-categories-and-tags.md), [ADR 0006](0006-intent-level-account-setup.md), and [ADR 0007](0007-web-ui-and-http-gateway.md)

## Context

The only way to record a movement of money today is `PostJournalEntry`: an
idempotency key, a date, and at least two postings, each a ledger account, a
positive amount, and a side. That is the ledger's language, not a person's.
Someone who paid for groceries by card thinks "I spent 12.50 on groceries with
my card", not "debit Groceries 12.50, credit Card balance 12.50", and has to
know that paying by card credits a liability while paying from a bank account
credits an asset.

ADR 0006 raised account setup to the level a person thinks at. This ADR does
the same for the movements most people record most often, so the web UI's form
(ADR 0007) and, later, MCP's quick capture ask for a transaction, and the sides
are decided once, in code.

## Decision

### Three transactions

A new facade package, `finance/workflows/facade/transactions`, has three
workflows, one per file:

| Workflow | A person says | Credits | Debits |
| --- | --- | --- | --- |
| `RecordExpense` | "I paid 12.50 for groceries from my card." | the balance | the expense category |
| `RecordIncome` | "My salary of 2000 arrived in my bank account." | the income category | the balance |
| `RecordTransfer` | "I moved 500 from my current account to my savings vault." | the balance it leaves | the balance it reaches |

A transfer may also cross currencies: "I moved 300 EUR from my current account
and 324.15 USD arrived in my dollar balance".

A balance is a ledger account of a financial account: "Cash", "Card balance",
a savings vault (ADR 0006). A category is an income or expense ledger account
(ADR 0003).

All three are one rule: the account the money leaves is credited, the account
it reaches is debited, and the amount is always positive. A card needs no
special case. Paying by card credits "Card balance", a liability, so the amount
owed grows. Paying the card off is a transfer from the bank's "Cash" to "Card
balance", which debits the liability, so the amount owed shrinks. A cashback
paid onto the card is income to "Card balance". The workflow tells the three
apart only to check that each account is of the kind its role needs.

### Inputs

| Field | Expense | Income | Transfer |
| --- | --- | --- | --- |
| `idempotencyKey` | yes | yes | yes |
| `balanceLedgerAccountId` | yes, paid from | yes, paid into | |
| `categoryLedgerAccountId` | yes, an expense category | yes, an income category | |
| `fromLedgerAccountId`, `toLedgerAccountId` | | | yes, two different balances |
| `amount` | yes, positive | yes, positive | yes, positive, the amount that leaves |
| `toAmount` | | | the amount that arrives: required between currencies, else omitted or equal to `amount` |
| `occurredAt` | yes | yes | yes |
| `description` | no | no | no |

The input types live in `finance/common/types/transactiontypes` and are
validated with CUE constraints, as the other facades' are.

### Checks

Before posting, each workflow retrieves its two ledger accounts through
`RetrieveLedgerAccount` and fails with `ErrInvalidParameter`, writing nothing,
when:

- a balance is not an asset or liability that belongs to a financial account.
  This keeps the shared "Opening balances" and "FX clearing" accounts out,
  since they belong to none;
- a category is not of the kind the workflow needs: an expense for
  `RecordExpense`, an income for `RecordIncome`;
- an expense's or an income's two accounts are in different currencies. A
  movement between currencies is an exchange, which `RegisterFXConversion`
  records, and the message says so. The ledger would also refuse it, as an
  entry that does not balance in either currency, but its message names
  currency ids;
- a transfer's two balances are the same account;
- a transfer between currencies has no `toAmount`, or one within a currency
  has a `toAmount` different from `amount`.

A closed or unknown account is left to `PostJournalEntry`, which already fails
with `ErrResourceClosed` and `ErrNotFound` inside the transaction that posts,
where an account cannot be closed concurrently.

### Posting and idempotency

Each workflow posts one journal entry with two postings, through
`PostJournalEntry` as a child workflow, so it is one financial write in one
activity, as CLAUDE.md requires. The facade adds no persistence of its own.

A transfer between currencies is posted instead through
`RegisterFXConversion`, again one activity, with no fee: the balance it leaves
is credited `amount` and the one it reaches debited `toAmount`, each currency
balancing through its own "FX clearing" account (ADR 0006). The transfer finds
those accounts by their key through `CreateLedgerAccount`, which creates one
only if a balance was opened without it. The rate is derived from the two
amounts and stored on the entry, as for any conversion.

The idempotency key is required, not defaulted. The caller makes one per
transaction the person means to record, such as one per opening of the UI's
form, so a request retried after a timeout or submitted twice records the
transaction once. A default key made inside the workflow would protect only
against a retry of the workflow, not of the request that started it.

The key is posted with the transaction's type in front of it,
`expense:<key>`, `income:<key>`, or `transfer:<key>`, so the same key sent for
two different types cannot return the other's entry. As with any
retrieve-or-insert (ADR 0005), a retry with a known key returns the entry
first posted with it, unchanged, even if its other fields differ.

### Results

Each workflow returns the id of the journal entry it posted, or found by its
key. Correcting a transaction is reversing its entry, as for any entry
(ADR 0001), and recording it again.

### The type is not stored

The entry records no transaction type. A screen that shows entries as
transactions derives the type from the kinds of their postings' accounts:
one on an expense category is an expense, one on an income category is an
income, and one between two balances is a transfer, through the clearing
accounts if it crosses currencies. A stored type could
contradict the postings, and entries posted by other workflows, such as
opening balances and trades, would have none.

### Exposure

The web gateway exposes the three workflows as `POST /api/transactions/expenses`,
`/api/transactions/incomes`, and `/api/transactions/transfers`, and the UI has
one form whose type chooses which accounts each picker offers. The UI builds
its pickers from the endpoints it already has: the balances of each open
financial account, and the open categories. MCP's quick capture of a single
movement (ADR 0007) uses the same workflows when it is built.

There is no endpoint that posts arbitrary postings. One is added when a screen
needs it, such as manual entries or split transactions.

### Not covered

These are left for later, each as a transaction of its own or an extension of
these three:

- **Exchange fees**: a transfer between currencies carries none; an exchange
  with a fee is recorded with `RegisterFXConversion`;
- **Splits**, one payment over several categories, which need more than two
  postings;
- **Refunds**, money back on an expense category, which swap an expense's
  sides;
- **The payee**, which `Counterparty` could hold once there is a way to choose
  or create one;
- **Tags** (ADR 0003);
- **Trades and vests**, which have their own workflows (ADR 0004).

## Examples

Groceries paid by card, with a key the UI made when the form opened:

```text
RecordExpense{idempotencyKey: "3f2a…", balanceLedgerAccountId: <Card balance (EUR)>,
              categoryLedgerAccountId: <Groceries (EUR)>, amount: 12.50,
              occurredAt: 2026-10-03T10:00, description: "Esselunga"}
  -> entry 41: debit Groceries 12.50, credit Card balance 12.50
```

Paying the card off from the bank, and the salary arriving:

```text
RecordTransfer{idempotencyKey: "9c1e…", fromLedgerAccountId: <Revolut Personal / Cash (EUR)>,
               toLedgerAccountId: <Card balance (EUR)>, amount: 300, occurredAt: …}
  -> entry 42: debit Card balance 300, credit Cash 300

RecordIncome{idempotencyKey: "77ab…", balanceLedgerAccountId: <Revolut Personal / Cash (EUR)>,
             categoryLedgerAccountId: <Salary (EUR)>, amount: 2000, occurredAt: …}
  -> entry 43: debit Cash 2000, credit Salary 2000
```

Requests that fail before posting:

```text
RecordExpense{... categoryLedgerAccountId: <Salary (EUR)>}
  -> ErrInvalidParameter: the category is an income category, not an expense one

RecordIncome{... balanceLedgerAccountId: <Cash (EUR)>, categoryLedgerAccountId: <Salary (USD)>}
  -> ErrInvalidParameter: the accounts are in different currencies; record an exchange

RecordTransfer{... from <Cash (EUR)>, to <Cash (USD)>, amount: 300}
  -> ErrInvalidParameter: the balances are in different currencies; give the amount that arrives as toAmount
```

Dollars for a trip, from the euro balance to the dollar one:

```text
RecordTransfer{idempotencyKey: "5d0b…", fromLedgerAccountId: <Cash (EUR)>,
               toLedgerAccountId: <Cash (USD)>, amount: 300, toAmount: 324.15, occurredAt: …}
  -> entry 44: debit FX clearing (EUR) 300, credit Cash (EUR) 300,
               debit Cash (USD) 324.15, credit FX clearing (USD) 324.15
```

Sending the first request again, with the same key, returns entry 41 and posts
nothing.

## Alternatives considered

- **Mapping transactions to postings in the UI**, over an endpoint that posts
  arbitrary postings. Less code now, but the rules about kinds and currencies
  would exist only in TypeScript, and MCP's quick capture would repeat them.
- **One `RecordTransaction` workflow with a type field.** One endpoint instead
  of three, but its input would be the union of the three, with fields that
  apply to some types only, and CUE constraints that branch on the type.
  Three small workflows each say exactly what they need.
- **A single from/to shape for every type**, with the type only labelling it.
  It is what the three share underneath, but it would accept an expense whose
  "to" is a balance, and could not check that each account plays its role.
- **Storing the type in the entry's metadata.** See above: derived, not stored.
- **Defaulting the idempotency key in the workflow.** It would protect only
  against a retry of the workflow, not against a request sent twice.

## Consequences

- The UI's form, and later MCP's quick capture, record the three most common
  movements without the person or the model choosing sides.
- The rules about which accounts a transaction may use live in one place and
  are tested as workflow tests with mocked child workflows.
- A screen that lists transactions derives their type from their postings.
- Exchanges, splits, refunds, and payees each need a later decision, and may
  extend these workflows' inputs.
