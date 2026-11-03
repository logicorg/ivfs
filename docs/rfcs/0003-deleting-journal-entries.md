# RFC 0003: Deleting journal entries

- Status: Accepted as [ADR 0011](../adrs/0011-deleting-journal-entries.md)
- Date: 2026-10-06
- Depends on: [ADR 0010](../adrs/0010-negative-holdings.md) (holdings that go negative)
- Related: [ADR 0001](../adrs/0001-double-entry-ledger.md) (entry lifecycle),
  [ADR 0004](../adrs/0004-position-subledger.md) (append-only movements, holdings
  never negative), [ADR 0006](../adrs/0006-intent-level-account-setup.md)
  (opening balances), [ADR 0008](../adrs/0008-intent-level-transactions.md)
  (correcting a transaction), [RFC 0004](0004-deleting-accounts.md) (deleting
  accounts)

## Problem

A posted journal entry can never be changed or deleted (ADR 0001). The only
correction is a reversal: a second entry with every side inverted, which
references the first. That is the right tool for an accountant, and the wrong
one for a person who typed 125 instead of 12.50 a minute ago. Fixing the typo
leaves three entries behind, the wrong one, its reversal, and the right one, and
every list of transactions shows all three for good.

The same is true of everything a person records through the intent-level
facades: an expense, an income, or a transfer (ADR 0008), an opening balance
(ADR 0006), and a trade or vest with its position movements (ADR 0004). A
units-only movement, such as a transfer of shares between two accounts, has its
own reversal, `ReversePositionMovement`, with the same drawback.

What a person expects is to delete the operation and get the ledger back to
the state it would be in had the operation never been recorded.

## What is already possible

- A **draft** entry can be deleted: the journal entry trigger allows it, and
  its postings are not protected while it is a draft. No activity exposes it.
- A posted entry always has at least two postings, so there is no "empty" or
  "orphaned" posted entry to clean up.
- A posted entry, its postings, and its movements are protected by three
  triggers: `prevent_settled_journal_entry_mutation`,
  `prevent_posted_posting_mutation`, and `guard_position_movement`.

## Why deleting an entry is safe for money

Every posted entry balances in each currency, and every balance is a sum over
postings. Removing a whole entry therefore removes a set of postings that sums
to zero in each currency: the ledger still balances, and every account's
balance becomes what it would have been without the entry. Nothing is stored
that was derived from the entry, so nothing goes stale.

This only holds for a whole entry. A single posting cannot be deleted, because
the entry would no longer balance; the operation is the unit, and an entry is
the operation.

## Proposal

### Deletion is a hard delete

`DeleteJournalEntry` deletes an entry of any status, its postings, and its
position movements, in one transaction. No row is kept, no status is set, and
nothing is recorded about the deletion. A reversal remains available for when
the trail is wanted.

`DeletePositionMovement` does the same for an entry-less movement group (a
split, a transfer, or an adjustment, ADR 0004), identified by any of its
movements, and deletes every movement of the group.

Both are idempotent: deleting something that does not exist succeeds, as
deleting an institution already does, so a retry after a successful delete is
not an error.

### Reversal pairs

A reversal and the entry it reverses net to zero, in postings and in units.

- Deleting an entry that has been reversed deletes its reversal with it. The
  two together change nothing, so removing both is always safe, and leaving the
  reversal alone would make it reverse nothing.
- Deleting a reversal alone is allowed, and puts the original's effect back.

Entry-less movement groups and their reversals (`ReversalOfMovementID`) follow
the same rules.

### Positions

Deleting a trade or vest removes its units from the history, as of the date it
was recorded, so every later point of that holding changes.

ADR 0010 allows holdings going negative. Thus, deleting an entry with
movements is always allowed.

A movement that was sized from the holding before it, a split, is not
recomputed. Deleting a buy that came before a split leaves the split's delta
as it was entered. A backdated buy inserted before a split has the same effect
today, so deletion adds no new case.

### Closed accounts

Deleting an entry changes the balance of every account it touches, so an entry
on an account closed since would leave a closed account with a balance, which
closing refuses (`refuseStrandedBalance`). A reversal is already allowed on a
closed account for the same reason, since closing is final and a mistake still
needs correcting, and deletion follows it: it is allowed.

### Idempotency keys are freed

A deleted entry takes its idempotency key with it. Recording the operation
again with the same key records it anew.

- For an opening balance this is useful. Its key is
  `opening-balance:<ledger account id>` (ADR 0006), so deleting a wrong opening
  balance and opening the account again posts the right one, which a reversal
  cannot do: the key would still return the wrong entry.
- For a transaction recorded from a form, a request sent again after the entry
  was deleted records it again. The UI makes a key per opening of the form
  (ADR 0008), so this needs the same form to be submitted again after the
  deletion.

### Enforcement

The three triggers stay, and gain one exception: a row may be deleted when the
transaction has declared that it is deleting that entry or movement group. The
delete activity sets a transaction-local setting, such as
`SET LOCAL ivfs.deleting_journal_entry = '<id>'`, before deleting, and each
trigger lets a delete through when the setting names the row's entry or group.
Updates stay forbidden, and so does any delete made without the setting.

The activity locks the ledger accounts and financial accounts the entry
touches, as posting does, so a concurrent close or holdings check waits for the
deletion.

### The commands

| Command | Activity file | Exposed as |
| --- | --- | --- |
| `DeleteJournalEntry` | `finance/activities/ledger/journal/delete.go` | `DELETE /api/journal-entries/{id}` |
| `DeletePositionMovement` | `finance/activities/ledger/portfolio/delete_movement.go` | `DELETE /api/position-movements/{id}` |

Each has its workflow, and the journal entry sheet in the UI gains a delete
action next to reverse.

## Options considered

- **Voiding instead of deleting.** The `voided` status exists, and every read
  already counts posted entries only, so allowing posted to voided would hide an
  entry without a new table. Rejected: the rows stay, the idempotency key stays
  taken, and the ledger is not the one it would be without the entry. It is a
  soft delete.
- **An audit table of deletions.** The first draft of this RFC proposed one.
  Rejected: there is a single user and no notion of an actor, a reversal
  already serves whoever wants a trail, and a record of a deletion is a soft
  delete in another table.
- **Reversals only.** Today's rule. It keeps the ledger's history complete,
  but leaves a list of transactions that a person cannot clean.
- **Deleting a single posting.** The entry would no longer balance.
- **Allowing deletes on the tables outright**, dropping the delete half of the
  triggers. Simpler, but any code path could then delete a posted entry, where
  the setting limits it to the activities meant to.

## What this would change

If adopted, the decision would be written as an ADR that amends:

- ADR 0001: "Posted entries are never updated or deleted" becomes "never
  updated"; a posted entry can be deleted whole, with its postings.
- ADR 0004: movements stay append-only for updates, and can be deleted with
  their entry or their group.
- CLAUDE.md, "Immutability of core financial attributes": a correction is a
  reversing entry or the deletion of the whole entry, never an update of it.

## Open questions

- **Editing.** With deletion, editing an entry becomes deleting it and
  recording it again, in two requests. Is an atomic replace worth having, so a
  failure in between cannot lose the entry?
- **Splits.** Is leaving a split's delta untouched acceptable, or should
  deleting a movement that comes before a split on the same holding be
  refused?
- **The `voided` status.** Nothing uses it for posted entries, and deletion
  makes voiding a draft the same as deleting it. Should it be removed?
- **The setting's mechanism.** A custom transaction-local setting works on
  every supported PostgreSQL version. An alternative is a database function
  that is the only way to delete, which needs a separate database role to stop
  plain deletes, and the application has one role today.
