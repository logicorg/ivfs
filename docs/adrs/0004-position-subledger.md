# ADR 0004: Complete the position subledger

- Status: Implemented
- Date: 2026-09-18
- Depends on: [ADR 0001](0001-double-entry-ledger.md) and [ADR 0002](0002-ledger-domain-and-application-api.md)
- Amends: ADR 0001 (cost-basis currency) and ADR 0002 (`PositionMovement`, its invariants, and the positions commands)
- Amended by: [ADR 0009](0009-intent-level-investments.md), [ADR 0010](0010-negative-holdings.md), [ADR 0011](0011-deleting-journal-entries.md), and [ADR 0017](0017-editing-journal-entries.md)

## Context

ADR 0002 makes `PositionMovement` the source of truth for units owned, but only
`RegisterInvestmentTrade` writes it. Investigating the gaps showed that the
subledger cannot yet be relied on:

- A movement that has no journal entry is invisible to every position query,
  which joins through the entry. Grants, vests, splits, and transfers between
  own accounts therefore cannot be recorded.
- `PositionAt` fails whenever a point in time is given, because `occurred_at`
  is ambiguous between the movement and its entry.
- `ReverseJournalEntry` inverts only the first movement of an entry.
- Holdings can go negative: buying 10 and selling 50 is accepted and reports
  -40 units, with cash booked for shares that were never owned.
- ADR 0002 says a posted entry and its linked movements cannot be changed or
  deleted, but only `ledger.posting` is protected. The entry header and every
  movement can be updated or deleted, which leaves orphans and permits history
  to be rewritten.
- The schema declares no foreign keys, so the `on_delete:restrict` tags do
  nothing and rows can reference parents that do not exist.
- The cost basis is a bare number described as being in a "configured
  reporting currency" that does not exist. An asset can be held and traded in
  several currencies, so a bare number cannot be summed.

Batch 4 (RSU and ESPP workflows) needs all of this to be settled first.

## Decision

### Holdings

A holding is the sum of `UnitsDelta` per financial account and asset, counting
**vested or delivered units only**. Unvested grants and vesting schedules are
not modelled.

A movement is *effective* when it has no journal entry or its entry is posted.
Position queries count effective movements only, so drafts never contribute.

### Movement kinds

`PositionMovement` gains a `Kind`, which decides who may write it:

| Kind | Entry | Written by | Meaning |
| --- | --- | --- | --- |
| `trade` | required | `RegisterInvestmentTrade` | A buy or sell with its cash entry. |
| `vest` | required | The Batch 4 vest command | Units received from a grant, atomically with the compensation entry. |
| `split` | none | `RegisterPositionMovement` | A change of units with no cash effect. |
| `transfer` | none | `RegisterPositionMovement` | Units moved between two of the owner's financial accounts. |
| `adjustment` | none | `RegisterPositionMovement` | A units-only correction, with a required description. |

`RegisterPositionMovement` takes a kind and a list of legs, mirroring the
postings of a journal entry, so a transfer or a split across several accounts
is one atomic write. It rejects `trade` and `vest` and points the caller to the
command that owns them, so `RegisterInvestmentTrade` is not duplicated.

The kind rules are:

- `split`: one asset, any non-zero signed delta per account, no cost basis or a
  zero one. The ratio is recorded in metadata as `split_ratio`. It is not
  stored as a multiplier, which would break the additive sum and backdating.
  Callers supply the deltas, and there is no ratio-driven command.
- `transfer`: exactly two legs on different financial accounts with equal and
  opposite units and cost basis, in the same currency.
- `adjustment`: at least one leg. It is **not** the way to record an opening
  balance, because it has no ledger effect.

An opening balance is recorded with `RegisterInvestmentTrade` against an
`Equity:Opening balances` ledger account, so the units and the ledger stay
atomic and no further kind is needed.

### Cost basis

`CostBasisBaseAmount` is renamed `CostBasisAmount` and every movement gains an
optional `CostBasisCurrencyID`. The two are both set or both null. The currency
belongs to the movement and not to the asset, because one asset may be held in
several currencies. There is no reporting-currency setting, and converting
between currencies is left to reporting, which can use `Conversion` rates.

The amount is the signed change to the position's total basis and carries the
sign of the units. A NULL amount means the basis is unknown, which is not the
same as zero. A buy adds the cost, a sell removes the basis supplied by the
caller (lot selection is out of scope), a vest adds fair market value times
units, a split adds zero, and transfer legs are equal and opposite.

Position results do not expose basis totals yet. They need per-currency sums, a
flag for movements with an unknown basis, and Batch 4's sell-basis policy.
History already returns each row's basis and currency, so a total can be added
later without a breaking change.

### Idempotency and corrections

- An entry-linked movement is idempotent through its entry's key, as now.
- An entry-less movement carries its own `IdempotencyKey` and a `LineNumber`,
  unique together. Retrying with the same key returns the existing rows
  unchanged, following the retrieve-or-insert convention.
- `ExternalID` stays a non-unique provenance field.

A movement row is no longer append-only (ADR 0017). It is deleted with its
entry or its group (ADR 0011), and replaced when its entry is edited. A
correction edits the movement's entry, or deletes its entry or group and
records it again.
`ReversalOfMovementID`, `ReversePositionMovement`, and reversing an entry's
movements are removed (ADR 0017).

### Holdings never go negative

Holdings must not be negative at **any** point in time, backdated movements
included. Movements are therefore entered in chronological order. Ties on
`occurred_at` are ordered by `id`, so within one instant a buy must be
inserted before a sell.

The activity enforces this inside its transaction:

1. Lock the affected `financial_account` rows in ascending id order with
   `FOR NO KEY UPDATE`. This serialises concurrent writers to the same account
   without blocking child inserts, and it also validates that the accounts
   exist and are open. A closed account rejects the write with
   `ErrResourceClosed`.
2. Look up the idempotency key, which removes the duplicate-key race.
3. Insert the entry, postings, and movements.
4. Mark the entry posted.
5. For each affected account and asset, compute the running sum ordered by
   `(occurred_at, created_at, id)` over effective movements (ADR 0017). If it is negative anywhere the
   transaction rolls back with `ErrResourceConflict{ResourceType: "Position"}`
   that names the first negative point.

Trades, transfers, and position movements all use this path.
`CloseFinancialAccount` refuses
while any position it holds is non-zero.

### Queries

- `PositionAt` counts effective movements with `occurred_at <= At`.
- `ListPositionMovements` filters by financial account, asset, kinds, and a
  half-open date range (`From` inclusive, `To` exclusive), ordered by
  `(occurred_at, created_at, id)` with a keyset cursor (ADR 0017). The sum of the history up to a
  time equals `PositionAt` at that time.

### Write order and the movement date

An entry-linked movement is written with its entry, inside the same
transaction, and the entry is inserted as posted (ADR 0017). The trigger that
guarded movements is removed. A foreign key on the entry's id and
`occurred_at` keeps a movement on the date of its entry, and moves it when the
entry's date changes. A shared helper writes the entry, postings, and
movements, and the trade and vest paths both use it.

### Ledger-wide integrity

- Nothing freezes a posted entry (ADR 0017). A deferred trigger checks its
  balance whenever the entry or one of its postings is written, so an entry is
  deleted whole, with its postings, in one transaction (ADR 0011).
- Foreign keys are declared across the schema, all `ON DELETE RESTRICT`, so a
  used row can never be deleted and history is never destroyed or nulled,
  except an entry, its postings, and its movements, which are deleted together
  (ADR 0011). They are added after the tables exist, with the same duplicate
  tolerance as the other ledger constraints.
- A foreign-key violation is mapped to `ErrResourceConflict` when a used
  parent is deleted and to `ErrNotFound` when a parent is missing. Left raw it
  would be retried by Temporal for no benefit.
- Check constraints on the movement enforce the allowed kinds, non-zero units,
  that `trade` and `vest` have an entry and the other kinds do not, that an
  entry-less movement has an idempotency key, and the basis pairing above.

### Batch 4

A vest is one atomic write: the compensation entry and its `vest` movement go
through the shared helper in one transaction. Sell-to-cover is an ordinary
`RegisterInvestmentTrade` whose entry carries the withholding postings. Income,
tax, cash, and units are then all reconstructible from ledger facts. An ESPP
purchase is a trade whose entry carries the discount and payroll postings.
A vest is corrected by deleting its entry (ADR 0011), in any order with its
sell-to-cover (ADR 0010).

Fair-market-value lookup, tax computation, payroll and broker parsing, vesting
schedules, lot selection, realised gains, and wash sales stay out of scope.

## Alternatives considered

- **A status or effective flag on the movement.** It duplicates state the
  entry already carries.
- **Dropping the join to the entry.** It holds only while every movement is
  written with a posted entry, and breaks once the draft lifecycle exists.
- **Recording a vest as a movement plus a separate entry, with a compensating
  workflow on failure.** Rejected. Cleaning up a posted entry needs a reversal,
  so the ledger would show attempts that failed, readers would see half of the
  write in between, and cleanup does not run when a workflow is terminated.
  ADR 0002 already requires one atomic write.
- **A global reporting currency for the cost basis.** It needs a setting and
  historical exchange rates, and changing it would silently change the meaning
  of past facts.
- **Allowing negative holdings, or checking only the current total.** The
  first accepts bad data such as a sale of shares never owned. The second lets a
  backdated sale leave a negative point in the history that `PositionAt`
  reports.
- **A ratio-driven stock split command.** It needs a lock, holdings at a past
  time, and rules for trades recorded late, and it adds surface for little gain.
- **Enforcing the holdings check with a trigger.** It adds a second code path
  and is heavier. It can be added later as a backstop.

## Consequences

- Units held include trades and every legitimate non-monetary movement, and
  every position query agrees.
- Movements must be entered in chronological order. Importers in Batch 5 must
  sort their input, or import in date order, because an out-of-order export
  would be rejected.
- An opening balance for an existing holding needs a trade against an equity
  account, which needs a cost or a stated value.
- Deleting a used row now fails cleanly instead of leaving orphans, and some
  test fixtures that referenced missing rows have to create them.
- Basis totals per currency are deferred to Batch 4.

## Amendments to earlier ADRs

ADR 0001:

- The cost basis is expressed in a per-movement currency. There is no
  "configured reporting currency".

ADR 0002:

- `PositionMovement` gains `Kind`, `IdempotencyKey`, `LineNumber`,
  `CostBasisCurrencyID`, `Description`, and
  `Metadata`, and `CostBasisBaseAmount` is renamed `CostBasisAmount`.
- "A posted entry, its postings, and its linked monetary position movements
  cannot be changed or deleted" now covers every movement, and is enforced by
  triggers instead of only by convention.
- "A monetary purchase or sale position movement references the journal entry"
  extends to a `vest`.
- The entry, postings, and movements are written in one transaction, and the
  entry is inserted as posted (ADR 0017).
- `RegisterInvestmentTrade` is no longer the only domain-specific financial
  command. `RegisterFXConversion` already exists and the Batch 4 vest command
  is another.
- "Under explicit policy" for `RegisterPositionMovement` means the kinds
  `split`, `transfer`, and `adjustment`. `ListPositionMovements` is added, and
  `PositionAt` counts effective movements.
- The persistence invariants gain the never-negative holdings rule, foreign
  keys with `RESTRICT`, and the journal-entry trigger.
