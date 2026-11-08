# ADR 0005: Uniqueness and idempotent creation of accounts and reference data

- Status: Implemented
- Date: 2026-09-27
- Depends on: [ADR 0002](0002-ledger-domain-and-application-api.md) and [ADR 0003](0003-categories-and-tags.md)
- Amends: ADR 0002 (persistence invariants and creation semantics) and ADR 0003 (category names)

## Context

ADR 0002 requires every activity that changes state to be independently
idempotent, and the repository's convention for creation is retrieve-or-insert:
look the entity up by its unique identifier, return it unchanged if it exists,
and insert it otherwise. Three creation activities do not follow it:

- `CreateFinancialAccount` always inserts. A retry creates a second financial
  account, and a second cash ledger account when one is requested.
- `CreateLedgerAccount`, and `CreateCategory`, which wraps it, always insert.
- `CreateCounterparty` always inserts. `Counterparty.name` is unique, so a retry
  fails with a raw unique violation instead of returning the counterparty.

`FinancialAccount` and `LedgerAccount` have no uniqueness rule at all, so there
is nothing to retrieve them by. This matters more once these commands are
exposed to an MCP client: Temporal retries an activity whose response is lost
after its insert committed, and an LLM may issue the same create twice.

An idempotency key per create request was considered and rejected (see
Alternatives). It protects against a retried request, but not against the same
account being asked for twice, which only a rule about the data itself can
catch. The rules below are chosen so that an entity's uniqueness key is also
its idempotency key.

## Decision

### Uniqueness keys

| Entity | Unique on | Applies to |
| --- | --- | --- |
| `Institution` | `name` | every institution |
| `FinancialAccount` | `(institution_id, name)` | every financial account |
| `FinancialAccount` | `(institution_id, external_id)` | financial accounts with an external ID |
| `LedgerAccount` | `(financial_account_id, currency_id, name)` | ledger accounts that belong to a financial account |
| `LedgerAccount` | `(parent_id, kind, currency_id, name)`, a missing parent counting as one value | ledger accounts with no financial account: categories, equity, and clearing accounts |
| `Counterparty` | `name` | every counterparty |
| `Currency` | `code`, and `name` | every currency (unchanged) |

The scopes follow how the entities are identified in practice. One institution
holds several financial accounts, such as a personal, a joint, and a business
account at Revolut, so a financial account's name is unique within its
institution only. A financial account holds one balance per currency, and may
hold several in the same currency when money is kept apart, so a ledger
account's name is unique within its financial account and currency: "Cash" in
EUR, "Cash" in USD, and "Savings vault" in EUR can all belong to Revolut
Personal. A ledger account that belongs to no financial account is unique
among its siblings, where the kind is part of the key because a category name
such as "Interest" or "Other" legitimately exists both as income and as an
expense. Currency and kind only distinguish top-level accounts in practice,
since a child always has its parent's kind and currency.

`external_id` identifies a financial account in the real world, such as an
IBAN, so two accounts that share one within an institution are always an error.
It is optional, so it is a constraint and not the key a create retrieves by.

A counterparty's `external_id` stays unconstrained until imports give it a
meaning.

### Case

Names are compared case-insensitively and stored as supplied, trimmed of
surrounding whitespace: "Revolut Personal" and "revolut personal" are the same
account, and it keeps the casing it was created with. Currency codes have one
canonical form, so they are upper-cased on every write and every lookup, and
compared exactly.

`external_id` is compared exactly.

### Names are reserved forever

Every key applies to closed rows as well as open ones. A name used by a closed
financial account or ledger account cannot be reused by a new one.

### Creation is retrieve-or-insert on the key

A create looks the entity up by its key. If it exists, the create returns it
unchanged and succeeds, whatever the other fields of the request; it does not
compare or reconcile them. Otherwise it inserts the entity.

This holds when the existing row is closed: the create succeeds and returns the
closed row, and a later write to it fails with `ErrResourceClosed`. Reporting
the closed row at creation time instead is left for a later revision.

A retried create returns the same result as the first call. For
`CreateFinancialAccount` this includes the cash ledger account created with it,
which a retry looks up by its own key.

A create whose key is free but which collides on another rule, such as a new
financial account with the external ID of another account, fails with
`ErrResourceConflict`.

### Renames and moves

Renaming or moving an entity onto a key that another row holds fails with
`ErrResourceConflict`. This covers `RenameLedgerAccount`, `MoveLedgerAccount`,
the category rename and move, `RenameCounterparty`, a rename through
`UpdateInstitution`, and a change of name or external ID through
`UpdateFinancialAccountMetadata`.

### Enforcement

The activities own the rules: they trim and normalise the input, look the entity
up with `lower(name) = lower(?)` (or the upper-cased currency code), and insert
when nothing matches.

A unique index enforces each key in the database as well, because a check in
the activity alone races: two concurrent creates can both find nothing and both
insert. With the index, the second insert fails, the activity recognises the
unique violation through `sqllib.AlreadyExists`, and it retrieves again to
return the row the other call created. The indexes are installed with the other
constraints in `pkg/sqllib`:

```sql
CREATE UNIQUE INDEX ON institution.institution (lower(name));
CREATE UNIQUE INDEX ON account.financial_account (institution_id, lower(name));
CREATE UNIQUE INDEX ON account.financial_account (institution_id, external_id);
CREATE UNIQUE INDEX ON ledger.ledger_account (financial_account_id, currency_id, lower(name))
    WHERE financial_account_id IS NOT NULL;
CREATE UNIQUE INDEX ON ledger.ledger_account (parent_id, kind, currency_id, lower(name))
    NULLS NOT DISTINCT WHERE financial_account_id IS NULL;
CREATE UNIQUE INDEX ON ledger.counterparty (lower(name));
CREATE UNIQUE INDEX ON tag.tag (lower(name));
```

The case-sensitive unique constraints on `institution.name`,
`counterparty.name` and `tag.name` are replaced by the indexes above. Two `NULL` values are
never equal in a unique index by default, so without `NULLS NOT DISTINCT` two
top-level accounts with the same name would not collide. `NULLS NOT DISTINCT`
makes a missing parent count as one value. A missing `external_id` is `NULL` and
never collides, which is the default behaviour that index needs.

`NULLS NOT DISTINCT` needs PostgreSQL 15 or later, which makes 15 the minimum
version the schema supports. It was preferred over the version-neutral
`COALESCE(parent_id, 0)`, which indexes an expression with `0` standing for "no
parent": `NULLS NOT DISTINCT` states the rule directly, needs no stand-in
value, and indexes `parent_id` itself, so a lookup of top-level accounts by
`parent_id IS NULL` can use the index.

### The cash ledger account

The cash ledger account that `CreateFinancialAccount` creates is named "Cash".
Its financial account and currency already identify it, so the name no longer
repeats them.

## Alternatives considered

- **A request idempotency key on every create**, as journal entries have. It
  makes a retried request safe without any rule on the data, but two separate
  requests for the same account still create two accounts. Once names are
  unique within a scope, the name is a natural idempotency key and a separate
  one adds nothing.
- **Name unique within the institution, for a ledger account.** It forces every
  name to repeat the currency ("EUR cash", "USD cash"), although a ledger account
  is in exactly one currency and the currency is already part of what it is.
- **Kind excluded for ledger accounts without a financial account.** It would
  forbid an "Interest" income category next to an "Interest" expense category.
- **Names lower-cased when stored.** It makes a plain unique constraint enough,
  but loses the casing the user chose ("IBKR" would read "ibkr").
- **Uniqueness enforced by the activity only.** It is simpler, but concurrent
  creates can both pass the check.
- **Names free to reuse once an account is closed**, through a partial index on
  open rows. It allows opening a new "Revolut Joint" after closing the old one,
  but a create would then have to choose between an open and a closed match.
  Reserving names is simpler and can be relaxed later.
- **`COALESCE(parent_id, 0)` instead of `NULLS NOT DISTINCT`.** It works on
  every PostgreSQL version, but indexes an expression with a stand-in value
  rather than the column, and states the rule less directly. The minimum
  version it avoids, PostgreSQL 15, was accepted instead.
- **`ErrResourceClosed` when a create matches a closed row.** It tells the
  caller early that the account cannot be used, at the cost of an exception to
  retrieve-or-insert. Deferred.

## Consequences

- `CreateFinancialAccount`, `CreateLedgerAccount`, `CreateCategory`, and
  `CreateCounterparty` become safe to retry and to expose to an MCP client, and
  each gains an idempotency test.
- A spoken reference to an account resolves to at most one row, given its scope:
  an institution and a name, or a financial account, a currency, and a name.
- `InsertInstitution` and `InsertCurrency`, which already retrieve-or-insert,
  change only in how they compare: institution names case-insensitively, and
  currency codes upper-cased.
- Renames and moves gain a conflict case.
- Existing databases with duplicate names cannot install the new indexes. The
  system is pre-alpha and databases are recreated rather than migrated.
- The schema requires PostgreSQL 15 or later, because of `NULLS NOT DISTINCT`.

## Amendments to earlier ADRs

ADR 0002:

- The persistence invariants gain the uniqueness keys above.
- "Independently idempotent" is defined, for creation, as retrieve-or-insert on
  the entity's uniqueness key.
- The first cash ledger account created with a financial account is named
  "Cash".

ADR 0003:

- A category's name is unique among its siblings of the same kind and
  currency, compared case-insensitively. Renaming or moving a category onto a
  taken name fails with `ErrResourceConflict`.
