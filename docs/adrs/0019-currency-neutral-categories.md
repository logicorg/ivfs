# ADR 0017: Currency-neutral categories

- Status: Proposed
- Date: 2026-11-08
- Depends on: [ADR 0003](0003-categories-and-tags.md) (categories as ledger
  accounts), [ADR 0005](0005-uniqueness-and-idempotent-creation.md) (the
  uniqueness key of a category), [ADR 0008](0008-intent-level-transactions.md)
  (the transactions that post to a category), and
  [ADR 0012](0012-deleting-accounts.md) (deleting and merging categories)
- Amends: ADR 0003 (a category's required currency), ADR 0005 (rename and move
  conflicts), ADR 0008 (an expense or income in another currency), ADR 0009
  (a vest's categories), ADR 0012 (deleting a category)
- Related: [ADR 0015](0015-net-worth-over-time.md) (the user's default
  currency). Issue: logicorg/ivfs#6

## Context

A category is a ledger account, and a ledger account holds one currency
(ADR 0003). So "Groceries" in EUR and "Groceries" in USD are two unrelated
accounts that a person must create, rename, and close one by one. People do
not think that way: a category is a name, and a transaction in any currency
can fall under it.

The accounting side should not change. One currency per account keeps
postings, roll-ups, and reports exact (ADR 0001, ADR 0003). What is missing is
a way to say "these accounts are one category", and a way to get the account
for a new currency without asking.

The constraints that shape the design:

- A child has its parent's kind and currency (ADR 0003), so an account for a
  child in a new currency needs one for each of its ancestors.
- The unique key of a category is `(parent_id, kind, currency_id, lower(name))`
  (ADR 0005). The same index also serves equity and clearing accounts.
- `RecordExpense`, `RecordIncome`, and a vest fail when a category's currency
  differs from the money's.
- A person's default currency exists but may be unset (ADR 0015).

## Decision

Keep one ledger account per category and currency, created when first needed.
Mark the accounts of one category with a shared key. There is no category
table: ADR 0003's rule stands.

### The key

`ledger.ledger_account` gains a column:

```sql
category_key uuid NOT NULL DEFAULT gen_random_uuid()
```

**A category is the set of ledger accounts that share a `category_key`.**

Names match once, when an account is inserted. From then on the key alone
identifies the category, so a rename cannot break it.

Every account gets a key, including balances and equity accounts, for which it
means nothing: each has a key of its own, so its category is that one row. This
is what lets rename, close, and move below act on the key with no check of the
account's kind.

### Joining a category

Before an income or expense account with no financial account is inserted, one
lookup looks for the category it belongs to:

```sql
SELECT la.category_key, la.closed_at
FROM ledger.ledger_account AS la
LEFT JOIN ledger.ledger_account AS p ON p.id = la.parent_id
WHERE la.financial_account_id IS NULL
  AND la.kind = ?
  AND lower(la.name) = lower(?)
  AND p.category_key IS NOT DISTINCT FROM ?  -- the new parent's key; NULL at top level
LIMIT 1
```

If it finds one, the new account takes its key and its `closed_at`. If not, the
database generates a key and the account starts a new category.

The lookup runs inside `CreateLedgerAccount` and when `StrayLedgerAccount`
creates an "Uncategorized", so that account is one category across currencies
too. `CreateLedgerAccount` stays retrieve-or-insert on the ADR 0005 key, which
is unchanged: the lookup only decides what an insert writes.

Inheriting `closed_at` means a closed category stays closed in every currency,
with no check in the workflows that post.

### Creating a category

`CreateCategory`'s `currencyId` becomes optional. The account is created:

1. in the parent's currency, if there is a parent. A given currency that
   differs from the parent's fails, as it does today;
2. otherwise in the given currency;
3. otherwise in the default currency;
4. otherwise it fails with `ErrInvalidParameter`, saying to choose a currency
   or set a default in Settings.

Creating "Food" in EUR when "Food" exists only in USD adds an account to the
existing category.

### Posting

`RecordExpense`, `RecordIncome`, and a vest keep their input: the
`categoryLedgerAccountId` may name the category's account in any currency.

After their other checks and just before posting, they call
`EnsureCategoryAccount`, a helper in the `finance/workflows/ledger/accounts`
package modelled on `ensureFXClearing`. When the named account is already in
the money's currency it returns it. Otherwise it walks up to the root, at most
two retrievals, and calls `CreateLedgerAccount` from the root down in that
currency, and the entry posts to the leaf.

No new activity is needed, and each step is idempotent. A closed account is
still left to `PostJournalEntry` (ADR 0008), so a request retried with a known
idempotency key still returns its entry. If posting fails after the accounts
were created they stay, empty and harmless.

### Grouping categories

ADR 0003 says postings should target leaves, and nothing enforces it. With
several currencies that becomes a bug: "Food" in EUR can have children while
"Food" in USD has none and takes postings, and then "Groceries" can never be
created in USD, because a used account cannot become a grouping account.

So the rule applies to the whole category:

- `RetrieveLedgerAccount` reports whether any account of the category has
  children, and the three workflows above refuse such a category with
  `ErrInvalidParameter`;
- `validateLedgerAccountParent` refuses a parent when any account of its
  category has postings, not only the parent itself.

`PostJournalEntry` does not change.

### Rename, close, and move

These stay in the existing activities, each in one transaction, so the generic
`/api/ledger-accounts/{id}` routes behave the same as the category ones.

| Operation | Rule |
| --- | --- |
| Rename | Updates `WHERE category_key = ?`. Fails with `ErrResourceConflict` when the lookup above finds a different key on the new name. |
| Close | Updates `WHERE category_key = ?`. Fails when any account of the category has active children. Closing again also closes any account of the category still open. |
| Move | Fails with `ErrResourceConflict` when the category has more than one account, and on a different key at the target, as for a rename. |

The rename check is needed because the unique index only sees a conflict within
one currency: "Supermarket" in USD and "Groceries" in EUR share no currency, so
renaming one onto the other would otherwise show two categories of one name.

A move is refused for several accounts because ADR 0003 already allows a move
only before use, and an account per currency exists almost only because of use.
`MoveCategory` first calls `EnsureCategoryAccount` on the target, since the
parent may not yet have an account in the moved category's currency.

### Deleting a category

`DeleteCategory` deletes every account of the category, in id order, each
through the existing `DeleteLedgerAccount` child and its ADR 0012 rules.
Postings move to "Uncategorized" in their own currency.

A category with children in any currency is refused before anything is deleted.
A deletion that fails partway leaves the remaining accounts as the category,
and can be run again on them.

A replacement names any account of another category of the same kind, and
`EnsureCategoryAccount` finds its account in each deleted currency. A
replacement from the same category fails with `ErrInvalidParameter`. The
deletion preview sums the same work across the accounts.

`DELETE /api/ledger-accounts/{id}` still removes only that one account.

### Reads and clients

`ledgerAccounts` and category totals gain `categoryKey`. They stay per account
and per currency: the API does not convert or merge amounts.

The UI groups rows by key into one category, shows totals per currency, and
offers every category whatever the currency of the balance, sending the
category's account in that currency when there is one and any of them when
not. It asks for a currency only when the person has no default. MCP's
`currencyId` becomes optional.

## Examples

Food and Groceries exist in EUR, and 20 USD is spent on Groceries:

```text
Before:  Food EUR (key F)
          └ Groceries EUR (key G)

RecordExpense{balance: <Cash (USD)>, category: <Groceries (EUR)>, amount: 20, ...}
  CreateLedgerAccount{Food, USD}                     -> inserted, joins F
  CreateLedgerAccount{Groceries, USD, <Food (USD)>}  -> inserted, joins G
  -> entry: debit Groceries (USD) 20, credit Cash (USD) 20

After:   Food EUR (F)                 Food USD (F)
          └ Groceries EUR (G)          └ Groceries USD (G)

The UI shows: Food
               └ Groceries
```

Sending it again with the same key returns the entry and creates nothing.

Renaming from either currency renames both:

```text
RenameCategory{<Groceries (USD)>, "Supermarket"}
  -> Supermarket EUR (G), Supermarket USD (G)
```

Requests that fail:

```text
CreateCategory{name: "Food", kind: expense}        with no default currency
  -> ErrInvalidParameter: choose a currency or set a default one in Settings

RecordExpense{... category: <Food (USD)>}          Food has children
  -> ErrInvalidParameter: "Food" groups other categories; pick one of them

RenameCategory{<Groceries (EUR)>, "Transport"}     "Transport" exists only in USD
  -> ErrResourceConflict: a category named "Transport" already exists
```

## Alternatives considered

- **A `ledger.category` table**, with accounts pointing at it. It stores the
  hierarchy once and needs no fan-out for rename, but it reverses ADR 0003, and
  account rows would lose `parent_id`, which the unique key needs:
  "Food > Other" and "Home > Other" would collide. Roll-ups, "Uncategorized",
  and deletion would all need rework. The feature does not justify a second
  source of truth.
- **Accounts matched by name, with no column.** The smallest schema change, but
  identity is a string, so a rename has to find its rows by the old name,
  and every read has to join ancestors by name.
- **A currency-neutral parent account**, the future ADR 0003 mentions. It
  bends the rule that a ledger account has one currency, for rows the ledger
  would then have to treat specially.
- **One account per currency created eagerly.** Most would stay empty forever,
  and adding a currency would mean back-filling every category.
- **A nullable key, set on categories only.** It avoids a meaningless value on
  balances, at the cost of a kind check in every operation that uses the key.
- **A second unique index on `(category_key, currency_id)`.** The name index
  already prevents duplicates, because accounts of one category share a name
  and have parents of one category.
- **Asking for the category's account in a new currency explicitly**, through
  a new activity or endpoint. `CreateLedgerAccount` already retrieves or
  inserts, as `ensureFXClearing` relies on.

## Consequences

- A person creates a category once and uses it in any currency. The ledger,
  its balances, and its invariants do not change.
- Rename and close change several rows at once for a category in several
  currencies, which the activities' comments and tests must cover.
- A rename that lands while another request inserts the old name's account in a
  new currency could leave that account with a new key. No lock prevents it:
  this is a single-person database, and the result is a visible, separate
  category.
- Posting to a grouping category now fails. It was allowed before, against
  ADR 0003's intent.
- A category used in several currencies cannot be moved. Close it and create a
  new one, or delete it into a replacement.
- The schema is created with `IfNotExists`, so existing databases do not gain
  the column. As in ADR 0005, the system is pre-alpha and databases are
  recreated rather than migrated.
- `design_principles.md` and `entity_relationships.md` gain the key when this
  is implemented. A category still is not a table or a kind.

## Amendments to earlier ADRs

ADR 0003:

- `CreateCategory`'s currency is optional.
- A category is the set of accounts sharing a key, one per currency. Parent
  and child accounts still share a currency.
- Rename and close act on the whole category. A move is refused for a category
  with more than one account.
- Postings through `RecordExpense`, `RecordIncome`, and a vest must target a
  category with no children in any currency.

ADR 0005:

- Renaming or moving a category fails with `ErrResourceConflict` when a
  different category holds the name among the target's siblings in any
  currency, not only in its own.

ADR 0008:

- An expense's or an income's category may be in another currency than the
  balance: the category's account in the balance's currency is used, and
  created if needed.

ADR 0009:

- A vest's categories follow the same rule, in the vest's currency.

ADR 0012:

- `DeleteCategory` and its preview act on every account of the category. A
  replacement names any account of another category, and is resolved per
  currency.
