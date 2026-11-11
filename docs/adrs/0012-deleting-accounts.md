# ADR 0012: Deleting accounts

- Status: Implemented
- Date: 2026-10-07
- Depends on: [ADR 0011](0011-deleting-journal-entries.md) (deleting journal
  entries), [ADR 0010](0010-negative-holdings.md) (holdings that go negative)
- Amends: ADR 0001 (a posting's account never changes), ADR 0003 (used
  accounts are closed, never deleted), ADR 0004 (movements are append-only),
  ADR 0005 (names are reserved forever), ADR 0006 (the shared accounts)
- Amended by: [ADR 0017](0017-editing-journal-entries.md)
- Related: [RFC 0004](../rfcs/0004-deleting-accounts.md) (the discussion this
  ADR decides), [ADR 0005](0005-uniqueness-and-idempotent-creation.md)
  (uniqueness keys), [ADR 0006](0006-intent-level-account-setup.md) (account
  structure and shared accounts)

## Context

A ledger account, a financial account, or an institution that has been used
can only be closed. Closing keeps every row, and the name stays reserved
forever (ADR 0005), so an account opened by mistake, or twice, or with a typo
in its name, stays in every list and blocks its own name for good.

What a person expects is one of two things:

- **Delete**: the account never existed. Its history goes with it.
- **Merge**: the account was a duplicate of another one. Its history becomes
  the other account's.

ADR 0011 made a whole journal entry deletable. Deleting an account is harder,
because not every entry on an account belongs to it alone. A transfer from a
bank account to a Revolut account is half the bank's history: deleting the
transfer with Revolut would change the bank's balance, which is still true.

Today `DeleteLedgerAccount` deletes a ledger account with no postings and no
children, `DeleteOrCloseLedgerAccount` and `DeleteOrCloseCategory` delete such
an account and close any other, and `Institution.Delete` deletes an institution
no financial account refers to. There is no delete for a financial account.
Every foreign key is `ON DELETE RESTRICT`.

## Decision

### The rule: surviving balances never change

Deleting or merging accounts never changes the balance, or the holdings, of an
account that survives. Everything below follows from it.

A **balance** is an asset or a liability ledger account, such as a "Cash" or a
"Card balance", apart from the shared "FX clearing" accounts (ADR 0006).
Categories, "Opening balances", "FX clearing", other equity accounts, and the
stray accounts below are not balances: they explain where money came from or
went, and may change when the history they explain is deleted.

### Deleting

Deleting an account looks at every journal entry and every entry-less movement
group that touches it, and does one of two things with each:

| The entry or movement group also touches... | Then it is... |
| --- | --- |
| only the account being deleted, categories, equity accounts, "FX clearing", or the stray accounts | deleted, as ADR 0011 does |
| a surviving balance, or units on a surviving financial account other than the stray one | kept, and its lines on the deleted account move to the stray accounts |

An expense paid from the account, its opening balance, and a currency
conversion inside it are deleted. A transfer to or from an account that
survives is kept, so the surviving side stays as it is.

Draft entries that touch a deleted account are deleted, since they have no
effect.

Since the stray accounts are not balances, the order of two deletions does not
change the result. A transfer between A and B is kept when A is deleted, with
A's side on the stray, and deleted when B is deleted afterwards, since it then
touches only B and the stray. Deleting several accounts is deleting each in
turn.

A ledger account with children cannot be deleted or merged until its children
are, and fails with `ErrResourceConflict`. A financial account's own ledger
accounts are handled with it, children first.

### The stray accounts

Lines moved off a deleted account land on a stray account. The batch activity
creates each one by retrieve-or-insert, inside its own transaction, the first
time it needs it:

| Stray | Kind | One per | Receives |
| --- | --- | --- | --- |
| "Untracked accounts" | equity ledger account, top-level | currency | postings of a deleted balance or equity account |
| "Uncategorized" | expense or income ledger account, top-level | kind and currency | postings of a deleted category |
| "Untracked accounts" | financial account, under an "Untracked" institution | | units of a deleted financial account |

"Untracked accounts" says how much moved between tracked accounts and accounts
that are no longer tracked. It is equity, like "Opening balances", and is not
expected to be zero. Its holdings can be negative: a transfer of 10 shares from
a deleted account to a surviving one leaves -10 on it (ADR 0010).

On a kept entry or group, a transfer leg, or a trade's or vest's movement,
moves to the stray financial account. A split or an adjustment leg on a deleted
account is deleted rather than moved, even when other legs of its group
survive: each leg is a fact about its own account only, and a group of either
kind stays valid with fewer legs.

A moved posting or movement does not record which account it came from.

### Merging

Deleting with a **replacement** merges the account into it. Every posting and
movement of the deleted account moves to the replacement, including those that
deleting alone would have deleted, and split and adjustment legs, so no history
is lost.

- A ledger account is replaced by an open ledger account of the same kind and
  currency.
- A financial account is replaced by an open financial account. Each of its
  ledger accounts is merged into the replacement's ledger account with the same
  name and currency, which is created there by retrieve-or-insert when missing.
  If the replacement's account of that name has another kind, the merge fails
  with `ErrResourceConflict` before it changes anything. Its movements move to
  the replacement.
- A replacement cannot be a shared or a stray account, and fails with
  `ErrInvalidParameter`. Merging into the stray is what deleting already does.
- An entry or a movement group whose every line ends up on one account is
  deleted, since it changes nothing: a transfer between a duplicate and the
  account it duplicates is one.

An asset is not merged into a liability, nor the other way round. Kind is
immutable after use (ADR 0003), and a merge across kinds would change it for
every posting it moves.

### Assigning stray lines

A line on a stray account belongs to an ordinary journal entry or movement
group, so it can be got back or got rid of:

- **Assigning** moves the stray lines of one entry, or one movement group, to
  real accounts: each stray ledger account's lines to an open ledger account in
  the same currency, a balance for "Untracked accounts" and a category of the
  same kind for "Uncategorized", and its stray movements to an open financial
  account. It is a merge of one entry instead of one account, for example after
  recreating an account deleted by mistake, or to give an expense in
  "Uncategorized" a category.
- **Deleting** the entry or the group, with ADR 0011, removes the stray lines
  and the surviving side together, as if the transfer had never happened.

Assigning only moves lines off a stray account. Moving an expense from one real
category to another stays a deletion and a new record (ADR 0011). Since a moved
line does not record where it came from, the stray holds the lines of every
deleted account mixed together, and assigning is done one entry at a time,
picked by date, amount, and the other side.

### Institutions

Deleting an institution deletes each of its financial accounts in turn, then
the institution. Since the order of deletions does not matter, a transfer
between two accounts of the same bank ends up deleted, not moved. An
institution is not merged; its accounts can be merged one at a time first.

### Categories

Deleting a used category moves its postings, on kept entries, to
"Uncategorized" of its kind and currency, so an expense stays an expense and
the total spent does not change; only the breakdown loses the category. With a
replacement, a category of the same kind and currency, it is a merge instead.

### Shared and stray accounts

"Opening balances", "FX clearing", and the stray accounts cannot be deleted
while they have postings or movements, and fail with `ErrResourceConflict`.
They are recognised by name, kind, currency, and being top-level with no
financial account, as ADR 0006 creates them; the stray financial account by its
name and its institution's.

Recognising them by name only works while their names cannot change. Today a
person can rename "FX clearing", which already breaks transfers in that
currency: the next one creates a new "FX clearing" and the clearing balance is
split across two accounts. So:

- `RenameLedgerAccount` and `MoveLedgerAccount` refuse to rename or move a
  shared or stray ledger account, and to rename or move a top-level account
  onto one of their names.
- `CloseLedgerAccount` refuses to close them, since every later write that
  needs them would fail on the closed account.
- The institution and financial account workflows refuse to create, rename, or
  close the "Untracked" institution and its "Untracked accounts".

Each check is in the workflow, which retrieves the account first, since every
caller goes through it, and fails with `ErrInvalidParameter` naming the
account. The batch activity creates the strays directly in its transaction, so
the checks never see it. A stored role column would make the recognition
independent of the name, and is left as a follow-up.

### Closed accounts, names, and keys

- A closed account can be deleted or merged, which is how a person removes an
  account closed long ago. A replacement must be open.
- Deleting removes the rows, so names and external IDs are freed: the unique
  indexes only see rows that exist.
- Every opening-balance key `opening-balance:<ledger account id>` dies with its
  account. A recreated account has a new id, and so a new key.

### Running a deletion

A deletion is a workflow that chains activities, each step in its own
transaction:

1. **Close the account without the usual checks.** Closing normally refuses an
   account with a balance or with units; here the account is going away, and
   closing stops new entries from being recorded on it. A financial account's
   ledger accounts are closed with it.
2. **Handle the next batch of entries.** One activity takes the entries on the
   account, in id order, up to a batch size the workflow sets, 100 to start,
   and deletes each one or moves its lines, as above. The workflow calls it
   again until it handles none.
3. **Handle the entry-less movement groups**, the same way.
4. **Delete the account's rows**, with today's delete activities. If a foreign
   key still blocks the delete, because something was recorded on the account
   in the meantime, the workflow goes back to step 2.

Each entry is handled whole, as one complete financial write, so the ledger
balances after every step and surviving balances never change. Every step is
idempotent: deleting an entry that is gone succeeds, and a line already on the
stray or the replacement does not move again. Whatever a step handles no longer
touches the account, so the loop ends. A workflow whose history grows past a
few thousand events continues as new with the same input.

A merge runs the same loop, moving lines to the replacement. If the replacement
is closed while a merge runs, the batch fails with `ErrResourceClosed`, and the
merge can be run again with another replacement, or none.

A deletion that fails and is not run again leaves the account closed, partly
emptied, and possibly with a balance, a state closing otherwise never allows.
Running it again, once the cause is fixed, picks up where it stopped.

Deletions of entries under ADR 0011 are still allowed on a closed account, so
one can arrive while a deletion runs. It is not refused: it only removes an
entry the batches would have handled. Nothing new can be posted to a closed
account, since ADR 0017 removes reversals.

### One deletion per account

The workflow's ID is derived from the account: `delete-ledger-account-<id>`,
`delete-financial-account-<id>`, and `delete-institution-<id>`. Deletions are
started with `WorkflowIDConflictPolicy` `UseExisting`, so a second request for
the same account while one is running gets the running deletion and waits on
it instead of starting another. It gets that deletion's result, whatever
replacement it asked for. The default ID reuse policy lets a deletion that
stopped be run again under the same ID.

A deletion of an institution starts its accounts' deletions as child workflows
with the same IDs. Child workflows take no conflict policy, so a child that is
already running as a separate request fails to start, and the institution's workflow waits a minute and starts it again,
when it finds the account gone and succeeds at once.

The gateway's `WorkflowTimeout` is an execution timeout, which would cut a long
deletion in half. The deletion endpoints start their workflow without one, and
wait at most `WorkflowTimeout` for its result: a deletion that is still running
then is answered with `202 Accepted` and its workflow ID, and asking again
joins it.

### Seeing it before doing it

A read-only `PreviewAccountDeletion` takes a ledger account, a financial
account, or an institution, and an optional replacement, and returns what the
deletion or the merge would do: the entries and movement groups deleted, those
kept with lines moved, the amount that would land on each stray account, and
any conflict that would stop it. The UI shows it before asking for
confirmation. It describes the account as it is when asked, so an entry
recorded in between is handled without having been shown.

### Positions

A merge and an assignment return the negative holding warnings of the
financial account they moved units onto, read with
`positions.NegativeHoldingsTx` in the batch's transaction, as the other writes
of positions do (ADR 0010). A deletion changes no surviving holding, so it
returns none. The stray financial account's holdings are expected to be
negative and are never reported.

### Enforcement

ADR 0017 removes the triggers and the two settings this section describes.
A moved posting stays on an account of the same currency because the activity
picks the stray by currency, and the balance check on postings refuses a move
that would unbalance an entry. What follows is the design as first
implemented.

Moving a line updates a posted row: a posting's `ledger_account_id`, or a
movement's `financial_account_id`. The triggers of ADR 0011 gain a second
exception, with two transaction-local settings set by `set_config(name, value,
true)`:

- `ivfs.deleting_ledger_account` names the ledger account whose lines are
  moving. `prevent_posted_posting_mutation` lets an update through when it
  changes only `ledger_account_id`, the old account is the one named, and the
  new account has the same currency. The balance check of
  `validate_posted_journal_entry` runs only when an entry is posted, so the
  currency check in the trigger is what keeps every entry balanced.
- `ivfs.deleting_financial_account` names the financial account whose
  movements are moving. `guard_position_movement` lets an update through when it
  changes only `financial_account_id` and the old account is the one named, and
  lets the delete of an entry-less split or adjustment leg on that account
  through.

The activity sets the setting again for each account it moves lines off, a
financial account's ledger accounts in turn, and for an assignment the stray
it moves lines off. Any other update stays forbidden.

Each batch locks the financial accounts it touches, the deleted one, and the
replacement, in ascending id order, and then the ledger accounts, as posting
does. The strays are not locked: a batch retrieves or creates them after its
locks, the stray financial account first and then the stray ledger accounts by
currency and kind, so two batches never wait on each other's inserts. The
foreign keys of the moved lines keep a stray from being deleted under a batch,
and a stray that has lines cannot be emptied.

### Not exposed to the MCP server

Deleting an account can delete years of history, and an MCP client acts
without the preview a person sees. The commands are workflows and gateway
endpoints only, until a confirmation step exists for MCP.

### The commands

| Workflow | What it does | Exposed as |
| --- | --- | --- |
| `DeleteLedgerAccount` | Exists. Runs the deletion on any ledger account, with an optional replacement. Its last step is today's activity. | `DELETE /api/ledger-accounts/{id}` |
| `DeleteCategory` | New, in the categories facade. Runs `DeleteLedgerAccount` on an income or expense account as a child workflow. | `DELETE /api/categories/{id}` |
| `DeleteFinancialAccount` | New. Runs the deletion on a financial account and its ledger accounts, with an optional replacement. | `DELETE /api/financial-accounts/{id}` |
| `DeleteInstitution` | Exists. Deletes each financial account as a child workflow, then the institution with today's activity, instead of refusing. | `DELETE /api/institutions/{id}` |
| `AssignUntracked` | New. Moves the stray lines of one entry or movement group, in one activity. | `POST /api/journal-entries/{id}/assign`, `POST /api/position-movements/{id}/assign` |
| `PreviewAccountDeletion` | New, read-only. | `GET` on `/deletion-preview` under each of the four |

A replacement is passed as `replacementId`. The deletion workflows share three
new activities, in `finance/activities/ledger/accounts`: closing without the
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

The bank's balance does not change. "Untracked accounts" (EUR) shows the 500
that left for an account no longer tracked. The name "Revolut Personal" is
free.

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

## Alternatives considered

- **Deleting every entry the account touches.** Simplest, and the closest to
  "the account never existed", but deleting Revolut would change the bank's
  balance through the transfers between them.
- **Refusing to delete while an entry touches a surviving balance.** Safe, but
  the accounts a person most often wants to delete are those money moved
  through.
- **Keeping the rows and renaming them**, with a suffix or a `deleted_at`
  column, to free the name. Every row stays, so it is a soft delete, and every
  read has to filter it out.
- **One stray account per deleted account**, a tombstone. It keeps a deleted
  account's lines apart and makes getting them back one step, but leaves a row
  behind for every deleted account: a soft delete again.
- **Recording on each moved line where it came from**, such as the deleted
  account's name in the entry's metadata. It would let a whole deleted account
  be assigned back at once. Left out, and can be added later without changing
  anything else: lines moved before it simply have no source.
- **Turning a surviving transfer leg into a one-leg `adjustment`**, instead of
  a stray financial account. It needs no stray institution, but changes a
  movement's kind, and the units cannot be assigned back.
- **Merge-only categories.** A used category could only be merged into
  another. It needs no "Uncategorized", but a category could not be removed
  without picking a target, and categories would work differently from
  accounts.
- **The whole deletion in one activity and one transaction.** A failure would
  leave nothing half done, and a few thousand entries fit in one transaction.
  The workflow was preferred for its retries and its progress, at the cost of a
  partly deleted account when it stops halfway.
- **Refusing reversals on an account being deleted.** It would tell the person
  why, instead of the reversal disappearing a moment later. The reversal's
  activity could ask Temporal whether `delete-financial-account-<id>` is
  running, with `DescribeWorkflowExecution`, which is consistent and works on
  every server. A search attribute holding the account's id would need
  registering and, on SQL visibility, Temporal server 1.20 or later, and
  visibility is updated asynchronously, so it would miss a deletion that has
  just started. Neither is needed while the next pass handles the reversal.
- **"Already started" and `GetWorkflow`** to join a running deletion. It works
  on every Temporal server, but takes two calls and an error check where
  `UseExisting` takes one.
- **A role column for the shared accounts**, instead of recognising them by
  name. More robust, and left as a follow-up to the guards.
- **Merging across asset and liability**, for a debit card set up as a `card`.
  The postings would keep their sides, so the ledger would still balance, but
  every moved posting would change kind. The card is deleted and recorded again
  instead, until this proves common.
- **Recategorising any entry** by assigning it from one real category to
  another. Useful, but a separate decision from deleting accounts, and a
  deletion and a new record do it today.
- **Exposing deletion to the MCP server.** Left until MCP has a confirmation
  step to show the preview.

## Consequences

- An account opened by mistake or twice can be removed or merged, and its name
  and external ID reused.
- Surviving balances and holdings never change when an account is deleted. The
  difference lands on "Untracked accounts", which is not expected to be zero.
- The ledger keeps no record of which account a stray line came from, so
  getting a deleted account back is one assignment per entry.
- A failed deletion can leave a closed account with a balance until it is run
  again.
- Starting deletions with `UseExisting` needs Temporal server 1.24 or later.
- A posted posting's account and a movement's financial account can change,
  and a split or an adjustment leg can be deleted alone, only through the two
  new settings. The triggers' tests gain a case for each change with the
  setting, one with a setting naming another account, one changing another
  column, one moving a posting across currencies, and one without the setting.
- "Opening balances", "FX clearing", "Untracked accounts", "Uncategorized", and
  the "Untracked" institution can no longer be renamed, moved, or closed, which
  also fixes the split clearing balance a renamed "FX clearing" causes today.
- `DeleteOrCloseLedgerAccount` and `DeleteOrCloseCategory` are removed, with
  their tests and their registrations.
- `docs/handover.md` describes the rule as built and is updated when this is
  implemented.

## Amendments to earlier ADRs

ADR 0001:

- A posting's `ledger_account_id` can change when its account is deleted or
  merged, to an account of the same currency. Nothing else about a posted
  posting changes.

ADR 0003:

- A used ledger account or category can be deleted, its lines handled as
  above, or merged into another of the same kind and currency. "Used accounts
  are closed, never deleted" no longer holds.
- `DeleteOrCloseCategory` is replaced by `DeleteCategory`.

ADR 0004:

- A movement's `financial_account_id` can change when its account is deleted or
  merged, and a split or an adjustment leg can be deleted with its account.
- "A used row can never be deleted" no longer holds for ledger accounts,
  financial accounts, and institutions, which are deleted as above.

ADR 0005:

- "Names are reserved forever" becomes "a name is reserved for as long as its
  row exists, closed or not".

ADR 0006:

- "Opening balances" and "FX clearing" cannot be renamed, moved, or closed.

`docs/design_principles.md`, "Immutability of core financial attributes":
moving a line off a deleted or merged account is the one update of a posted
posting or movement, and changes only the account it is on.
