# ADR 0017: Editing journal entries

- Status: Proposed
- Date: 2026-11-08
- Depends on: [ADR 0011](0011-deleting-journal-entries.md) (deleting journal
  entries), [ADR 0016](0016-deleting-counterparties.md) (a posted entry's
  counterparty may change)
- Amends: ADR 0001 (posted entries are never updated), ADR 0008 (correcting a
  transaction), ADR 0011 (editing is a delete and a new record)
- Related: [ADR 0003](0003-categories-and-tags.md) (tags),
  [ADR 0004](0004-position-subledger.md) (position movements),
  [ADR 0005](0005-uniqueness-and-idempotent-creation.md) (idempotent creation)

## Context

ADR 0011 made editing a delete followed by a new record, in two requests, and
left an atomic replace "for when one proves needed". It has: the journal entry
sheet is where a person notices a wrong amount, a wrong card, or a wrong date,
and two requests mean two failure points, a new entry id, and a freed
idempotency key that the second request must not collide with.

The edits a person makes are of two kinds:

- **Metadata**: the date, the description, the counterparty, and, once
  assignment exists, the tags (ADR 0003). No balance moves.
- **Money**: the accounts, the amounts, the sides. Balances move, and the
  entry must still balance in each currency.

An entry may have been reversed. A reversal is a separate entry whose postings
mirror the original's, and which copies its counterparty and metadata
(`reversal.go`). An edit of the original that leaves the reversal alone breaks
the pair: the two no longer net to zero, and ADR 0011's rule that they are
deleted together would remove a non-zero set of postings.

The system has one user per database (ADR 0013) and practically no concurrent
writes to the same entry. No audit trail is required: ADR 0011 already chose
not to keep one for deletions.

Three shapes were weighed:

1. Delete and record again, atomically. Simple, but the entry gets a new id,
   which breaks the link to its reversal, any URL, and anything that refers to
   the entry later (reconciliation, imports).
2. One update command that replaces the entry's state in one transaction.
3. A thin activity per kind of change (postings, accounts, tags, ...), called
   one after the other by the workflow. A failure between two leaves an entry
   whose reversal no longer mirrors it, and every step needs its own
   idempotency and compensation.

## Decision

### One update, in one transaction

`UpdateJournalEntry` takes an entry id and the entry's full desired state, and
replaces it in one transaction:

| Field | Editable |
| --- | --- |
| `occurredAt`, `description`, `counterpartyId`, `metadata` | yes, replaced |
| `postings` | yes, replaced as a set |
| `id`, `idempotencyKey`, `externalId`, `status`, `postedAt`, `createdAt`, `reversalOfEntryId` | no |

The entry row is kept and updated, so its id, its idempotency key, and its link
to a reversal survive. Its children are replaced wholesale: every posting is
deleted and the new set inserted, with fresh line numbers. Nothing refers to a
posting by id, so no reference is lost. Diffing old and new postings would add
code for no observable difference.

Metadata edits and money edits go through the same command. A form edits the
whole entry, and a separate metadata-only path would be a second way to do a
subset of the same thing.

The update is idempotent without a key: running it twice with the same input
leaves the same state. Retries are harmless.

### Reversals follow the entry

When the entry has been reversed, the same transaction rewrites its reversal:
its postings become the mirror of the new ones (`Side.Opposite()`), and its
counterparty and metadata are copied again, as `ReverseJournalEntry` does when
it creates one. Its `occurredAt` and description stay as they were: they
describe when and why the reversal was recorded, which the edit does not change.
A reversal can be reversed in turn, so the activity walks the whole chain, each
link mirroring the one before it.

A reversal cannot be edited directly. It is refused with
`ErrInvalidParameter`, whose message says to edit the entry it reverses.

### Entries with position movements are refused

A trade or a vest moves cash and units together (ADR 0004). Replacing its
postings without its movements would leave the two disagreeing, and editing
movements belongs to an intent-level command that knows what a trade is. An
entry with position movements is refused with `ErrInvalidParameter`, and is
corrected by deleting and recording it again (ADR 0011) until such a command
exists.

### No optimistic locking

There is no version column and no `WHERE version = ?`. The last write wins. The
only realistic conflict is one person editing the same entry from two devices,
and the cost of losing one of those edits is retyping it. A version column can
be added later without changing the command's shape.

The activity still locks the ledger accounts it touches, old and new, in
ascending id order, as posting and deletion do, so a concurrent close or
balance read waits for it.

### Closed accounts

As with deletion (ADR 0011), an edit is allowed on an entry that touches a
closed account: closing is final, and a mistake still needs correcting. A
posting may not be moved onto a closed account the entry did not already
touch, which is refused with `ErrResourceClosed`, as posting is.

### Tags

Tag assignment does not exist yet (ADR 0003). When the `journal_entry_tag`
table is built, the desired state gains `tagIds`, and the update replaces the
entry's assignments in the same transaction, as it does postings. Assignments
carry no money and are not frozen by any trigger.

### Enforcement

The triggers stay, and gain an exception keyed by a transaction-local setting,
as ADR 0011 does for deletion:

- `ivfs.editing_journal_entry` names the entry being edited. It is set with
  `set_config(name, value, true)` for each entry of the chain before its rows
  are touched, and ends with the transaction.
- `prevent_settled_journal_entry_mutation` lets an update through when the
  setting names the row and none of the non-editable columns above change,
  compared as whole rows without the editable ones, so a column added later is
  frozen by default.
- `prevent_posted_posting_mutation` lets an insert or a delete through when the
  setting names the posting's entry. Updates of postings stay forbidden: they
  are replaced, not changed.

The balance check, `validate_posted_journal_entry`, is a deferred constraint
trigger that today fires on `INSERT OR UPDATE OF status`. It becomes
`INSERT OR UPDATE`, so it runs at commit for every update of an entry, and the
edit always updates the entry row (at least `updated_at`). An edit that leaves
the entry or its reversal unbalanced fails the commit. The activity also checks
the balance before writing, so the person gets an `ErrInvalidParameter` naming
the currency instead of a trigger's exception.

### Intent-level updates

A person edits an expense, an income, or a transfer, not postings. Each facade
of ADR 0008 gains an update next to its record:

| Workflow | Input |
| --- | --- |
| `UpdateExpense` | `journalEntryId` and the fields of `RecordExpense` but the key |
| `UpdateIncome` | `journalEntryId` and the fields of `RecordIncome` but the key |
| `UpdateTransfer` | `journalEntryId` and the fields of `RecordTransfer` but the key |

Each retrieves the entry, derives its type from its postings as ADR 0008
describes, and refuses an entry of another type with `ErrInvalidParameter`:
turning an expense into a transfer is deleting and recording it again. It then
runs the same checks as its record, builds the postings with the same code,
and calls `UpdateJournalEntry` as a child workflow. A transfer may switch
between one currency and two: the postings and metadata are built as
`RegisterFXConversion` builds them, so the clearing postings and
`effective_rate` appear or disappear with it.

### The commands

| Command | Files | Exposed as |
| --- | --- | --- |
| `UpdateJournalEntry` | `finance/activities/ledger/journal/update.go`, `finance/workflows/ledger/journal/update.go` | none |
| `UpdateExpense`, `UpdateIncome`, `UpdateTransfer` | `finance/workflows/facade/transactions/update_expense.go`, `update_income.go`, `update_transfer.go` | `PUT /api/transactions/expenses/{id}`, `/incomes/{id}`, `/transfers/{id}` |

There is no endpoint that replaces arbitrary postings, for the same reason
ADR 0008 has none that posts them. The journal entry sheet gains an edit action
next to reverse and delete, which opens the transaction form filled in.

## Example

A person recorded groceries of 125 from their card, entry 41, and reversed it
by mistake, entry 45. They edit entry 41 to 12.50 from their bank account:

```text
UpdateExpense{journalEntryId: 41, balanceLedgerAccountId: <Cash (EUR)>,
              categoryLedgerAccountId: <Groceries (EUR)>, amount: 12.50, ...}
  -> entry 41: debit Groceries 12.50, credit Cash 12.50
     entry 45: credit Groceries 12.50, debit Cash 12.50
```

Both keep their ids. Card balance is back to what it was before entry 41, and
the pair still nets to zero.

Editing a buy of 10 ACME is refused, and the message says to delete and record
it again.

## Alternatives considered

- **Delete and record again, in one transaction.** Atomic, but the entry gets
  a new id, which breaks the reversal link and every later reference, and a
  freed idempotency key is taken again under a new id.
- **A thin activity per kind of change**, chained by the workflow. Each is
  small, but a failure between two leaves the entry and its reversal
  disagreeing, and undoing earlier steps needs compensations. The pieces can
  be small functions inside the one activity instead.
- **Append-only corrections**, a reversal and a new entry for each edit. The
  accounting answer, and the one ADR 0011 already rejected for a personal
  ledger: the list fills with noise nobody asked for.
- **Optimistic locking with a version column.** Guards against two devices
  editing the same entry, which is rare enough here that losing one edit is
  acceptable.
- **Diffing postings** instead of replacing them. Keeps posting ids, which
  nothing uses.
- **Editing entries with position movements**, by replacing their movements
  too. Possible with the same pattern, but a trade's postings and movements are
  derived from one intent, and an edit of them belongs to that intent's
  command.
- **Updating a reversal on its own.** It would break the pair. The reversal
  follows its original.

## Consequences

- An entry can be corrected in place, keeping its id, its key, and its
  reversal, in one request that either fully applies or does not.
- A posted entry is no longer frozen: its date, description, counterparty,
  metadata, and postings may change through `UpdateJournalEntry`. Its identity
  columns stay immutable.
- The ledger keeps no record of what an entry was before an edit.
- A balance report run twice may differ for a past date without a new entry
  appearing. That was already true of deletion.
- Trades and vests are still corrected by deletion.
- The two triggers gain an edit exception, and their tests gain a case for an
  edit with the setting, one naming another entry, one changing a frozen
  column, and one without the setting. The balance trigger's tests gain an
  unbalanced edit that fails at commit.
- `UpdateJournalEntry` gets activity tests covering a draft, a posted entry, a
  reversal chain, a refused reversal, a refused entry with movements, a closed
  account, and an idempotency case. Each facade update gets a workflow test
  with mocked children, including a refused change of type.

## Amendments to earlier ADRs

ADR 0001:

- "Posted entries are never updated, except their counterparty (ADR 0016)"
  becomes "Posted entries are updated only through `UpdateJournalEntry`
  (ADR 0017), and their counterparty (ADR 0016)".

ADR 0008:

- "Correcting a transaction is reversing its entry, as for any entry
  (ADR 0001), and recording it again" becomes "Correcting a transaction is
  updating it (ADR 0017), or reversing or deleting its entry".

ADR 0011:

- "Editing is a delete and a new record" holds only for entries with position
  movements. Every other entry is edited in place (ADR 0017).
- "Updates stay forbidden, apart from an update of the counterparty alone
  (ADR 0016)" gains "and an edit that declares it (ADR 0017)".
