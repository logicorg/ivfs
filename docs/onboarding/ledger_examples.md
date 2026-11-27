# The ledger model, in practice

This page is a reading path through `Institution`, `FinancialAccount`,
`LedgerAccount`, `Counterparty`, `JournalEntry`/`Posting`, and
`Category`, and `PositionMovement`, told through six worked examples. It is meant to be
useful to a human onboarding onto the codebase and to an agent implementing a
new activity or workflow, without having to first read all the ADRs.

For the design rationale, see [ADR 0001](../adrs/0001-double-entry-ledger.md)
(the ledger), [ADR 0002](../adrs/0002-ledger-domain-and-application-api.md)
(the domain model and API),
[ADR 0003](../adrs/0003-categories-and-tags.md) (tags), and
[ADR 0019](../adrs/0019-currency-neutral-categories.md) (categories). This page
only restates what's needed to read the examples; it is not a replacement for
the ADRs' invariants.

## The entities, in one paragraph each

**`Institution`** (`pkg/model/institution.go`) is just identity: a bank, a
broker, an employer's payroll system if you want to track it. It has a
`Name`, a `Kind` (free-form string, e.g. `"bank"`), and an `OpenedAt`/
`ClosedAt`. It holds no money and has no currency.

**`FinancialAccount`** (`pkg/model/ledger.go`) is the provider-side
relationship: "my Revolut account", "my IBKR account". It belongs to exactly
one `Institution` (`InstitutionID`) but is **not itself a balance and has no
currency of its own** — a single Revolut `FinancialAccount` can back both a
EUR balance and a USD balance. It exists so a client can group "everything
at Revolut" without forcing every currency into one bucket.

**`LedgerAccount`** (`pkg/model/ledger.go`) is the actual balance-bearing
bucket — the thing debits and credits move in and out of. Every
`LedgerAccount` has exactly one `CurrencyID` and one `Kind` (`asset`,
`liability`, `equity`, `income`, or `expense`). It *optionally* points at a
`FinancialAccountID` when it represents a real institution-held balance (e.g.
"Revolut EUR cash"); when it doesn't, it's a shared chart-of-accounts entry
such as "Opening balances", "FX clearing", "Expenses", or "Income", one per
currency. "Expenses" and "Income" are the only expense and income accounts,
and ledger accounts have no hierarchy.

**`Category`** (`pkg/model/category.go`) says what an income or an expense
was for: "Groceries", "Salary". It has a `Kind` (`income` or `expense`), an
optional `ParentID` for a hierarchy at most two parent links deep, and **no
currency and no balance**. It is carried by a posting (`Posting.CategoryID`)
to the income or expense account of the same kind, and its total is the sum
of those postings (see ADR 0019).

**`Counterparty`** (`pkg/model/ledger.go`) is optional descriptive metadata
on a `JournalEntry` — a merchant, an employer, a person. It is attached via
`JournalEntry.CounterpartyID` and is **never a posting target**; postings
only ever point at a `LedgerAccountID`. It can be deleted: the entries that
name it keep their postings and lose the counterparty, so no balance moves.

**`JournalEntry` + `Posting`** (`pkg/model/ledger.go`) are the actual
double-entry record and the one thing every example below produces.
A `JournalEntry` is the header (when it happened, its description, its
counterparty, its status). Each `Posting` is one accounting line: a
`LedgerAccountID`, a positive `Amount` (in that account's currency), a
`Side` (`debit` or `credit`), and an optional `CategoryID`. A *posted* entry must have at least two
postings, and for every currency present, the debits must sum to the
credits. The database checks this on every write, so a posted entry can be
edited or deleted, but never left unbalanced.

**`PositionMovement`** (`pkg/model/ledger.go`) is the separate subledger for
investment units (shares, ETF units, ...). It's the source of truth for "how
many units of AAPL do I own", tracked independently of money. A movement
that has a monetary effect (an ordinary buy/sell) references the
`JournalEntry` that moved the cash, and both rows are written in the same
SQL transaction. A movement with no monetary effect (a split, units moved
between two of your own accounts, a correction) has no entry and carries an
idempotency key of its own.

## The one primitive underneath almost everything

There is a temptation to think "expense", "income", "transfer", and "FX
conversion" are different kinds of operations. Mechanically, **all four are
the same command**: `PostJournalEntryInput` (`finance/common/types/
journaltypes/types.go`), executed by the `PostJournalEntry` activity/workflow
(`finance/activities/ledger/journal`, `finance/workflows/ledger/journal`).
It takes an `OccurredAt`, an optional `Description`/`CounterpartyID`, and a
list of `PostingInput{LedgerAccountID, Amount, Side, CategoryID}` — at least
two, balanced per currency. What makes an entry "an expense" versus "a transfer"
is purely which `LedgerAccount`s and how many postings appear in that list;
there's no separate expense/transfer/income table or command.

The one genuine exception is buying or selling an asset: that needs a
`PositionMovement` row written atomically alongside the postings, so it gets
its own command, `RegisterInvestmentTrade` (`finance/activities/ledger/
portfolio`, `finance/workflows/ledger/portfolio`) — see example 5. Units
received from a vest have an entry and a movement too, and use `RegisterVest`,
see example 6. A change of units with no cash effect has no entry at all and
uses `RegisterPositionMovement` instead, described at the end of example 5.

Every posting amount is positive; the `Side` field, not the sign, says
whether it increases or decreases the account. For an asset or expense
account, a debit increases the balance; for a liability, equity, or income
account, a credit increases it.

## Example 1 — grocery expense

Buying groceries at Tesco with a Revolut EUR card.

```text
JournalEntry: counterparty = Tesco (optional)
  Debit  Expenses:EUR             45.00   (LedgerAccount, kind=expense, no FinancialAccountID; category = Groceries)
  Credit Assets:Revolut:EUR       45.00   (LedgerAccount, kind=asset, FinancialAccountID = Revolut)
```

- Two postings, one currency (EUR), debits = credits. Done with one
  `PostJournalEntry` call.
- `Expenses:EUR` is the shared "Expenses" `LedgerAccount` of the currency,
  which the `RecordExpense` workflow finds or creates. "Groceries" is a
  `Category`, created once with `CreateCategory` and reused in every
  currency; the debit carries its id.
- `Assets:Revolut:EUR` is the cash `LedgerAccount` linked to the Revolut
  `FinancialAccount`. It's typically created alongside the `FinancialAccount`
  itself (`CreateFinancialAccount` can create a first cash ledger account in
  the same transaction).
- `Tesco` is an optional `Counterparty`, attached via `CounterpartyID` — it
  never appears as a posting.

## Example 2 — income from an employer

Salary paid into the same Revolut EUR account.

```text
JournalEntry: counterparty = Employer (optional)
  Debit  Assets:Revolut:EUR   3000.00
  Credit Income:EUR           3000.00   (category = Salary)
```

- Same primitive, same shape as example 1, just with the asset account on
  the debit side and the income account on the credit side. There is no
  separate "income" command — this is still one `PostJournalEntry` call.
- `Income:EUR` is the shared "Income" account of the currency, and "Salary"
  an income `Category` on its posting.

## Example 3 — transfer between two owned accounts, same currency

Moving EUR100 from Revolut EUR to an IBKR cash EUR balance.

```text
JournalEntry
  Debit  Assets:IBKR:Cash:EUR    100.00
  Credit Assets:Revolut:EUR      100.00
```

- Both legs are asset accounts in the *same* currency, so this is still a
  plain two-posting, single-currency entry — no FX machinery involved.
  `Assets:IBKR:Cash:EUR` and `Assets:Revolut:EUR` are two different
  `LedgerAccount`s, each linked to its own `FinancialAccount` (IBKR, Revolut)
  but sharing the currency.
- This is exactly why `FinancialAccount` and `LedgerAccount` are separate:
  the transfer moves currency between institutions, not between
  currency-typed containers that happen to also be provider-specific.

## Example 4 — transfer with currency exchange

Exchanging USD100 held in Revolut USD for EUR90 in Revolut EUR, with a USD1
fee — the worked example from ADR 0001.

```text
JournalEntry metadata: { type: "fx_conversion", effective_rate: "90/99" }

  USD postings (balance among themselves)
    Debit  Assets:FX-Clearing:USD   99.00
    Debit  Expenses:USD              1.00   (category = FX fees)
    Credit Assets:Revolut:USD      100.00

  EUR postings (balance among themselves)
    Debit  Assets:Revolut:EUR       90.00
    Credit Assets:FX-Clearing:EUR   90.00
```

- Still one `JournalEntry`, still one `PostJournalEntry` call — but now four
  postings across two currencies. The balance rule (debits = credits) is
  enforced **independently per currency**, not across the whole entry: USD
  postings (99 + 1 debit = 100 credit) balance on their own, and EUR postings
  (90 debit = 90 credit) balance on their own.
- A single currency cannot post directly into another, so a pair of
  single-currency `Assets:FX-Clearing:*` ledger accounts bridges the two
  legs. They're ordinary asset `LedgerAccount`s, just conventionally named
  `FX-Clearing`.
- The fee is an explicit posting to "Expenses" in the currency it was
  actually charged in, under an expense category — never netted into the
  exchange rate. Any rounding residual from currency-precision quantization
  would similarly be an explicit posting, never silently dropped.
- The quoted/effective rate and provider reference live in the journal
  entry's free-form `Metadata` map, for audit purposes — they don't affect
  posting validation.
- `RegisterFXConversion` builds exactly this entry from the two settled
  amounts, the source, destination, and clearing ledger accounts, the fee's
  category, and an idempotency key. The workflow finds "Expenses" in the
  source currency for the fee. It records the rate as the fraction `90/99` in the entry
  metadata, since it has no exact decimal form. Building the same entry by
  hand with `PostJournalEntry` remains possible.

## Example 5 — buying stock

Buying one AAPL share for EUR100 out of an IBKR EUR cash balance.

```text
JournalEntry
  Debit  Assets:IBKR:Investments:EUR   100.00
  Credit Assets:IBKR:Cash:EUR          100.00

PositionMovement (same DB transaction, references the JournalEntry above)
  Kind                = trade
  FinancialAccountID  = IBKR
  AssetID             = AAPL
  UnitsDelta          = +1
  CostBasisAmount     = 100.00
  CostBasisCurrencyID = EUR
```

- This is the one case that is **not** a plain `PostJournalEntry` call. Use
  `RegisterInvestmentTrade` (`RegisterInvestmentTradeInput` in
  `finance/common/types/portfoliotypes/trade.go`), which takes the same
  `PostJournalEntryInput` for the cash side plus `FinancialAccountID`,
  `AssetID`, `UnitsDelta`, and an optional cost basis, an amount together with
  its currency, and writes the posted entry, its postings, and the
  `PositionMovement` row atomically in one SQL transaction. The cost basis has
  a currency of its own because one asset can be held and bought in several
  currencies.
- `Assets:IBKR:Investments:EUR` and `Assets:IBKR:Cash:EUR` are both ordinary
  asset `LedgerAccount`s linked to the IBKR `FinancialAccount` — the cash
  side of the trade is still a normal balanced posting pair.
- Current share ownership is derived by summing `UnitsDelta` for a given
  `FinancialAccountID`+`AssetID`, not stored as a running total anywhere —
  see `PositionAt`.
- A sale is the same shape with the debit/credit reversed and a negative
  `UnitsDelta`.
- A change of units with no cash effect, such as a split, moving shares
  between two of your own financial accounts, or a correction, is not a trade.
  It is a `PositionMovement` with no `JournalEntryID`, written by
  `RegisterPositionMovement` with one leg per financial account, under one
  idempotency key. `DeletePositionMovement` undoes the whole command, and
  `ListPositionMovements` returns the history. An opening balance is not an
  adjustment: record it with `RegisterInvestmentTrade` against an equity account
  so the ledger stays balanced. See
  [ADR 0004](../adrs/0004-position-subledger.md).

## Example 6 — an RSU vest with sell-to-cover

100 ACME units vest on 1 March at a fair market value of EUR10 each, and the
broker sells 50 of them at once, at the same price, to pay the tax. These are
two facts, so they are two commands.

```text
RegisterVest
  JournalEntry
    Credit Income:EUR                    1000.00   (category = RSU compensation)
    Debit  Assets:IBKR:Investments:EUR   1000.00
  PositionMovement (same DB transaction)
    Kind = vest, UnitsDelta = +100, CostBasisAmount = 1000.00, CostBasisCurrencyID = EUR

RegisterInvestmentTrade (an hour later)
  JournalEntry
    Debit  Expenses:EUR                  400.00    (category = Tax withheld)
    Debit  Assets:IBKR:Cash:EUR          100.00
    Credit Assets:IBKR:Investments:EUR   500.00
  PositionMovement (same DB transaction)
    Kind = trade, UnitsDelta = -50, CostBasisAmount = -500.00, CostBasisCurrencyID = EUR
```

- `RegisterVest` (`RegisterVestInput` in `finance/common/types/portfoliotypes/
  vest.go`) takes the same `PostJournalEntryInput` as a trade, so you supply the
  postings, plus the units, and the cost basis with its currency. The command
  does not know how the compensation is taxed: the income and any tax withheld
  are just postings. The units are the ones delivered, net of any the employer
  withheld, and the cost basis is their fair market value on the vesting date.
  If the employer had kept 40 of the 100 units for tax instead, the vest would
  be 60 units with a cost basis of 600, and its entry would credit the income
  1000, debit the investments 600, and debit the tax 400.
- Sell-to-cover is an ordinary `RegisterInvestmentTrade` whose entry carries the
  withholding posting. The vest has to be recorded first, since holdings never
  go negative and movements of the same instant are ordered by when they were
  written.
- Everything can be read back: the units from `PositionAt` (50), the income
  and tax from the totals of their categories (1000 and 400), the cash
  proceeds from the balance of its ledger account (100), and the cost basis by summing the basis of the history from
  `ListPositionMovements` (500).
- A vest is corrected by deleting its entry with `DeleteJournalEntry` and
  recording it again. A holding left negative by it is reported as a warning.
- An ESPP purchase is not a vest, since it has cash on the other side. It is a
  `RegisterInvestmentTrade` whose entry credits the payroll cash account and
  books the discount as income, with the fair market value as the cost basis:
  10 units worth 20 each bought for 17 are debit investments 200, credit bank
  cash 170, and credit ESPP discount income 30.
- These stories run end to end in
  `finance/scenarios/equity_compensation_test.go`.

## Reading entries back

`ListJournalEntries` returns a page of entries, each with its postings. Filter
it by ledger account (the entries that have a posting on it), category (a
posting carrying it or one of its sub-categories), counterparty,
exact external ID, status, and a date range where `From` is included and `To`
is not. It lists oldest first, or newest first with `Descending`. When more
entries follow the page it comes with a `Next` cursor: pass it as `After` in the
next call to get the following page, until `Next` is empty. An empty status list
means every status, so ask for `posted` to leave out drafts.

To read one entry, use `RetrieveJournalEntry`. To read the lines of one account
use `ListPostings`, and for what an account holds, `LedgerAccountBalance` or
`TrialBalanceAt`. The units movements have their own paged history,
`ListPositionMovements`.

## Quick reference

| I want to... | Entity/ies involved | Command |
| --- | --- | --- |
| Record any plain expense/income/transfer movement | `LedgerAccount`(s), optional `Counterparty` | `PostJournalEntry` |
| Convert money between two currencies | source/destination/clearing `LedgerAccount`s, a fee `Category` | `RegisterFXConversion` |
| Buy or sell an asset | `LedgerAccount`(s) + `PositionMovement` | `RegisterInvestmentTrade` |
| Receive units from an RSU vest | `LedgerAccount`(s) + `PositionMovement` of kind `vest` | `RegisterVest` |
| Split, move units between own accounts, or correct units | `PositionMovement` only, no `JournalEntry` | `RegisterPositionMovement` (undo with `DeletePositionMovement`, read with `ListPositionMovements`) |
| Set up a new provider relationship | `Institution`, `FinancialAccount` | `CreateFinancialAccount` (can create a first cash `LedgerAccount` in the same call) |
| Add a spending/income category | `Category` (income/expense kind, no currency) | `CreateCategory` |
| Correct a posted entry | the `JournalEntry` is deleted and recorded again | `DeleteJournalEntry` |
| Page through entries by account, counterparty, external ID, status or date | `JournalEntry` + `Posting` | `ListJournalEntries` |
| Delete a counterparty, clearing it from its entries | `Counterparty`, the `JournalEntry` rows naming it | `DeleteCounterparty` |

## Common mistake to avoid

Don't reach for a new table or a new "transaction type" when adding a
feature. If it's a movement of money, it's a `PostJournalEntryInput` with the
right set of postings; the only structural extension in the whole model is
the units subledger (`PositionMovement`) for investments. `Counterparty`
never goes in a posting, `FinancialAccount` never holds a balance directly,
and a `Category` never holds money: it labels a posting to "Expenses" or
"Income".
