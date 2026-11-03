# ADR 0003: Model categories as ledger accounts and keep tags orthogonal

- Status: Implemented
- Date: 2026-09-05
- Depends on: [ADR 0001](0001-double-entry-ledger.md) and [ADR 0002](0002-ledger-domain-and-application-api.md)

## Context

The previous model had a separate `Category` table and a `Tag` model with an
optional category relationship. That makes accounting classification look like
a second domain model even though a category such as `Expenses:Groceries` is
already naturally represented by a ledger account in the chart of accounts.

Categories and tags also answer different questions:

- a ledger-account hierarchy answers *what account is affected?*;
- a tag answers *what other context applies to this event?*

The application is still pre-alpha. The schema can be recreated directly and
does not need a compatibility or migration layer.

## Decision

### Tables and models

Keep:

- `ledger.ledger_account` as the source of truth for accounting categories,
  account balances, and the chart-of-accounts hierarchy;
- `tag.tag` as an optional, flat vocabulary of horizontal annotations; and
- the ledger, position, and reference-data tables defined by ADR 0001 and ADR
  0002.

Delete:

- `tag.category` and the `Category` model;
- the obsolete `category` schema entry;
- `Tag.CategoryID` and its foreign-key relationship; and
- category-specific persistence code that assumes a `Category` row.

There is no `ledger.category` table. A category is a ledger-account concept,
not another persisted financial object.

Tags are not part of posting validation and do not affect balances. The initial
ledger does not require a tag-assignment table. When annotation is implemented,
the preferred association is a many-to-many `journal_entry_tag` table, with
tags attached to journal entries rather than embedded in accounting lines.

### Ledger-account hierarchy

`LedgerAccount.ParentID` is a self-reference to another ledger account. It
creates a tree such as:

```text
Expenses
└── Food
    └── Groceries
```

The hierarchy has a strict maximum depth of two parent links:

- depth 0: root account (`Expenses`);
- depth 1: child (`Food`); and
- depth 2: grandchild (`Groceries`).

An account may not be its own parent, and a move may not create a cycle. A
parent must be active when a child is created or moved. Until currency-neutral
group accounts exist, parent and child accounts must have the same
`CurrencyID` and `Kind` so that roll-up balances remain meaningful.

Accounts with children are grouping accounts. Postings should target leaf
accounts only. An account that already has posted postings cannot be made into
a grouping account by adding children.

Parent balances are reporting roll-ups of descendant leaf balances, calculated
separately for each currency. `LedgerAccountBalance` for a leaf remains the
direct posting balance; recursive roll-up queries are an application concern.

### Category API

The canonical persistence and activity API remains named after the domain
model:

- `CreateLedgerAccount`;
- `RenameLedgerAccount`;
- `MoveLedgerAccount`;
- `CloseLedgerAccount`; and
- `DeleteLedgerAccount` for an unused account only.

For clients that want a simpler personal-finance vocabulary, retain a thin
`categories` workflow facade:

- `CreateCategory` creates an unlinked `LedgerAccount` with an explicit
  `Kind` of `income` or `expense`, a required currency, and an optional parent;
- `RenameCategory` delegates to `RenameLedgerAccount`;
- `MoveCategory` delegates to `MoveLedgerAccount`; and
- `CloseCategory` delegates to `CloseLedgerAccount`.

The category facade covers income and expense classification. Asset, liability,
equity, and institution-linked accounts use the generic ledger-account API
directly. The category API does not silently default the kind to expense; the
caller must choose income or expense.

The facade may also expose `DeleteOrCloseCategory` for convenience:

- if the account has no postings and no children, delete it;
- if it has posted history and no active children, close it; and
- otherwise return a conflict and require the caller to handle its children.

This convenience operation never deletes an account with financial history.

### Allowed and prohibited mutations

| Operation | Policy |
| --- | --- |
| Create | Allowed when the parent is active, the depth is at most 2, and kind/currency rules are satisfied. |
| Rename | Allowed without changing postings or balances. The stable account ID remains the historical reference. |
| Move/change `ParentID` | Allowed only while the affected subtree has no postings. After use, close the old account and create a new one. |
| Change `Kind` | Allowed only before postings exist; immutable after use. |
| Change `CurrencyID` | Allowed only before postings exist; immutable after use. |
| Change `FinancialAccountID` | Allowed only before postings exist; immutable after use. |
| Close | Allowed when there are no active children. It preserves history and prevents new postings or children. Closing is idempotent. |
| Hard delete | Allowed only when there are no postings, children, or references. Used accounts are closed, never deleted. |

Renaming a used account is mathematically safe but changes the name displayed
by current and historical reports. If stable historical labels become
important, an account-name history table can be added later; it is not needed
for the initial personal backend.

### Tags

Tags remain optional horizontal annotations. A journal entry may eventually
have several tags such as `reimbursable`, `vacation`, or `tax-deductible`, and
the same tag may apply to entries in unrelated ledger-account branches.

Tags have no `ParentID`, no accounting kind, and no currency. They cannot be
moved in an accounting hierarchy. Their simple lifecycle is:

- create a flat tag;
- rename it without changing financial facts;
- archive it once it has been assigned; and
- hard-delete only an unused tag.

Tag assignment and search are deferred until the ledger's critical account,
FX, cashflow, and position workflows are complete.

## Consequences

- The chart of accounts is the only hierarchy used for accounting categories.
- Existing category workflow names can remain a convenient client API without
  introducing a second table or source of truth.
- Account moves cannot silently rewrite historical classifications after an
  account is used.
- Tags remain available for future cross-cutting filters without polluting
  double-entry invariants.
- Category and tag cleanup can be implemented directly; no migration behavior
  is required.
