# ADR 0019: Currency-neutral categories, as a dimension of postings

- Status: Proposed
- Date: 2026-11-14
- Depends on: [ADR 0001](0001-double-entry-ledger.md) (one currency per ledger
  account), [ADR 0003](0003-categories-and-tags.md) (categories as ledger
  accounts, which this reverses), [ADR 0008](0008-intent-level-transactions.md)
  (the transactions that name a category),
  [ADR 0012](0012-deleting-accounts.md) (deleting and merging categories), and
  [ADR 0017](0017-editing-journal-entries.md) (a posted entry can be edited)
- Amends: ADR 0003 (what a category is), ADR 0005 (the unique key of a
  category and of a top-level account), ADR 0006 (the shared accounts),
  ADR 0008 and ADR 0009 (how a transaction and a vest name a category),
  ADR 0012 (deleting a category, "Uncategorized"), ADR 0017 (the update
  workflows)
- Related: [ADR 0018](0018-transactions-page.md) (the category filter).
  Issue: logicorg/ivfs#6

## Context

A category is a ledger account, and a ledger account holds one currency
(ADR 0003). So "Groceries" in EUR and "Groceries" in USD are two unrelated
accounts that a person must create, rename, and close one by one. People do
not think that way: a category is a name, and a transaction in any currency
can fall under it.

One currency per account is worth keeping. It is what makes a balance one
number and what the balance check of every entry relies on (ADR 0001). What
has to give is the idea that a category is an account.

An earlier draft of this ADR kept one account per category and currency and
tied them together with a shared key. It needed a name lookup on every insert,
ancestor accounts created on first use of a currency, rename and close fanned
out over several rows, a grouping rule checked across currencies, and three
invariants that only application code kept. All of that exists to hide the
copies. This version has no copies.

What a category has to do:

- classify an income or an expense, in any currency;
- nest, at most two parent links deep (ADR 0003);
- give exact totals per category and currency over a period;
- be renamed, closed, deleted, and merged as one thing.

None of these needs a balance of its own, which is the one thing a ledger
account is for.

## Decision

A category is a row of a new `ledger.category` table, and a posting may carry
one. Money spent or earned posts to one shared "Expenses" or "Income" ledger
account per currency. A category has no currency.

### The tables

```sql
CREATE TABLE ledger.category (
  id         bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  parent_id  bigint REFERENCES ledger.category (id),
  kind       text NOT NULL,          -- income or expense
  name       text NOT NULL,
  closed_at  timestamptz,
  created_at timestamptz NOT NULL DEFAULT current_timestamp,
  updated_at timestamptz NOT NULL DEFAULT current_timestamp
);

CREATE UNIQUE INDEX category_name_lower_key
  ON ledger.category (parent_id, kind, lower(name)) NULLS NOT DISTINCT;

ALTER TABLE ledger.posting
  ADD COLUMN category_id bigint REFERENCES ledger.category (id);
```

The hierarchy keeps ADR 0003's rules, now on the category: at most two parent
links, a child has its parent's kind, no cycles, and a parent is open when a
child is created or moved under it. A category's kind never changes.

### The shared accounts

"Expenses" and "Income" join "Opening balances" and "FX clearing" as shared
ledger accounts (ADR 0006): top-level, with no financial account, one per
currency, found by name and kind, and created by retrieve-or-insert through
`CreateLedgerAccount` the first time a currency needs one.

They are the only income and expense ledger accounts. `CreateLedgerAccount`
refuses any other name for those two kinds, so there is one way to classify
spending, not two. The rules ADR 0012 gives the shared accounts apply: they
cannot be renamed, moved, or closed, nor deleted while they have postings.

### What a posting may carry

| Rule | Where it is checked |
| --- | --- |
| A category only on a posting to an income or expense account, and of that account's kind. | The deferred check of `validate_posted_journal_entry`, which already runs for every entry a transaction touched. |
| The category has no children: postings target leaves. | The activities that write postings. |
| The category is open. | The activities that write postings, as for a closed account. |
| A category with postings cannot become a parent. | Creating a category and moving one. |

A posting to "Expenses" or "Income" with no category is valid. It is
uncategorized.

The two leaf rules cannot both be bypassed by a race. Writing a posting locks
its categories `FOR SHARE`, after its ledger accounts and in id order, as it
locks the accounts, and then checks for children. Creating or moving a child
locks the new parent `FOR NO KEY UPDATE` and then checks for postings.
Whichever commits second sees the first. Close and delete take the same lock.

### What the ledger still proves

Nothing about double entry changes. An expense is still one balanced entry:

```text
Dr Expenses (EUR)  12.50   category: Groceries
Cr Cash (EUR)      12.50
```

Every entry balances in each currency, and the trigger that checks it is
untouched. The balance of "Expenses (EUR)" is everything spent in EUR.

What changes is how fine the chart of accounts is. The ledger proves the total
earned and spent per currency; the breakdown by category is a property of the
postings. It cannot drift from the total: a posting has at most one category,
so the categories of an account and its uncategorized postings always sum to
the account's balance.

The type of a transaction is still not stored, and is derived as ADR 0008
says: a posting to an expense account makes an expense, one to an income
account an income.

### Categories

The category workflows keep their names and act on `ledger.category`. They
move from `finance/workflows/facade/categories` to
`finance/workflows/ledger/categories`, one workflow per activity, with the
activities in `finance/activities/ledger/categories`: a category is now an
entity, with nothing underneath for a facade to compose.

| Workflow | Rule |
| --- | --- |
| `CreateCategory` | Takes a name, a kind, and an optional parent. No currency. Retrieve-or-insert on `(parent_id, kind, lower(name))`. |
| `RenameCategory` | One row. `ErrResourceConflict` when a sibling of the same kind holds the name. |
| `MoveCategory` | Changes `parent_id`. Allowed only while the category and its descendants have no postings, as ADR 0003 decided. |
| `CloseCategory` | One row. Refused with open children. Idempotent. |
| `DeleteCategory` | Below. |
| `RetrieveCategory`, `ListCategories` | Return categories: `id`, `parentId`, `kind`, `name`, `closedAt`. The currency filter goes. |

"Uncategorized" is refused as a name, compared as the unique keys are, so the
label a client gives the postings with no category is never also a category.

### Posting

`PostingInput` gains an optional `categoryId`, which `PostJournalEntry` and
`ReplaceJournalEntry` write with the posting.

The workflows that name a category take its id instead of a ledger account's:

| Workflow | Field today | Field now |
| --- | --- | --- |
| `RecordExpense`, `RecordIncome`, `UpdateExpense`, `UpdateIncome` | `categoryLedgerAccountId` | `categoryId` |
| A vest, in `RecordInvestment` and `UpdateInvestment` | `incomeCategoryLedgerAccountId`, `taxCategoryLedgerAccountId` | `incomeCategoryId`, `taxCategoryId` |
| The fee of `RegisterFXConversion` | `fee.ledgerAccountId` | `fee.categoryId` |

Each retrieves the category, checks its kind, finds the shared account of the
money's currency with a helper modelled on `ensureFXClearing`, and posts to it
with the category. The check that the category and the balance share a
currency goes, since a category has none. A category stays required where it
is today.

A request retried with a known idempotency key still returns its entry
unchanged (ADR 0005).

### Recategorising

Changing the category of an expense is `UpdateExpense` with another
`categoryId`, which replaces the entry (ADR 0017). The posting stays on the
same account, so no balance moves and no account has to exist first. Giving an
uncategorized expense a category is the same call.

### Deleting and merging a category

`DeleteCategory` is one activity in one transaction. It replaces what ADR 0012
does for a category.

- Without a replacement, the category's postings lose their category.
- With one, they take the replacement, which must be an open category of the
  same kind with no children, and not the category itself. Anything else
  fails with `ErrInvalidParameter` before a row changes.
- A category with children is refused with `ErrResourceConflict`.
- The row is then deleted, which frees its name. Deleting a category that does
  not exist succeeds.

No entry is deleted and no posting changes account, so totals per currency are
untouched, no holding changes, and there are no warnings to return. The
deletion preview returns the number of entries affected in each currency.

The "Uncategorized" stray accounts go, with the part of `AssignUntracked` that
moved lines off them. "Untracked accounts" and the rest of ADR 0012 stand.
Deleting a balance still deletes an expense paid from it, since the entry's
other line is on a shared account.

### Reads and clients

- `CategoryTotals` groups by `posting.category_id` and the account's currency.
  A row is a category, a kind, a currency, and a total. A row with no category
  is the uncategorized total of that kind and currency. Totals stay per leaf;
  rolling them up to a parent is left to the client, as today.
- `ListJournalEntries` gains the `CategoryIDs` filter ADR 0018 plans: entries
  with a posting in any of these categories or their descendants. It replaces
  the ledger-account filter as the way to list a category's entries.
- A posting in an API response carries its `categoryId`.
- The UI shows one tree of categories, offers every open leaf whatever the
  currency of the balance, and never asks for a category's currency.
- MCP's `create_category` loses `currencyId`.

### What is removed

- `ledger_account.parent_id`, its foreign key, `MoveLedgerAccount`, and the
  depth, cycle, and parent checks of ledger accounts. Only categories used
  the hierarchy. The unique key of a ledger account with no financial account
  becomes `(kind, currency_id, lower(name))`.
- The category facade package, `refuseNonCategory`, and `checkSameCurrency`
  for an expense and an income.
- The "Uncategorized" stray, its reserved name among ledger accounts, and its
  branch in `StrayLedgerAccount`.

## Examples

Groceries exists, and 20 USD is spent on it for the first time in USD:

```text
Categories:  Food
              └ Groceries

RecordExpense{balance: <Cash (USD)>, categoryId: <Groceries>, amount: 20, ...}
  CreateLedgerAccount{Expenses, expense, USD}   -> found, or inserted
  -> entry: Dr Expenses (USD) 20 [Groceries] / Cr Cash (USD) 20

Categories:  unchanged
```

Totals for the month, with spending in two currencies:

```text
CategoryTotals{from, to}
  -> {Groceries, expense, EUR, 312.40}
     {Groceries, expense, USD,  20.00}
     {<none>,    expense, EUR,  15.00}
```

Renaming, recategorising, and merging:

```text
RenameCategory{<Groceries>, "Supermarket"}
  -> one row; every posting follows

UpdateExpense{journalEntryId: 41, categoryId: <Eating out>, ...}
  -> Dr Expenses (EUR) 12.50 [Eating out] / Cr Cash (EUR) 12.50

DeleteCategory{<Supermarket>, replacementId: <Groceries>}
  -> the postings of Supermarket, in every currency, are in Groceries
```

A vest of 100 units at 10 USD with 40 withheld:

```text
Cr Income (USD)       1000  [RSU compensation]
Dr Investments (USD)   600
Dr Expenses (USD)      400  [Tax withheld]
```

Requests that fail:

```text
RecordExpense{... categoryId: <Food>}                Food has children
  -> ErrInvalidParameter: "Food" groups other categories; pick one of them

RecordExpense{... categoryId: <Salary>}
  -> ErrInvalidParameter: "Salary" is an income category, not an expense one

CreateCategory{name: "Fresh", parentId: <Groceries>} Groceries has postings
  -> ErrResourceConflict: a category with postings cannot group others

RenameCategory{<Groceries>, "Transport"}             a sibling has the name
  -> ErrResourceConflict: a category named "Transport" already exists
```

## Alternatives considered

- **One account per category and currency, tied by a shared key.** The earlier
  draft. It leaves ADR 0003 standing, at the cost listed in the Context: the
  copies have to be created, kept alike, and hidden, by code alone.
- **A category table with one ledger account per currency under it**, as a
  financial account has. Names and the hierarchy are stored once, and it keeps
  a balance per category. But the per-currency accounts remain, created on
  first use, and recategorising still moves a posting between accounts, the
  target of which may not exist yet.
- **The currency on the posting, with one currency-less account per
  category**, as plain-text accounting tools do. The cleanest model of a
  category, but it bends ADR 0001's rule for two kinds of account, and the
  balance check and every read of an account's currency would have to follow
  the posting instead.
- **The category on the journal entry.** One column fewer to check, but a
  vest has two categories and a split has several.
- **Tags as categories.** A tag is flat, optional, and many to an entry
  (ADR 0003). A category is a tree, has a kind, and is one per posting, which
  is what makes its totals sum to the account's.
- **One "Expenses" account in all currencies.** It would need the currency on
  the posting, as above.
- **An "Uncategorized" category row** for postings with none. It would have to
  be created, reserved, and protected from deletion, to say what a null
  already says.
- **Lifting ADR 0003's restriction on moving a used category.** A move no
  longer touches a posting, so it is now only a matter of policy. It is left
  as it is: this ADR changes what a category is, not what a person may do to
  one.

## Consequences

- A person creates a category once and uses it in any currency. There is
  nothing per currency to create, keep alike, or clean up.
- ADR 0003's central decision is reversed: there is a category table again.
  It is not the table that ADR removed, which classified transactions beside
  the ledger. Money still lives only in postings.
- The chart of accounts no longer lists categories. A report read straight
  from account balances shows income and expenses per currency, and needs the
  postings' categories for the breakdown.
- A category has no balance. What was read as the balance of a category's
  account is its total over a period.
- Three rules that the account tree stated by structure, kind, leaf, and open,
  are now checks: one in the database, two in the activities that write
  postings.
- Recategorising, deleting, and merging a category stop being accounting
  writes. They change one column and cannot unbalance an entry.
- Making a category optional on an expense is now only a change of validation.
- The categories code, the transaction and vest workflows, the conversion
  fee, the totals report, and the UI's category screens are rewritten. The
  ledger-account hierarchy and most of ADR 0012's category handling are
  deleted.
- The schema is created with `IfNotExists`, so existing databases do not gain
  the table or the column. As in ADR 0005, the system is pre-alpha and
  databases are recreated rather than migrated.
- `design_principles.md` and `entity_relationships.md` change when this is
  implemented: a category is a table, and ledger accounts have no hierarchy.

## Amendments to earlier ADRs

ADR 0003:

- A category is a row of `ledger.category`, not a ledger account, and has no
  currency. A posting to an income or expense account may carry one.
- The hierarchy and its rules move from ledger accounts to categories. Ledger
  accounts have no parent.
- The category workflows act on categories and no longer delegate to the
  ledger-account ones.

ADR 0005:

- A category is unique on `(parent_id, kind, lower(name))`, a missing parent
  counting as one value.
- A ledger account with no financial account is unique on
  `(kind, currency_id, lower(name))`.

ADR 0006:

- "Expenses" and "Income" are shared accounts, one per currency, and the only
  income and expense ledger accounts.

ADR 0008:

- An expense debits "Expenses", and an income credits "Income", in the
  balance's currency, with the category on that posting. The input names the
  category by `categoryId`, and the currency check between the two is gone.

ADR 0009:

- A vest credits "Income" and debits "Expenses" in its currency, with
  `incomeCategoryId` and `taxCategoryId` on those postings.

ADR 0012:

- Deleting a category clears it from its postings, or sets the replacement.
  It deletes no entry and moves no posting.
- The "Uncategorized" stray accounts are removed. `AssignUntracked` handles
  "Untracked accounts" only.
- "Expenses" and "Income" are protected as the other shared accounts are.

ADR 0017:

- `UpdateExpense`, `UpdateIncome`, and `UpdateInvestment` take category ids,
  as the record workflows do. Changing only the category of an entry is such
  an update.

ADR 0018 (proposed):

- `CategoryIDs` names categories, and matches on `posting.category_id`.
