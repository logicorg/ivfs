# ADR 0016: Deleting counterparties

- Status: Implemented
- Date: 2026-11-07
- Depends on: [ADR 0011](0011-deleting-journal-entries.md) (deleting journal
  entries), [ADR 0012](0012-deleting-accounts.md) (deleting accounts)
- Amends: ADR 0001 (posted entries are never updated), ADR 0002 (counterparties
  are archived), ADR 0005 (names are reserved forever), ADR 0011 (updates stay
  forbidden)
- Related: [ADR 0005](0005-uniqueness-and-idempotent-creation.md) (uniqueness
  keys), [ADR 0008](0008-intent-level-transactions.md) (the payee)

## Context

A counterparty created by mistake, or twice ("Tesco" and "TESCO"), can only be
renamed. Nothing removes it. ADR 0002 planned `ArchiveCounterparty`, which was
never built, and ADR 0011 and ADR 0012 have since chosen hard deletion for
entries and accounts.

A counterparty is descriptive metadata on a journal entry (ADR 0001). It is
never a posting target, and no balance, holding, or check reads it. Removing it
from an entry changes no posting, so every balance and every holding stays as
it is, and ADR 0012's rule, surviving balances never change, holds without
effort.

Two things stand in the way:

- `journal_entry.counterparty_id` refers to `ledger.counterparty` with
  `ON DELETE RESTRICT`, as every key does, so a referenced counterparty cannot
  be deleted.
- `ledger.prevent_settled_journal_entry_mutation` refuses any update of an
  entry that is not a draft, so the reference cannot be cleared from a posted
  entry either.

A counterparty is low-value data, and a short window in which it is half deleted
is acceptable. Bundling the steps into one transaction to rule out a concurrent 
post that will almost never happen is not worth it.

## Decision

### A posted entry's counterparty may change

The counterparty is the one column of a posted entry that may be updated.
`prevent_settled_journal_entry_mutation` lets an update through when no other
column changes, apart from `updated_at`:

```sql
IF TG_OP = 'UPDATE'
   AND to_jsonb(NEW) - 'counterparty_id' - 'updated_at'
     = to_jsonb(OLD) - 'counterparty_id' - 'updated_at' THEN
  RETURN NEW;
END IF;
```

Comparing the rows without those columns catches any other column, including
one added later, as the posting trigger does for a move (ADR 0012). The check
needs no transaction-local setting, unlike ADR 0011 and ADR 0012, so it works
from a single autocommit statement.

`validate_posted_journal_entry` runs only on `UPDATE OF status`, so it is not
involved. Postings and position movements stay frozen.

The trigger allows any change of the counterparty, not only clearing it. This
ADR uses it only to clear it, and it leaves room for setting or changing the
payee of a recorded entry (ADR 0008) without deleting and recording it again.

### The foreign key stays `RESTRICT`

Every key stays `ON DELETE RESTRICT`, with no `SET NULL`. References are
cleared first and the counterparty deleted second. The key is what makes the
order safe: a counterparty that is still referenced cannot be deleted, so no
entry is ever left pointing at a missing one.

### Two activities, one workflow

| Activity | Does | When nothing matches |
| --- | --- | --- |
| `ClearCounterpartyReferences` | `UPDATE ledger.journal_entry SET counterparty_id = NULL, updated_at = now() WHERE counterparty_id = ?` | succeeds |
| `DeleteCounterparty` | `DELETE FROM ledger.counterparty WHERE id = ?` | succeeds |

Each is one statement and is idempotent. `DeleteCounterparty` turns a foreign
key violation into an `ErrResourceConflict` with `sqllib.StillReferenced`, as
`DeleteAsset`, `DeleteCurrency`, and `DeleteInstitution` do.
`ErrResourceConflict` is already in the default `NonRetryableErrorTypes`, so
Temporal does not retry the delete alone, which would fail every time.

The `DeleteCounterparty` workflow runs the two in order. When the delete
returns `ErrResourceConflict`, an entry referring to the counterparty was
recorded between the two, and the workflow runs both again. It makes at most
two rounds and then returns the conflict. Any other failure follows the
default activity retries.

The two activities are separate operations, so neither wraps the other in a
transaction. A failure between them leaves the references cleared and the
counterparty in place. That is harmless.

### Reversals

A reversal copies `CounterpartyID` from the entry it reverses
(`reversal.go`). Both refer to the counterparty, so both are cleared, and the
pair stays consistent.

### Names are freed

Deleting removes the row, so its name leaves the `lower(name)` unique index and
can be used again, as ADR 0012 decided for accounts.

### Counterparty-specific ledger accounts are untouched

A receivable or payable account for a party, such as "Owed by Alice"
(ADR 0001), is an ordinary ledger account with no reference to the
counterparty. It survives, and is deleted on its own, as ADR 0012 describes.

### Idempotent replay

Retrying an earlier `PostJournalEntry` with the same idempotency key returns
the stored entry, which no longer has a counterparty. The retry returns what is
stored, as it always does, and does not set it again.

### The command

| Command | Files | Exposed as |
| --- | --- | --- |
| `DeleteCounterparty` | `finance/activities/ledger/accounts/counterparty_clear_references.go`, `finance/activities/ledger/accounts/counterparty_delete.go`, `finance/workflows/ledger/accounts/delete_counterparty.go` | `DELETE /api/counterparties/{id}` |

The result carries the number of entries cleared.

## Example

A person created "TESCO" by mistake and recorded 4 expenses with it. They
delete it: the 4 entries keep their postings and lose their counterparty, and
"TESCO" is gone. Balances do not move. The name "TESCO" can be created again.

If an expense naming "TESCO" is recorded between the two activities, the
delete fails on the foreign key, the workflow clears that entry too, and the
second delete succeeds.

## Alternatives considered

- **Archiving** (ADR 0002). It keeps the row, keeps the name taken, and needs
  every read and picker to filter archived rows. Entries and accounts are hard
  deleted (ADR 0011, ADR 0012), and a counterparty is the least valuable of the
  three.
- **One activity in one transaction.** It closes the window between the two
  steps, at the cost of a bundled operation for a race that is unlikely.
- **Deleting the counterparty first.** `RESTRICT` refuses it while any entry
  refers to it, and without the key nothing would stop dangling references.
- **`ON DELETE SET NULL`.** One statement clears and deletes atomically, but it
  breaks the "every key restricts deletion" rule that `pkg/sqllib/foreign_keys.go`
  and ADR 0011 rely on, and the cascaded update still needs the trigger
  exception.
- **A transaction-local setting** naming the counterparty being deleted, as ADR
  0011 and ADR 0012 do. It would tie the update to a transaction for no gain:
  the counterparty carries no money, so no update of it needs guarding.

## Consequences

- Counterparties can be deleted, and their names reused.
- A posted entry is no longer fully frozen: its counterparty may change. Every
  monetary column, its postings, and its movements stay immutable.
- A failure between the two activities leaves a counterparty that nothing
  refers to, until the workflow is run again.
- `prevent_settled_journal_entry_mutation` gains a case, and its tests in
  `pkg/sqllib/journal_entry_trigger_test.go` gain one for an update of the
  counterparty alone (allowed), one that also changes another column
  (refused), and one on a draft (allowed, as before).
- `ClearCounterpartyReferences` and `DeleteCounterparty` each get activity
  tests with an idempotency case, covering posted entries, drafts, and a
  reversal pair. The workflow test mocks both and covers a conflict followed by
  success, and the conflict returned after the last round.
- The doc comment of `ErrResourceClosed` mentions an archived counterparty,
  which no longer exists as a concept, and is updated.

## Amendments to earlier ADRs

ADR 0001:

- "Posted entries are never updated" becomes "Posted entries are never updated,
  except their counterparty (ADR 0016)".

ADR 0002:

- The Counterparties row of the command table becomes "Create, retrieve,
  list/search, rename, delete `Counterparty`".
- `CreateCounterparty` / `ArchiveCounterparty` becomes `CreateCounterparty` /
  `DeleteCounterparty`.
- "A posted entry, its postings, and its linked monetary position movements
  cannot be changed or deleted" gains "apart from the entry's counterparty
  (ADR 0016)". The `JournalEntry` doc comment, "A posted entry is immutable",
  gains the same exception.

ADR 0005:

- "Names are reserved forever" no longer covers counterparties: deleting a
  counterparty frees its name, as ADR 0012 already does for accounts.

ADR 0011:

- "Updates stay forbidden, and so does any delete made without the setting"
  becomes "Updates stay forbidden, apart from an update of the counterparty
  alone (ADR 0016), and so does any delete made without the setting".

`docs/onboarding/entity_relationships.md` and `docs/onboarding/ledger_examples.md`
describe the counterparty as optional metadata, which stays true. They gain a
line saying it can be deleted, and the operations table in
`ledger_examples.md` gains `DeleteCounterparty`.
