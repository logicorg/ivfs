# ADR 0011: Deleting journal entries

- Status: Implemented
- Date: 2026-10-07
- Depends on: [ADR 0010](0010-negative-holdings.md) (holdings that go negative)
- Amends: ADR 0001 (posted entries are never deleted), ADR 0004 (movements
  are append-only, and the `voided` status)
- Amended by: [ADR 0016](0016-deleting-counterparties.md) and
  [ADR 0017](0017-editing-journal-entries.md)
- Related: [RFC 0003](../rfcs/0003-deleting-journal-entries.md) (the
  discussion this ADR decides), [ADR 0006](0006-intent-level-account-setup.md)
  (opening balances), [ADR 0008](0008-intent-level-transactions.md)
  (correcting a transaction), [RFC 0004](../rfcs/0004-deleting-accounts.md)
  (deleting accounts)

## Context

A posted journal entry can never be changed or deleted (ADR 0001). The only
correction is a reversal: a second entry with every side inverted, which
references the first. That is the right tool for an accountant, and the wrong
one for a person who typed 125 instead of 12.50 a minute ago. Fixing the typo
leaves three entries behind, the wrong one, its reversal, and the right one, and
every list of transactions shows all three for good.

The same holds for everything recorded through the intent-level facades: an
expense, an income, or a transfer (ADR 0008), an opening balance (ADR 0006),
and a trade or vest with its position movements (ADR 0004). An entry-less
movement group, such as a transfer of shares between two accounts, has its own
reversal, `ReversePositionMovement`, with the same drawback.

What a person expects is to delete the operation and get the ledger back to the
state it would be in had the operation never been recorded.

Deleting a whole entry is safe for money. Every posted entry balances in each
currency, and every balance is a sum over postings, so removing an entry
removes a set of postings that sums to zero in each currency. The ledger still
balances, every account's balance becomes what it would have been without the
entry, and nothing stored was derived from the entry, so nothing goes stale.
This holds only for a whole entry: deleting one posting would leave the entry
unbalanced. The operation is the unit, and an entry is the operation.

Today a draft entry can already be deleted, but nothing exposes it. A posted
entry, its postings, and its movements are protected by three triggers,
`prevent_settled_journal_entry_mutation`, `prevent_posted_posting_mutation`,
and `guard_position_movement` (`pkg/sqllib/ledger_constraints.go`,
`pkg/sqllib/position_constraints.go`), and every foreign key is
`ON DELETE RESTRICT`.

## Decision

### Deletion is a hard delete

`DeleteJournalEntry` deletes an entry of any status, its postings, and its
position movements, in one transaction. No row is kept, no status is set, and
nothing is recorded about the deletion.

`DeletePositionMovement` does the same for an entry-less movement group (a
split, a transfer, or an adjustment, ADR 0004). It is identified by any of its
movements, and deletes every movement written under the group's idempotency
key. A movement
that belongs to a journal entry, a trade or a vest, is refused with
`ErrInvalidParameter`: it is deleted with its entry.

Both are idempotent. Deleting something that does not exist succeeds, as
deleting an institution already does, so a retry after a successful delete is
not an error.

### Positions

Deleting a trade, a vest, or a movement group removes its units from the
history as of the date they were recorded, so every later point of that holding
changes.

Holdings may go negative (ADR 0010), so deleting is never refused for what it
leaves held. Both deletions read the positions they touched with
`positions.NegativeHoldingsTx` after deleting, in the same transaction, and
return what they find as warnings, as the other writes of positions do.

A movement sized from the holding before it, a split, is not recomputed.
Deleting a buy that came before a split leaves the split's delta as it was
entered. A backdated buy inserted before a split has the same effect today, so
deletion adds no new case, and the negative holding warning reports what goes
below zero.

### Closed accounts

Deleting an entry changes the balance of every account it touches, so an entry
on an account closed since leaves a closed account with a balance, which
closing refuses (`refuseStrandedBalance`). Closing is final, and a mistake
still needs correcting, so deletion is allowed on a closed ledger account or
financial account.

### Idempotency keys are freed

A deleted entry or movement group takes its idempotency key with it. Recording
the operation again with the same key records it anew.

- For an opening balance this is the point. Its key is
  `opening-balance:<ledger account id>` (ADR 0006), so deleting a wrong opening
  balance and opening the account again posts the right one, which a reversal
  cannot do: the key would still return the wrong entry.
- For a transaction recorded from a form, the same request sent again after
  the entry was deleted records it again. The UI makes a key each time the form
  is opened (ADR 0008), so this needs the same form to be submitted again after
  the deletion.

### Editing is a delete and a new record

Editing an entry is deleting it and recording it again, in two requests. A
failure between the two leaves the entry deleted, and the person records it
again. An atomic replace would need its own command per intent (an expense, a
trade, an opening balance, ...), and is left for when one proves needed.

### Enforcement

The three triggers stay, and gain one exception: a row may be deleted when the
transaction has declared that it is deleting that entry or movement group. The
delete activity sets a transaction-local setting before deleting:

- `ivfs.deleting_journal_entry` names the entry being deleted.
- `ivfs.deleting_position_movement_group` names the idempotency key of the
  entry-less group being deleted.

It is set with `set_config(name, value, true)`, the parameterisable form of
`SET LOCAL`, and ends with the transaction, so it never outlives the delete on
a pooled connection. Each trigger lets a delete through when the setting names
the row: the entry's id on `ledger.journal_entry`, the posting's entry on
`ledger.posting`, and on `asset.position_movement` the movement's entry, or its
idempotency key when it has no entry. Updates stay forbidden, apart from an
update of the counterparty alone (ADR 0016), and so does any delete made
without the setting.

The foreign keys stay `ON DELETE RESTRICT`, with no cascade. The activity
deletes in order, the entry's movements, its postings, and then the entry, so
a delete made outside it still fails on a foreign key.

The activity locks the ledger accounts and financial accounts the entries
touch, in ascending id order, as posting does, so a concurrent close, posting,
or holdings read waits for the deletion.

### The `voided` status is removed

Nothing writes `voided`: no activity voids an entry, and every read already
counts posted entries only. With deletion, voiding a draft is the same as
deleting it. The status is removed from `JournalEntryStatus`, the
`journal_entry_status_check` constraint, and the list filter, leaving `draft`
and `posted`. The system is pre-alpha, so nothing is migrated.

### The commands

| Command | Activity file | Exposed as |
| --- | --- | --- |
| `DeleteJournalEntry` | `finance/activities/ledger/journal/delete.go` | `DELETE /api/journal-entries/{id}` |
| `DeletePositionMovement` | `finance/activities/ledger/portfolio/delete_movement.go` | `DELETE /api/position-movements/{id}` |

Each has its workflow, in `finance/workflows/ledger/journal/delete.go` and
`finance/workflows/ledger/portfolio/delete_movement.go`, and the journal entry
sheet in the UI gains a delete action. The result carries the
negative holding warnings and nothing else.

## Example

A person records a grocery expense of 125 that should have been 12.50. They
delete it: the entry and its two postings are gone, the cash and groceries
balances are what they were before, and the list shows nothing in its place.
They record the expense again with the right amount.

A person who opened a brokerage account with an opening balance of 1,000 that
should have been 10,000 deletes the opening balance entry and opens the account
again. The key `opening-balance:<ledger account id>` is free, so the right
balance is posted.

A person deletes a buy of 10 ACME whose units were sold since. The buy, its cash
postings, and its movement are deleted, and the result warns that ACME in IBKR
is -10 units from the sale's date.

## Alternatives considered

- **Voiding instead of deleting.** Allowing posted to voided would hide an
  entry without a new table, since every read counts posted entries only. The
  rows stay, the idempotency key stays taken, and the ledger is not the one it
  would be without the entry. It is a soft delete.
- **An audit table of deletions.** There is a single user and no notion of an
  actor, a reversal already serves whoever wants a trail, and a record of a
  deletion is a soft delete in another table.
- **Reversals only.** Today's rule. It keeps the ledger's history complete, but
  leaves a list of transactions that a person cannot clean.
- **Deleting a single posting.** The entry would no longer balance.
- **Allowing deletes on the tables outright**, dropping the delete half of the
  triggers. Simpler, but any code path could then delete a posted entry, where
  the setting limits it to the activities meant to.
- **A deleting database function as the only way to delete.** It needs a
  separate database role to stop plain deletes, and the application has one
  role.
- **An atomic replace** alongside deletion. It keeps a failure between the
  delete and the new record from losing the entry, but needs a command per
  intent, and the two requests are enough for a person fixing their own entry.
- **Refusing to delete a movement before a split** on the same holding. It
  keeps a split's delta consistent with the holding it was sized from, but a
  backdated buy already breaks that, and the warning reports the result.
- **Keeping `voided`.** Nothing uses it, and deletion covers its one meaning.

## Consequences

- A mistake can be removed without leaving a trail in the list of transactions.
- The ledger keeps no record that an entry existed.
- A wrong opening balance can be replaced, which a reversal could not do.
- A closed account can be left with a balance by a deletion.
- A split's delta can disagree with the holding it was sized from after a
  deletion, as it already can after a backdated buy.
- A request sent again after a deletion records the operation again. The UI's
  key per opening of a form keeps this to a deliberate resubmission.
- The three triggers gain a delete exception, and their tests gain a case for a
  delete with the setting, one with a setting naming another row, and one
  without it.
- `voided` leaves the model, the constraint, and the tests that use it to
  stand for a non-effective entry, which use a draft instead.
- `docs/handover.md` describes the rule as built and is updated when this is
  implemented.

## Amendments to earlier ADRs

ADR 0001:

- "Posted entries are never updated or deleted" becomes "never updated". A
  posted entry can be deleted whole, with its postings and its reversals.
- `status` is `draft` or `posted`.

ADR 0004:

- Movements stay append-only for updates, and can be deleted with their entry
  or their group, by the activities that declare it.
- "Only draft to posted and draft to voided are allowed, and only a draft can
  be deleted" becomes "only draft to posted is allowed", and a posted entry can
  be deleted as above.
- "A used row can never be deleted" still holds for every row but an entry, its
  postings, and its movements, which are deleted together.
- "Drafts and voided entries never contribute" becomes "drafts never
  contribute".

CLAUDE.md, "Immutability of core financial attributes": a posted entry and its
postings are never updated. A correction is a reversing entry or the deletion of
the whole entry, never an update of it.
