# RFC 0004: Deleting accounts

- Status: Accepted as [ADR 0012](../adrs/0012-deleting-accounts.md)
- Date: 2026-10-06
- Depends on: [ADR 0011](../adrs/0011-deleting-journal-entries.md) (deleting
  journal entries), [ADR 0010](../adrs/0010-negative-holdings.md) (holdings
  that go negative)
- Related: [ADR 0003](../adrs/0003-categories-and-tags.md) (category
  lifecycle), [ADR 0004](../adrs/0004-position-subledger.md) (position
  movements), [ADR 0005](../adrs/0005-uniqueness-and-idempotent-creation.md)
  (names reserved forever), [ADR 0006](../adrs/0006-intent-level-account-setup.md)
  (account structure and shared accounts)

## Problem

A ledger account, a financial account, or an institution that has been used
can only be closed. Closing keeps every row, and the name stays reserved
forever (ADR 0005), so an account opened by mistake, or twice, or with a typo
in its name, stays in every list and blocks its own name for good.

What a person expects is one of two things:

- **Delete**: the account never existed. Its history goes with it.
- **Merge**: the account was a duplicate of another one. Its history becomes
  the other account's.

RFC 0003 makes a whole journal entry deletable. Deleting an account is harder,
because not every entry on an account belongs to it alone. A transfer from a
bank account to a Revolut account is half the bank's history: deleting the
transfer with Revolut would change the bank's balance, which is still true.

## What is already possible

- `DeleteLedgerAccount` deletes a ledger account with no postings and no
  children. `DeleteOrCloseLedgerAccount` deletes such an account and closes any
  other.
- `Institution.Delete` deletes an institution that no financial account refers
  to, and succeeds when it does not exist.
- There is no delete for a financial account.
- Every foreign key restricts deletion, so a row that is still referenced
  cannot be removed by accident.

## The rule: surviving balances never change

Deleting or merging accounts never changes the balance, or the holdings, of an
account that survives. Everything else follows from it.

A **balance** is an asset or a liability ledger account, such as a "Cash" or a
"Card balance", apart from the shared "FX clearing" accounts (ADR 0006).
Categories, "Opening balances", "FX clearing", and the stray accounts below are
not balances: they explain where money came from or went, and are allowed to
change when the history they explain is deleted.

## Proposal

### Deleting

Deleting an account looks at every journal entry and every position movement
that touches it, and does one of two things with each:

| The entry or movement group also touches... | Then it is... |
| --- | --- |
| only the account being deleted, categories, "Opening balances", "FX clearing", or the stray accounts | deleted, as RFC 0003 does |
| a surviving balance, or units on a surviving financial account other than the stray one | kept, and its lines on the deleted account move to the stray accounts |

An expense paid from the account, its opening balance, and a currency
conversion inside it are deleted. A transfer to or from an account that
survives is kept, so the surviving side stays as it is.

Draft entries that touch a deleted account are deleted, since they have no
effect. A reversal touches the same accounts as the entry it reverses, so the
pair is always treated the same way.

Because the stray accounts are not balances, the order in which two accounts
are deleted does not change the result. A transfer between A and B is kept
when A is deleted, with A's side moved to the stray, and deleted when B is
deleted afterwards, since it then touches only B and the stray. Deleting B
first, or both at once, ends the same way. Deleting several accounts is
therefore deleting each of them in turn.

### The stray accounts

Lines moved off a deleted account need an account to land on. The stray
accounts are created by retrieve-or-insert the first time they are needed:

| Stray | Kind | One per | Receives |
| --- | --- | --- | --- |
| "Untracked accounts" | equity ledger account, top-level | currency | postings of a deleted balance |
| "Uncategorized" | expense, and income, ledger account, top-level | kind and currency | postings of a deleted category |
| "Untracked accounts" | financial account, under an "Untracked" institution | | units of a deleted financial account |

"Untracked accounts" says how much moved between tracked accounts and accounts
that are no longer tracked. It is equity, like "Opening balances", and is not
expected to be zero. Its holdings can be negative: a transfer of 10 shares
from a deleted account to a surviving one leaves -10 in the stray, which is
why this RFC depends on RFC 0005.

A transfer leg, or a trade's or vest's movement on a kept entry, moves to the
stray financial account. A split or an adjustment leg on a deleted account is
deleted rather than moved, even when other legs of its group survive. Each leg
is a fact about its own account only, and a group of either kind stays valid
with fewer legs.

No moved posting or movement records which account it came from, at least at
first (see Options considered).

### Merging

Deleting with a **replacement** merges the account into it. Every posting and
movement of the deleted account moves to the replacement, including those that
deleting alone would have deleted, so no history is lost.

- A ledger account is replaced by an open ledger account with the same kind and
  currency.
- A financial account is replaced by an open financial account. Each of its
  ledger accounts moves to the replacement's ledger account with the same name,
  kind, and currency, which is created there by retrieve-or-insert when
  missing. Its movements move to the replacement.
- An entry or a movement group that ends up with every line on one account is
  deleted, since it changes nothing: a transfer between a duplicate and the
  account it duplicates is one.

### Getting a stray balance back, or rid of it

A line on a stray account belongs to an ordinary journal entry or movement
group:

- **Assigning** moves the stray lines of one entry, or one movement group, to a
  real account: postings to a ledger account in their currency and of a
  compatible kind, movements to a financial account. It is a merge of one entry
  instead of one account, for example after recreating an account deleted by
  mistake, or to give an expense in "Uncategorized" a category.
- **Deleting** the entry or the group, with RFC 0003, removes the stray lines
  and the surviving side together, as if the transfer had never happened.

Assigning only moves lines off a stray account. Since a moved line does not
record where it came from, the stray holds the lines of every deleted account
mixed together, and assigning is done one entry at a time, picked by date,
amount, and the other side.

### Institutions

Deleting an institution deletes each of its financial accounts in turn, then
the institution. Since the order of deletions does not matter, a transfer
between two accounts of the same bank ends up deleted, not moved. An
institution is not merged; its accounts can be merged one at a time first.

### Categories

Deleting a used category moves its postings to "Uncategorized" of its kind and
currency, so an expense stays an expense and the total spent does not change;
only the breakdown loses the category. With a replacement, a category of the
same kind and currency, it is a merge instead. A category with child
categories cannot be deleted until they are.

### Shared and stray accounts

"Opening balances", "FX clearing", and the stray accounts cannot be deleted
while they have postings or movements. They are recognised by name, kind,
currency, and being top-level with no financial account, as ADR 0006 creates
them.

Recognising them by name only works while their names cannot change. Today a
person can rename "FX clearing", which already breaks transfers in that
currency: the next one creates a new "FX clearing" and the clearing balance is
split across two accounts. The `RenameLedgerAccount` and `MoveLedgerAccount`
workflows therefore refuse to rename or move these accounts, and to rename a
top-level account to one of their names. The check is in the workflow, which
retrieves the account first, since every caller goes through it. A stored role
column on the ledger account would make the recognition independent of the
name, and is left as a follow-up.

### Closed accounts, names, and keys

- A closed account can be deleted or merged, which is how a person removes an
  account closed long ago. A replacement must be open.
- Deleting removes the rows, so names and external IDs are freed. The unique
  indexes only see rows that exist.
- Every opening-balance key `opening-balance:<ledger account id>` dies with its
  account. A recreated account has a new id, and so a new key.

### Running a deletion

A deletion is a workflow that chains activities, one per step, each in its own
transaction:

1. **Close the account without the usual checks.** Closing normally refuses an
   account with a balance or with units; here the account is going away, and
   closing stops new entries from being recorded on it.
2. **Handle the next batch of entries.** One activity takes the entries on the
   account, up to a batch size set by the workflow, and deletes each one or
   moves its lines, as above. The workflow calls it again until it handles none.
3. **Handle the entry-less movement groups**, the same way.
4. **Delete the account's rows.** If a foreign key still blocks the delete,
   because something was recorded on the account in the meantime, the workflow
   goes back to step 2.

Each entry is handled whole, as one complete financial write, so the ledger
balances after every step and surviving accounts never change, as CLAUDE.md
requires. Every step is idempotent: deleting an entry that is gone succeeds,
and a line already on the stray does not move again. Whatever a step handles no
longer touches the account, so the loop always ends.

The workflow's ID is derived from the account, such as
`delete-financial-account-<id>` or `delete-ledger-account-<id>`, so asking to
delete the same account twice joins the running deletion instead of starting a
second one. A deletion of an institution starts its accounts' deletions as
child workflows with the same IDs. A merge runs the same loop, moving lines to
the replacement instead.

A deletion that fails and is not run again leaves the account closed, partly
emptied, and possibly with a balance, a state that closing otherwise never
allows. Running it again, once the cause is fixed, picks up where it stopped.

Reversals, and deletions of entries under RFC 0003, are still allowed on a
closed account, so one can arrive while a deletion runs. It is not refused: a
reversal touches the same accounts as the entry it reverses, so the next pass
of step 2 handles it like any other entry, and step 4 sends the workflow back
if it arrives late. Refusing it instead, with a clear error, is possible later
(see Options considered).

### Seeing it before doing it

Deleting an account can delete a lot of history. A read-only
`PreviewAccountDeletion` returns what a delete or a merge would do: the entries
and movement groups deleted, those kept with lines moved, and the amount that
would land on each stray account. The UI shows it before asking for
confirmation. It describes the account as it is when asked, so an entry
recorded in between is handled without having been shown.

### Enforcement

Moving a line updates a posted row: a posting's `ledger_account_id`, or a
movement's `financial_account_id`. The triggers of RFC 0003 gain a second
exception, with a transaction-local setting naming the account being deleted
or merged, which lets those columns change, and nothing else, on that
account's rows only. A posting cannot move to an account of another currency,
so every entry still balances. Deleting the split and adjustment legs of a
deleted account uses the same setting.

Each batch locks the financial accounts it touches, the deleted one, the
replacement, and the stray, in ascending id order, and the ledger accounts, as
posting does.

### The commands

| Workflow | What it does |
| --- | --- |
| `DeleteLedgerAccount` | Exists. Runs the deletion above on an account with postings, with an optional replacement. Its final step is today's activity, which deletes an unused account. |
| `DeleteFinancialAccount` | New. Runs it on a financial account and its ledger accounts, with an optional replacement. |
| `DeleteInstitution` | Exists. Deletes each financial account as a child workflow, then the institution with today's activity, instead of refusing. |
| `AssignUntracked` | New. Moves the stray lines of one entry or movement group to a real account, in one activity. |
| `PreviewAccountDeletion` | New, read-only. |

The deletion workflows share three new activities: closing without the
checks, handling a batch of entries, and handling a batch of movement groups.
Deleting an account that does not exist succeeds. `DeleteOrCloseLedgerAccount`
and `DeleteOrCloseCategory` no longer have a purpose and are removed.

## Examples

Deleting "Revolut Personal", which has an opening balance, an expense, and a
transfer from the bank:

```text
Opening balance   Dr Revolut Cash 1000 / Cr Opening balances 1000   -> deleted
Groceries         Dr Groceries 50      / Cr Revolut Cash 50         -> deleted
From the bank     Dr Revolut Cash 500  / Cr Bank Cash 500           -> kept:
                  Dr Untracked accounts 500 / Cr Bank Cash 500
```

The bank's balance does not change. "Untracked accounts" (EUR) shows 500 that
left for an account no longer tracked. The name "Revolut Personal" is free.

Merging the duplicate "Revolut personal EUR" into "Revolut Personal":

```text
From the bank     Dr Duplicate Cash 200 / Cr Bank Cash 200      -> Dr Revolut Cash 200 / Cr Bank Cash 200
Coffee            Dr Coffee 3 / Cr Duplicate Cash 3             -> Dr Coffee 3 / Cr Revolut Cash 3
Between the two   Dr Duplicate Cash 100 / Cr Revolut Cash 100   -> deleted: both lines on Revolut Cash
```

Deleting the category "Coffee":

```text
Coffee            Dr Coffee 3 / Cr Revolut Cash 3   -> Dr Uncategorized 3 / Cr Revolut Cash 3
```

Deleting the broker "IBKR", where 10 ACME were bought and later moved to
"Degiro":

```text
Buy               Dr IBKR Investments / Cr IBKR Cash, IBKR +10 ACME   -> deleted
Transfer          IBKR -10 ACME, Degiro +10 ACME                      -> kept:
                  Untracked -10 ACME, Degiro +10 ACME
```

Degiro still holds 10 ACME. If IBKR is recreated, assigning the transfer's
stray leg to it puts the -10 back on IBKR, and the buy is recorded again.

## Options considered

- **Deleting every entry the account touches.** Simplest, and the closest to
  "the account never existed", but deleting Revolut would change the bank's
  balance through the transfers between them.
- **Refusing to delete while an entry touches a surviving balance.** Safe, but
  the accounts a person most often wants to delete are those that money moved
  through.
- **Keeping the rows and renaming them**, with a suffix or a `deleted_at`
  column, to free the name. Every row stays, so it is a soft delete, and every
  read has to filter it out.
- **One stray account per deleted account**, a tombstone. It would keep the
  deleted account's lines apart, and make getting them back one step, but it
  leaves a row behind for every deleted account: a soft delete again.
- **Recording on each moved line where it came from**, such as the deleted
  account's name in the entry's metadata. It would let a whole deleted account
  be assigned back at once. Left out of the first implementation, and can be
  added later without changing anything else: lines moved before it would
  simply have no source.
- **Turning a surviving transfer leg into a one-leg `adjustment`**, instead of
  a stray financial account for units. It needs no stray institution, but it
  changes a movement's kind, and the units cannot be assigned back.
- **Merge-only categories.** A used category could only be merged into
  another. It needs no "Uncategorized", but a category could not be removed
  without picking a target, and categories would work differently from
  accounts.
- **The whole deletion in one activity and one transaction.** A failure would
  leave nothing half done, and a few thousand entries fit comfortably in one
  transaction. A workflow was preferred for its retries and its progress, at
  the cost of a partly deleted account when it stops halfway.
- **Refusing reversals on an account being deleted.** It would tell the person
  why, instead of the reversal disappearing a moment later.
  - With the workflow IDs above, the reversal's activity could ask Temporal
    whether `delete-financial-account-<id>` is running, through
    `DescribeWorkflowExecution`. The answer is consistent, and it needs no
    setup and no particular server version. A possible improvement later.
  - With a search attribute on the deletion workflows holding the account's id,
    the activity could query visibility instead. Visibility is updated
    asynchronously, so the check would miss a deletion that has just started,
    and custom search attributes need registering and, on SQL visibility,
    Temporal server 1.20 or later. They are the right tool only for listing
    running deletions in a UI.
- **A role column for the shared accounts**, instead of recognising them by
  name. More robust, and left as a follow-up to the guard on renames.

## What this would change

If adopted, the decision would be written as an ADR that amends:

- ADR 0001: a posting's `ledger_account_id` can change when its account is
  deleted or merged.
- ADR 0003: a used category can be deleted, its postings moving to
  "Uncategorized", or merged into another; the hard delete rule for used
  accounts is replaced by this RFC's.
- ADR 0004: a movement's `financial_account_id` can change when its account is
  deleted or merged, and the legs of a split or an adjustment can be deleted
  with their account.
- ADR 0005: a name is reserved for as long as its row exists, closed or not,
  rather than forever.
- ADR 0006: "Opening balances" and "FX clearing" cannot be renamed or moved.
- CLAUDE.md, "Immutability of core financial attributes": hard delete is no
  longer reserved for unused rows.

## Open questions

- **Names.** "Untracked accounts", "Untracked", and "Uncategorized" become
  reserved names. Are they the right ones?
- **Recategorising.** Assigning lets an expense move from "Uncategorized" to a
  category. Should moving one from any category to another be allowed too, or
  stay a deletion and a new entry?
- **Merging an asset into a liability.** Is the same kind too strict, for
  example for a debit card set up as a `card` by mistake?
