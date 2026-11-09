# ADR 0017: Editing journal entries and removing reversals

- Status: Proposed
- Date: 2026-11-09
- Depends on: [ADR 0004](0004-position-subledger.md) (position movements),
  [ADR 0008](0008-intent-level-transactions.md) (transactions),
  [ADR 0009](0009-intent-level-investments.md) (trades and vests), and
  [ADR 0011](0011-deleting-journal-entries.md) (deleting journal entries)
- Amends: ADR 0001, ADR 0002, ADR 0004, ADR 0006, ADR 0007, ADR 0008, ADR 0009,
  ADR 0010, ADR 0011, ADR 0016
- Related: [ADR 0005](0005-uniqueness-and-idempotent-creation.md) (idempotent
  creation), [ADR 0012](0012-deleting-accounts.md) (deleting accounts)

## Context

ADR 0011 made editing a delete followed by a new record, in two requests, and
left an atomic replace "for when one proves needed". Two requests mean a
failure between them loses the entry, and the entry comes back with a new id.

Reversals are the other way to correct an entry, and ADR 0011 kept them for
whoever wants a trail. In a single-person ledger with deletion and editing they
earn little, and they cost a lot:

- deleting an entry or a movement group has to follow the chain of its
  reversals, read it twice under locks, and delete it last to first;
- average cost has to skip a movement and its reversal when both are present;
- posting has an exception for closed accounts;
- an edit would have to rewrite the chain in the same transaction, so that the
  pair still nets to zero.

`ReversePositionMovement` has no route or MCP tool at all.

A refund does not need a reversal. It is an income on a category such as
"Refund", on its own date and for any amount.

## Decision

### Scope

This ADR changes Temporal workflows and activities only, and removes reversals
everywhere. It adds nothing to the HTTP gateway, MCP, or the UI.
The gateway, MCP, and the UI change only to remove reversals, apart from the
movements cursor in [Ordering movements](#ordering-movements). The workflows 
below are registered, and a later decision exposes them.

### Reversals are removed

These go:

- `ReverseJournalEntry` and `ReversePositionMovement`, as workflows and
  activities;
- `ledger.journal_entry.reversal_of_entry_id`,
  `asset.position_movement.reversal_of_movement_id`, their foreign keys, and
  `position_movement_reversal_unique`;
- `ReversalOfEntryID` on `PostJournalEntryInput`, and the closed-account
  exception it grants in `PostEntryWithMovementsTx`;
- the chains in `DeleteJournalEntry`, `DeletePositionMovement`, and the account
  deletion activities, which then delete one entry or one group;
- the skip of a movement and its reversal in `cost_basis.go`, and the
  reversal check in `RegisterPositionMovement`;
- `POST /api/journal-entries/{id}/reverse`, its MCP tool, the reverse dialog of
  the journal entry sheet, the reversal fields of the API's responses, and the
  reversal markers of the UI's statements and movements.

A wrong entry is edited or deleted. A wrong entry-less movement group is deleted
with `DeletePositionMovement`.

The system is pre-alpha, and databases are recreated rather than migrated
(ADR 0005).

### Replacing an entry

A new `ReplaceJournalEntry` workflow runs one activity of the same name. It
deletes the entry and writes it again **with the same id**, in one transaction.

| Field | On replace |
| --- | --- |
| `occurredAt`, `description`, `metadata`, postings, position movements | replaced by the input |
| `id`, `idempotencyKey`, `externalId`, `counterpartyId`, `createdAt`, `postedAt` | kept from the old row |

The input is the entry's id, the replaced fields, and zero or more movements,
each with its kind, financial account, asset, units, and cost basis. A movement
takes the entry's kept `externalId`, as `RegisterInvestmentTrade` and
`RegisterVest` copy it, so its units still trace to the same source record as
the cash.

The activity:

- reads the entry, and fails with `ErrNotFound` when it does not exist, or
  `ErrInvalidParameter` when it is not posted, since writing it again posts it;
- locks the financial and ledger accounts of the old entry and of the input
  once, in ascending id order, then reads the entry again;
- checks closed accounts, as below;
- deletes the entry with `deletion.DeleteEntryTx`;
- writes it again through the code that `PostEntryWithMovementsTx` uses, given
  the kept fields;
- returns the ids of the new movements, and `positions.NegativeHoldingsTx` for
  the positions of both the old and the new movements as warnings.

Every step reuses what deletion and posting already do. The triggers do not
change: `ivfs.deleting_journal_entry` already allows the delete, and the entry
is written as a draft and then posted, so `guard_position_movement` accepts its
movements and `validate_posted_journal_entry` runs at commit as it does for any
new entry. No row of a posted entry is updated.

It works because only `ledger.posting` and `asset.position_movement` refer to a
journal entry once reversals are gone, and both are replaced. A foreign key
added later must be replaced with them or the delete fails.

The counterparty is kept because the transactions of ADR 0008 and ADR 0009 do
not carry one, so replacing it from them would clear it.

A retried activity writes the same state again, so no idempotency key is
needed. A movement gets a new id on every replace, which nothing refers to, and
keeps its place through `created_at`, as below.

A later record request sent with the entry's original key returns the entry as
edited, as any retrieve-or-insert returns what is there (ADR 0005).

### Ordering movements

The movements of a position are replayed by `occurred_at`, then `created_at`,
then `id`, no longer by `occurred_at`, then `id`.

The UI dates a trade at the start of its day, so the movements of one day tie.
With `id` as the tie-break, a replaced movement would sort after everything
recorded since it. Correcting a buy sold the same day would put the sale first,
report a negative holding that never happened, and leave the basis unknown.

A movement written by a replace takes the entry's kept `createdAt`, which is
the value it had before, since an entry and its movements are written in one
transaction. `id` then only breaks ties inside one transaction, so nothing
depends on a movement keeping it. For a movement never replaced the order does
not change: `id` and `created_at` rise together.

The order changes in the running sums of `holdings.go`, in `cost_basis.go`, in
`list_movements.go`, and in `position_movement_position_idx`.
`PositionMovementCursor` gains `createdAt`, in the activity and where
`web/api/holdings` parses it, because a holding's history must list movements
in the order they are replayed.

### Closed accounts

An edit may keep a posting or a movement on a closed account that the entry
already touches: closing is final, and a mistake still needs correcting. An
account the entry did not already touch fails with `ErrResourceClosed`, for a
ledger account and for a financial account alike.

### Concurrency

There is no optimistic locking. The last write wins. The only realistic
conflict is one person editing the same entry from two devices.

### Holdings

Deleting the old movements and writing the new ones is never refused for what
it leaves held (ADR 0010). A vest needs the check on replace, unlike
`RegisterVest`: an edit can lower its units below what has since been sold.

Average cost and gains are worked out when read (ADR 0009), so they follow an
edited trade with no further work.

### Intent-level updates

Each record workflow gains an update, which runs the same checks and the same
builder, then calls `ReplaceJournalEntry` as a child workflow.

| Workflow | Input | Package |
| --- | --- | --- |
| `UpdateExpense` | `journalEntryId` and the fields of `RecordExpense` but the key | `facade/transactions` |
| `UpdateIncome` | `journalEntryId` and the fields of `RecordIncome` but the key | `facade/transactions` |
| `UpdateTransfer` | `journalEntryId` and the fields of `RecordTransfer` but the key | `facade/transactions` |
| `UpdateInvestment` | `journalEntryId` and the fields of `RecordInvestment` but the key | `facade/investments` |

`UpdateInvestment` has the switch on kind that `RecordInvestment` has, over
`buildBuy`, `buildSell`, and `buildVest`.

`UpdateExpense`, `UpdateIncome`, and `UpdateTransfer` return the entry's id and
the warnings of the replace, which `RecordResult` has no field for and needs
none, since recording them never warns. `UpdateInvestment` returns `TradeResult`,
which has them.

A transfer between currencies takes its postings and metadata from
`RegisterFXConversionInput.ToJournalEntry`, so a transfer can switch between one
currency and two, and `effective_rate` is worked out again from the new amounts.
A transfer has no fee or provider reference, so `UpdateTransfer` on a conversion
recorded with them drops the fee's posting and clears `provider_reference`.

**An update states what the entry is now, whatever it was before.** The type of
a transaction is not stored (ADR 0008), so `UpdateExpense` on an income makes it
an expense, and `UpdateInvestment` on an expense makes it a trade, adding the
movement. `UpdateExpense` on a trade removes the movement, and the warnings
report any holding left negative.

A record request retried with the original key after such a change can fail
with the `ErrResourceConflict` that `RegisterInvestmentTrade`, `RegisterVest`,
and `RegisterFXConversion` return for a key on an entry of another type. That
needs a change of type between a request and its retry.

### Tags

When `journal_entry_tag` exists, the replace input gains the entry's tags, and
the activity writes them again with the entry.

### The commands

| Command | Files |
| --- | --- |
| `ReplaceJournalEntry` | `finance/activities/ledger/journal/replace.go`, `finance/workflows/ledger/journal/replace.go` |
| `UpdateExpense`, `UpdateIncome`, `UpdateTransfer` | `finance/workflows/facade/transactions/update_expense.go`, `update_income.go`, `update_transfer.go` |
| `UpdateInvestment` | `finance/workflows/facade/investments/update.go` |

## Examples

Groceries of 125 from a card, entry 41, corrected to 12.50 from a bank account:

```text
UpdateExpense{journalEntryId: 41, balanceLedgerAccountId: <Cash (EUR)>,
              categoryLedgerAccountId: <Groceries (EUR)>, amount: 12.50, ...}
  -> entry 41: debit Groceries 12.50, credit Cash 12.50
```

A buy of 10 ACME, entry 50, corrected to 12 units:

```text
UpdateInvestment{journalEntryId: 50, kind: buy, buy: {units: 12, ...}}
  -> entry 50: same id, new postings, movement of +12 with a new id
```

Lowering a vest of 100 units to 40, after 60 have been sold:

```text
UpdateInvestment{journalEntryId: 51, kind: vest, vest: {unitsVested: 40, ...}}
  -> entry 51 replaced, with a warning that the holding is at -20
```

Moving a posting onto a closed account the entry never touched:

```text
UpdateExpense{journalEntryId: 41, balanceLedgerAccountId: <closed account>, ...}
  -> ErrResourceClosed, writing nothing
```

## Alternatives considered

- **Updating the entry in place**, with a new `ivfs.editing_journal_entry`
  setting. It needs exceptions in `prevent_settled_journal_entry_mutation` and
  `prevent_posted_posting_mutation`, and the balance trigger firing on every
  update, to end up where delete and re-insert already is.
- **Delete and record again with a new id.** Atomic, but the entry's URL and
  anything that later refers to it break, and the key is taken by a new row.
- **Keeping reversals and rewriting the chain on an edit.** It keeps a dated
  trail, at the cost in Context, for a person who can already edit or delete.
- **Refusing a change of type.** It protects nothing once an update states the
  whole entry, and needs code to derive the old type.
- **Refusing entries with position movements.** A trade's postings and movement
  come from one builder, and deleting and writing them together keeps them
  agreeing.
- **Keeping the old movement's id on a replace.** It keeps the order too, but
  matches old and new movements by position and leaves meaning in a surrogate
  key.
- **Increases before decreases on a tie.** It never reports a false negative,
  but a sale of everything and a new buy on one day would blend their basis.
- **A sequence column, or times of day from the UI.** Either would state which
  of a day's trades came first, where `created_at` infers it from the order of
  recording. Both need new input, so they are left for later. Nothing here rules
  them out.
- **Optimistic locking.** Guards against two devices editing the same entry,
  which is rare enough that losing one edit is acceptable.
- **A `RecordRefund` command.** An income on a "Refund" category records the
  same money. Its cost is that an expense category shows gross spending.

## Consequences

- An entry, a trade, or a vest can be corrected in one transaction, keeping its
  id and key. Nothing exposes this until a later decision adds the routes.
- The ledger keeps no trail of a correction, by edit or by deletion, and a
  report run twice can differ for a past date.
- Posting has no exception for closed accounts.
- Deletion and average cost lose their reversal cases.
- Entry-less movement groups (splits, transfers, adjustments) are corrected by
  deleting and recording them again, in two requests.
- A refund shows as income, so spending and earnings are both gross.
- The API loses `/reverse` and the reversal fields. The OpenAPI document and the
  generated UI client are regenerated.
- Existing tests that use a reversal as a fixture are rewritten, not deleted.
- `ReplaceJournalEntry` gets activity tests for a posted entry, a refused draft,
  an unknown entry, a closed account kept and a new one refused, each change of
  movements (added, removed, changed), negative holdings on both the old and new
  positions, a movement keeping its place among same-day movements, and an
  idempotency case. Each update gets a workflow test with
  mocked children, including a change of type.
- `design_principles.md`, `entity_relationships.md`, and `ledger_examples.md`
  are updated when this is implemented.

## Amendments to earlier ADRs

ADR 0001:

- `reversal_of_entry_id` is removed. A correction is an edit or a deletion, not
  a reversing entry.

ADR 0002:

- `ReversalOfEntryID` and `ReverseJournalEntry` are removed.

ADR 0004:

- `ReversalOfMovementID`, `ReversePositionMovement`, and reversing an entry's
  movements are removed. Movements stay append-only, and are written again only
  with their entry.
- Movements are ordered by `(occurred_at, created_at, id)`, where it says
  `(occurred_at, id)`.

ADR 0006:

- A wrong opening balance is corrected by editing or deleting it.

ADR 0007:

- `/api/journal-entries/{id}/reverse` is removed.

ADR 0008:

- "Correcting a transaction is reversing its entry, as for any entry
  (ADR 0001), and recording it again" becomes "Correcting a transaction is
  updating it (ADR 0017) or deleting its entry".

ADR 0009:

- A correction is an update or a deletion. Average cost has no reversals to
  skip.

ADR 0010:

- The writes that report negative holdings lose the two reversals, and gain
  `ReplaceJournalEntry`.

ADR 0011:

- "Reversals go with what they reverse" is removed, and "a reversal stays
  available for when a trail is wanted" no longer holds.
- "Editing is a delete and a new record, in two requests" becomes one
  transaction that keeps the entry's id (ADR 0017), except for entry-less
  movement groups.
- The closed-account rule rests on correcting mistakes, and no longer on a
  reversal being allowed.

ADR 0016:

- The section on reversals no longer applies.
