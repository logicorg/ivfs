# ADR 0001: Adopt a double-entry ledger

- Status: Implemented
- Date: 2026-08-26
- Amended by: [ADR 0004](0004-position-subledger.md),
  [ADR 0011](0011-deleting-journal-entries.md),
  [ADR 0016](0016-deleting-counterparties.md), and
  [ADR 0017](0017-editing-journal-entries.md)

## Context

The existing `Transaction` model describes a movement with one source and one
destination.  It combines financial accounts and counterparties in alternative
columns and cannot represent a split transaction naturally.  It also makes an
account balance an interpretation of transaction direction rather than a
projection of immutable accounting lines.

The application has no production data.  We can replace this model directly;
backwards compatibility and data migration are out of scope.

## Decision

### Core ledger

Replace the current two-sided `transaction.transaction` model with a ledger
made of `JournalEntry`, `Posting`, and `LedgerAccount`.

`JournalEntry` represents one real-world financial event.  It is the header
that groups the related accounting lines and records shared attributes:

- `id`
- `occurred_at`
- `description`
- `external_id` for imported records
- optional `idempotency_key` for retried commands
- optional `counterparty_id`
- optional metadata, including provider references and precise FX-rate details
- `status`: `draft` or `posted` (ADR 0011 removes `voided`)

`Posting` represents one accounting line in that event.  Every posting has:

- `journal_entry_id`
- `line_number`
- `ledger_account_id`
- positive `amount`, in the currency of its ledger account
- `side`: `debit` or `credit`

`LedgerAccount` is a chart-of-accounts bucket.  It is the direct link between
a posting and the balance it changes.  It has a name, optional parent, and one
of these kinds: `asset`, `liability`, `equity`, `income`, or `expense`.  Every
monetary ledger account has one currency.

Examples include:

```text
Assets:Revolut:EUR
Assets:IBKR:Cash:EUR
Assets:IBKR:Investments:EUR
Expenses:Groceries:EUR
Income:Salary:EUR
Equity:Opening balances:EUR
```

For asset accounts, debits increase the balance and credits decrease it.  For
liability, equity, and income accounts, the natural direction is reversed.
Expense accounts increase with debits.

A posted journal entry must have at least two postings.  The database must
enforce the following rules as part of the transaction that posts the entry:

1. posting amounts are greater than zero;
2. for each currency, the total debit amount equals the total credit amount;
3. line numbers are unique within an entry; and
4. a posting cannot be changed or removed after its entry is posted.

Foreign exchange is recorded with a pair of FX-clearing ledger accounts, one
for each currency.  This preserves the balancing rule in every currency. The
journal entry's optional metadata may preserve a provider reference and the
effective or quoted rate. Fees are separate expense postings in the currency
in which they were charged.

Rates may be calculated at high precision, but every posting amount must obey
its currency's configured minor-unit precision. Any settlement difference
introduced by quantization is recorded explicitly in the affected currency,
for example in `Expenses:FX-Rounding:EUR`; it is never silently discarded.

### Financial accounts and counterparties

The existing institution-held `Account` concept is a financial account, not a
general ledger account.  It should be named `FinancialAccount` when the model
is refactored.  It identifies the real-world relationship with a provider and
has no accounting currency of its own.  It groups one or more single-currency
ledger accounts.  This permits Revolut to have EUR and USD balances under one
financial account and a broker to have separate cash and investment ledger
accounts.

Create a `Counterparty` entity for merchants, employers, and people.  A
counterparty is optional descriptive metadata on a journal entry; it is not an
alternative source or destination field and is normally not a posting account.
Use a counterparty-specific receivable or payable ledger account only when the
application needs to track an amount owed to or by that party.

There will be no polymorphic `account-or-counterparty` column.  A posting has
exactly one `ledger_account_id`; source and destination are inferred from the
debit and credit postings.

### Investments and positions

Keep a dedicated units subledger, renamed from `AssetTransaction` to
`PositionMovement`.  It is the source of truth for shares owned, rather than a
separate `financial_transaction` table.

A position movement records at least:

- `financial_account_id`
- `asset_id`
- signed `units_delta`
- `traded_at`
- cost basis in the configured reporting currency
- the related `journal_entry_id` when the movement has a monetary effect

The purchase or sale of an asset creates its monetary postings and its
position movement atomically.  Current ownership is the sum of
`units_delta` per financial account and asset.  Provider-import records, if
needed later for reconciliation, are raw `BankTransaction` or
`BrokerTransaction` records linked to a journal entry or position movement;
they are not the source of truth for balances or positions.

### Entry lifecycle

Draft entries may be edited.  Once posted, entries and their postings are
immutable.  A correction is an edit or a deletion of the entry, not a reversing
entry (ADR 0017).  Posted entries are never updated, except their counterparty
(ADR 0016).  A posted entry can be deleted whole, with its postings
(ADR 0011).

### Transaction naming

The present two-sided transaction table does not survive as a ledger table.
Its replacement is `ledger.journal_entry` plus `ledger.posting`.  The user
interface may still call a journal entry a “transaction”.  Imported statement
rows use a distinct name, such as `BankTransaction`, to avoid conflating an
external record with the accounting entry it produces.

## Examples

Assume EUR is the relevant currency in the examples, except where noted.

### Buy groceries from Tesco using Revolut EUR

```text
JournalEntry: counterparty = Tesco
  Debit  Expenses:Groceries:EUR   EUR45
  Credit Assets:Revolut:EUR       EUR45
```

### Transfer EUR100 from Revolut EUR to IBKR

```text
JournalEntry
  Debit  Assets:IBKR:Cash:EUR     EUR100
  Credit Assets:Revolut:EUR       EUR100
```

### Buy one AAPL share for EUR100 in IBKR

```text
JournalEntry
  Debit  Assets:IBKR:Investments:EUR EUR100
  Credit Assets:IBKR:Cash:EUR     EUR100

PositionMovement
  IBKR / AAPL / units_delta = +1 / cost = EUR100
```

### Exchange USD100 in Revolut USD for EUR90 in Revolut EUR, charging a USD1 fee

```text
JournalEntry metadata
  type: fx_conversion
  effective_rate: "90/99"

  USD postings
    Debit  Assets:FX-Clearing:USD  USD99
    Debit  Expenses:FX-Fees:USD     USD1
    Credit Assets:Revolut:USD     USD100

  EUR postings
    Debit  Assets:Revolut:EUR      EUR90
    Credit Assets:FX-Clearing:EUR  EUR90

```

## Consequences

- Account balances are derived by summing posted entries rather than by
  interpreting source and destination columns.
- Transfers, fees, tax withholding, and split categories can use more than
  two postings without special transaction types.
- The model gains an auditable, immutable history. ADR 0017 removes the
  explicit reversal semantics it also had.
- The implementation must create and post entries and their postings in one
  database transaction, with database-level per-currency balance checks.
- Existing account, transaction, balance, asset, and workflow code must be
  refactored to use the new types; no compatibility layer is required.
