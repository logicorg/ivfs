# ADR 0016: Currency-neutral categories

- Status: Proposed
- Date: 2026-11-07
- Related: [ADR 0003](0003-categories-and-tags.md) (categories as ledger
  accounts), [ADR 0005](0005-uniqueness-and-idempotent-creation.md) (the
  uniqueness key of a category), [ADR 0008](0008-intent-level-transactions.md)
  (the transactions that post to a category), [ADR 0015](0015-net-worth-over-time.md)
  (`user_settings.default_currency_id`). Issue: logicorg/ivfs#6

## Context

A category is a ledger account, and a ledger account holds one currency
(ADR 0003). So "Groceries" in EUR and "Groceries" in GBP are two unrelated
accounts that the user must create, name, rename, move, and close one by one.
Users do not think that way: a category is a name, and a transaction in any
currency can fall under it.

The accounting side is fine and should not change: a balance in one currency
per account keeps postings, roll-ups, and reports exact (ADR 0001, ADR 0003).
What is missing is a way to say "these accounts are one category".

Constraints from the current code:

- A child account must have its parent's kind and currency (ADR 0003), so a
  per-currency copy of a child needs a per-currency copy of every ancestor.
- The unique key of a category is `(parent_id, kind, currency_id, name)`
  (ADR 0005), so same-named siblings in different currencies are already
  legal, but nothing relates them.
- `RecordExpense` and `RecordIncome` take a `CategoryLedgerAccountID` and fail
  when its currency differs from the balance's (ADR 0008).
- Category totals (`CategoryTotals`) are per ledger account, so one name
  currently yields one row per currency.
- A default currency per user exists (`user_settings.default_currency_id`,
  ADR 0015); it may be unset.

## Decision

Introduce a currency-neutral **category** as the thing users create, name,
move, and close, and keep **one leaf ledger account per category and
currency**, created when first needed.

### Model

- A `ledger.category` table holds `id`, `parent_id` (self-reference, same
  depth limit as ADR 0003), `kind` (`income` or `expense`), `name`, and a
  closed flag. It is unique on `(parent_id, kind, name)`, a missing parent
  counting as one value.
- `ledger_account` gains a nullable `category_id`. Unique on
  `(category_id, currency_id)` where set. Category accounts carry no
  `parent_id` of their own: the hierarchy lives on the category, so a parent
  needs no account in every currency its children use.
- Postings still target a leaf `ledger_account`; ledgers, balances, and the
  invariants of ADR 0001 are untouched.
- A parent category's balance is a reporting roll-up, per currency, of the
  accounts of its descendants (as in ADR 0003, but joined through
  `category_id`).

### Creation

- `CreateCategory` takes a name, kind, and optional parent category. It
  creates the category and, as the issue proposes, one account in the user's
  default currency. If no default is set it fails with
  `ErrInvalidParameter` telling the caller to set one in the settings, or to
  pass a currency explicitly. It remains retrieve-or-insert on the key above.
- A new activity `RetrieveOrCreateCategoryAccount(categoryID, currencyID)`
  returns the category's account in that currency, inserting it when absent.
  It owns its transaction and is idempotent (AGENTS.md): a retry returns the
  existing account unchanged.

### Posting

- The transaction facades (`RecordExpense`, `RecordIncome`) take a
  `CategoryID` instead of a ledger account id. They resolve the category's
  account in the balance's currency through the activity above, then post as
  today. The currency check against the balance becomes true by construction.
- Accounts are never created eagerly for currencies the category never sees.

### Lifecycle

Rename, move, and close act on the category. Close also closes its accounts,
which keeps history and blocks new postings. Delete is allowed only when no
account of the category has postings (ADR 0012 rules apply per category).

## Options considered

1. **Eager accounts**, one per category and currency at creation. Simple, but
   most would stay empty forever, they appear in every account list, and
   adding a currency would mean back-filling every category.
2. **Lazy accounts matched by name**, with no new table: a category is the set
   of same-named, same-kind accounts under equally named ancestors. Smallest
   schema change. Rejected: identity is a string, so rename, move, and close
   must fan out across accounts and race with concurrent creation, and every
   ancestor must exist in every currency (the same-currency parent rule).
3. **Lazy accounts with a persisted category (chosen).** One new table and
   one nullable column. Identity is an id, the hierarchy is stored once, and
   fan-out is limited to close.
4. **Currency-neutral group accounts**, the future ADR 0003 mentions: a
   neutral parent account with per-currency children. Close in spirit to
   option 3, but it bends the ledger account's one-currency meaning for
   non-posting rows. A separate category table keeps `ledger_account`
   honest.

## Consequences

- Partly supersedes ADR 0003: its removal of a `Category` table is reversed,
  in a narrower form (no accounting data on it), and the "parent and child
  share a currency" rule no longer applies to category accounts. The ADR 0005
  key for categories moves to the new table.
- Existing categories need a one-off migration: group same-named accounts
  under one category row. The project is pre-alpha, so recreating the schema
  is an acceptable alternative (ADR 0003 did this).
- The API changes: `GET /api/categories` returns categories with the
  currencies they hold accounts in; the record endpoints take `categoryId`;
  totals gain a roll-up per category and currency. The UI's category form
  drops the currency picker.
- A user can no longer pick a category's account in a currency by hand;
  reports needing the account still read `ledger_account`.
- Net worth (ADR 0015) is unaffected: it reads asset and liability balances.

## Open questions

1. Should `CreateCategory` still accept an optional currency, or always use
   the default? Proposed: optional, defaulting to the user's setting.
2. If a category is closed and a posting in a new currency arrives, the
   posting fails as closed (consistent with ADR 0003). Confirm that reopening
   is out of scope.
3. Migration vs. recreating the schema for existing data.
