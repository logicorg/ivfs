# ADR 0022: Debts and real assets

- Status: Implemented
- Date: 2026-11-27
- Depends on: [ADR 0006](0006-intent-level-account-setup.md) (account types),
  [ADR 0008](0008-intent-level-transactions.md) (transactions),
  [ADR 0009](0009-intent-level-investments.md) (trades and the "Investments"
  balance), [ADR 0015](0015-net-worth-over-time.md) (what net worth counts), and
  [ADR 0017](0017-editing-journal-entries.md) (the update workflows)
- Amends: ADR 0006 (two account types; loans are no longer "not covered") and
  ADR 0008 (split expenses and transfers with charges)

## Context

Three debts have to be supported: credit card debt, a mortgage (including
switching it to another lender), and a margin loan. A mortgage also brings the
house it pays for.

A credit card already works: a `card` account holds "Card balance", a
liability, and purchases, interest, payments, and cashback are expenses,
transfers, and incomes on it (ADR 0008). A margin loan already works too: it is
the brokerage's "Cash" gone below zero, which is how brokers report it, and an
asset balance may be negative, as an overdraft is (ADR 0006).

A mortgage does not fit as well:

- No account type holds a loan. A `card` gives a liability, but named "Card
  balance" and under an institution of kind "card issuer".
- A house has nowhere to live. A `brokerage` holds units of assets, but a house
  is not an investment held at a broker.
- An instalment is one payment that both reduces the debt and pays interest.
  `RecordExpense` and `RecordTransfer` post two lines each, so one instalment
  is two entries: a statement line no longer matches one entry, and deleting or
  editing one half leaves the balances wrong.

The constraint is to change the database as little as possible, and not at
all if the model allows it.

## Decision

Nothing in this ADR changes the schema. Account types are recipes, not stored
(ADR 0006), `asset.kind` has no CHECK constraint, and a split is more postings
on one entry.

### Two account types

| Type | Holds | Examples | Balances per currency, under the financial account |
| --- | --- | --- | --- |
| `loan` | a debt with a contract | a mortgage, a car or personal loan, a securities-backed credit line | "Principal" (liability) |
| `real_asset` | tangible things, and money | a house, a share of one, a car | "Cash" (asset) and "Investments" (asset) |

`loan` is the shape of `card`: one liability per currency. Its institution is
the lender, created as kind "lender". A loan already running opens with what is
still owed as its opening balance, posted against "Opening balances" with the
sides a liability takes, as a card's is.

`real_asset` has the balances of a `brokerage` and behaves like one: the
units are position movements on the financial account (ADR 0004), "Investments"
holds their cost and is left out of net worth (ADR 0015), and the value is
units × an `AssetValue`, entered by hand or by a provider (ADR 0014). It is a
type of its own so that a house is not listed as a brokerage. Keeping the
balance names means `IsBrokerageInvestments`, `WhereNotBrokerageInvestments`,
and `retrieveBrokerage` are unchanged, and `RecordInvestment` buys and sells on
it as on any brokerage. Its institution is optional: without one, the account
belongs to the built-in institution "Real assets", created on first use, as a
wallet belongs to "Physical cash".

A house is an `Asset` of a new kind, `real_estate`, and a car one of kind
`vehicle`. A share of either is a fraction of a unit: a home owned with a
partner is 0.5. Depreciation is a lower value entered over time, as a
revaluation is, with no schedule and no entries.

A loan is its own financial account rather than a balance under the current
account, because it is its own contract: switching it closes one financial
account and opens another, and closing the current account does not close the
mortgage. Adding a kind to `AddBalance` was considered and rejected: it would
show the person ledger kinds, which ADR 0006 and ADR 0008 decide in code; the
balance's name would be chosen by the caller, so nothing could find a loan by
name; and a lender with no current account would need `OpenAccount` to accept
no currencies.

### Splits on the existing transactions

An expense or a transfer may be detailed in one entry. The input keeps its
shape, and `amount` keeps its meaning: what leaves the balance.

- `RecordExpense` and `UpdateExpense` take optional `lines: [{categoryId,
  amount}]`, for a payment spent on several things. With `lines`, `categoryId`
  is omitted, every line is an expense category, and the lines add up to
  `amount`. Without them, nothing changes.
- `RecordTransfer` and `UpdateTransfer` take optional `charges: [{categoryId,
  amount}]`, for what moving the money cost: the interest of an instalment, a
  fee, an early-repayment penalty. Each charge debits "Expenses" in the source
  currency under an expense category. Within one currency the balance it
  reaches is debited `amount` less the charges, and `toAmount`, if given, must
  equal that. Between currencies at most one charge is allowed, posted as the
  fee of `RegisterFXConversion`, which already takes a source-currency fee
  included in the source amount.

Both remain one entry, one `PostJournalEntry` (or `RegisterFXConversion`), one
idempotency key with its type in front, and one row on the transactions page.
A new split-payment facade was considered and rejected: it would duplicate the
checks of the expense and transfer workflows and need an update workflow of its
own, and expenses and transfers would each have a second way to be recorded.

## Examples

Amounts in EUR.

### A credit card

```text
Purchase            RecordExpense    from Amex:Card balance      45.00, Groceries
Interest, fees      RecordExpense    from Amex:Card balance      12.00, Card interest
Cashback            RecordIncome     to   Amex:Card balance       5.00, Cashback
Payment             RecordTransfer   Bank:Cash → Amex:Card balance  1,000.00
Balance transfer    RecordTransfer   Amex:Card balance → Visa:Card balance  2,000.00,
                    charges [Card fees 40.00]
```

### A margin loan

```text
Buy 10,000 with 6,000 of cash  RecordInvestment buy on IBKR   IBKR:Cash is −4,000
Margin interest                RecordExpense from IBKR:Cash    18.00, Margin interest
Paying it down                 RecordTransfer Bank:Cash → IBKR:Cash  4,000.00
```

The borrowing is the negative "Cash". A credit line that is a contract of its
own, such as a securities-backed loan, is a `loan`, and each drawdown is a
transfer from its "Principal" to the brokerage's "Cash".

### Buying a house with a mortgage

A 300,000 house, with 10,000 of notary fees, agency fees, and taxes, paid by a
240,000 mortgage and 70,000 of savings:

```text
OpenAccount{type: loan, institutionName: "Intesa", accountName: "Mortgage",
            currencies: [{code: EUR}]}
OpenAccount{type: real_asset, accountName: "Home", currencies: [{code: EUR}]}
InsertAsset{name: "Via Roma 1", kind: real_estate}

RecordTransfer   Mortgage:Principal → Home:Cash    240,000.00
RecordTransfer   Bank:Cash → Home:Cash              70,000.00
RecordInvestment buy on Home, 1 unit of Via Roma 1, amount 300,000.00, fee 10,000.00
                                                   Home:Cash back to 0, basis 310,000
AssetValue       Via Roma 1, 300,000 today, and again whenever it is revalued
```

Home's "Cash" stands for the notary, who receives the money and pays the
seller. A sale runs the other way: a `sell` into Home's "Cash", then transfers
from it that pay off the mortgage and bring the rest to the bank.

### A mortgage instalment

1,200 from the bank, of which 400 reduces the debt and 800 is interest:

```text
RecordTransfer Bank:Cash → Mortgage:Principal  amount 1,200.00,
               charges [Mortgage interest 800.00]

  Credit Bank:Cash             1,200.00
  Debit  Mortgage:Principal      400.00
  Debit  Expenses                800.00   (category = Mortgage interest)
```

A prepayment is the same transfer without charges. Interest added to the debt
is an expense paid from "Principal".

### Switching a mortgage

The new lender pays off the old one. The house does not move: only the debt
does.

1. Open the new loan: `OpenAccount{type: loan, institutionName: <new lender>}`,
   with no opening balance.
2. Take from the old lender's payoff statement the principal outstanding, the
   interest accrued to the payoff date, and any early-repayment penalty.
3. Record the payoff as one transfer from the new "Principal" to the old one.
   Crediting the new liability makes the new debt grow by what was paid out:

   ```text
   RecordTransfer NewLoan:Principal → OldLoan:Principal  amount 182,640.00,
                  charges [Mortgage interest 410.00, Loan fees 230.00]
   ```

   Fees paid from savings instead are a `RecordExpense` from the bank's
   "Cash". An Italian *surroga* has none, so it is a transfer without charges.
   A cash-out refinance, which also pays money into the bank, adds a second
   transfer from the new "Principal" to the bank's "Cash".
4. Check that the old loan's balance is zero
   (`GET /api/ledger-accounts/{id}/balance`). A few cents left are the lender's
   rounding, recorded as interest with a transfer and a charge.
5. Close the old loan (`POST /api/financial-accounts/{id}/close`). Its history
   stays, and net worth still counts it on the dates it was open.
6. Optionally, tag the payoff entry, such as `remortgage-2027`.

A switch to a loan in another currency makes step 3 an exchange between the
two "Principal" balances, with at most one charge.

## Consequences

- The account setup form offers `loan` and `real_asset`, and the asset form
  offers `real_estate` and `vehicle`.
- The expense and transfer forms gain a way to add lines and charges. The
  transactions page and the journal entry sheet show a split entry as one row
  with its postings.
- The MCP tools for expenses and transfers take `lines` and `charges` once
  they are exposed.
- `docs/onboarding/ledger_examples.md` gains these examples, and its quick
  reference gains rows for loans, real assets, and splits.
- A negative brokerage "Cash" is net worth's ordinary arithmetic. Showing it as
  "borrowed" in the UI is a presentation choice left to the UI.

## Not covered

- Loan terms: rate, term, credit limit, and amortisation schedule, and so any
  projection or suggested split of an instalment. They would be the first
  change to the schema, and none of the three debts needs them to record what
  happened.
- A buy funded from other balances, or a sale paid into them, in one entry
  (`fundedBy`, `proceedsTo`). The money passes through the account's "Cash"
  instead, in separate entries.
- One payment into several balances in one entry, such as a cash-out
  refinance.
- More than one charge on a transfer between currencies.
- Splitting an income, such as a payslip with tax and pension deductions.
