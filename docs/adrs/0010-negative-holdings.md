# ADR 0010: Holdings that go negative

- Status: Implemented
- Date: 2026-10-06
- Depends on: [ADR 0004](0004-position-subledger.md)
- Amends: ADR 0004 (holdings never go negative, and the write path that enforces it)
- Amended by: [ADR 0009](0009-intent-level-investments.md) (the basis a sale removes)
  and [ADR 0017](0017-editing-journal-entries.md) (the writes that report)
- Related: [RFC 0005](../rfcs/0005-negative-holdings.md) (the discussion this
  ADR decides), [RFC 0003](../rfcs/0003-deleting-journal-entries.md) (deleting
  journal entries), [RFC 0004](../rfcs/0004-deleting-accounts.md) (deleting
  accounts)

## Context

ADR 0004 forbids a holding from being negative at any point in time. Every
write that adds a movement runs `positions.CheckHoldingsTx` in its transaction,
and is refused with `ErrResourceConflict{ResourceType: "Position"}` if the
running sum of units, in date order, dips below zero anywhere.

The rule was meant to keep out a sale of shares never owned. It also refuses
ordinary ways of using the app:

- **Entering history out of order.** Recording this month's sale before
  backfilling the buy from two years ago is refused, so movements must be
  entered in chronological order and an importer must sort its input.
- **Correcting in the wrong order.** A buy whose units were sold since cannot be
  reversed until the sale is, and a vest cannot be reversed while a
  sell-to-cover exists.
- **Deleting** (RFC 0003). Deleting a buy whose units were sold later would be
  refused, so a mistaken operation could not be deleted without first deleting
  everything after it.
- **Deleting accounts** (RFC 0004). The stray financial account holds the other
  side of transfers to and from deleted accounts, and goes negative whenever
  units moved from a deleted account to a surviving one.
- **Short positions**, which some brokers allow.

The units ledger is the only one with such a rule. A cash balance can already go
below zero, and a mistyped amount is accepted on both ledgers alike. Once a
mistake can be fixed by deleting the operation (RFC 0003), refusing it up front
is less needed than when a reversal was the only fix.

## Decision

### Holdings may be negative

A holding may be negative at any point in time. The writes that call
`CheckHoldingsTx` no longer refuse (ADR 0017 removes the two reversals from
them):

| Activity | File |
| --- | --- |
| `RegisterInvestmentTrade` | `finance/activities/ledger/portfolio/trade.go` |
| `RegisterPositionMovement` | `finance/activities/ledger/portfolio/register_movement.go` |

Movements can be entered in any order. Their place in the history is still
`(occurred_at, id)`, which every report keeps using.

### A negative holding is a warning

Refusing was the only way a person learnt that a holding went negative. To keep
a sale of 100 instead of 10 from passing silently:

- A new read, `positions.NegativeHoldingsTx`, alongside `CheckHoldingsTx`,
  runs the same running sum. For each financial account and asset it is given, it
  returns the first point where the holding is negative, with the date and the
  units held there, and returns no error for it.
- Each of the four writes calls it after writing, in the same transaction, in
  place of `CheckHoldingsTx`, and returns what it finds in its result as a list
  of warnings, one per negative position. The write is not refused. A retry that finds the earlier write by its idempotency key returns
  the warnings as they are now.
- The workflows and the HTTP gateway pass the warnings through, and the UI shows
  them: "ACME in IBKR is -90 units on 3 March".

The warning reaches the person when the mistake is made, when it is cheapest to
fix, by deleting the operation (RFC 0003) or recording the missing buy.
`RegisterVest` only adds units and has no warning, and the deletions of RFC 0003
and RFC 0004, which can remove units, return it too.

### What stays

- **The account lock.** `LockFinancialAccountsTx` still serialises writers to
  one financial account, in ascending id order. It also makes a concurrent
  identical request wait and then find the entry by its idempotency key, and a
  concurrent close wait for the write, so it is needed without the check.
- **Closing.** `CloseFinancialAccount` keeps refusing an account whose holdings
  are not zero. A negative holding is as stranded as a positive one.
- **The movement rules.** Non-zero units, the kinds, a transfer's two equal and
  opposite legs, and the cost basis pairing are unchanged.

### Cost basis

The basis is a signed sum, like the units, and needs no change. A sale removes
the basis the caller supplies, so a sale with no buy before it leaves a negative
basis, which a later backdated buy brings back. An average cost per unit is
undefined while the units are zero or negative. Position results do not expose
one yet (ADR 0004), and whatever adds it must allow for that.

### Short positions

A negative holding is tolerated, not modelled. A short sale is recorded as a
sale and shows as a negative number of units, with the signed basis above.
There is no notion of a short position, of its own cost basis, or of its
realised gains, which stay out of scope as they are for long positions.

## Example

A person records this month's sale of 10 ACME in IBKR first. The write succeeds
and returns a warning that ACME in IBKR is -10 units from the sale's date. They
then backfill the buy of 10 from two years ago, which succeeds with no warning,
and the holding is 10 until the sale and 0 after it.

Had they typed 100 for the sale, the warning would name -90 units on the sale's
date, and they would delete the sale and record it again.

## Alternatives considered

- **Keeping the rule, with exceptions** for deletion, reversals, and the stray
  financial account. It keeps the typo check, but every exception is a path
  that has to prove it cannot hurt, and entering history out of order stays
  refused.
- **Allowing it per account**, with a flag on the financial accounts that may
  hold short positions. It is the most faithful to a real broker, but does not
  help backfilling or deleting, which happen on any account.
- **Checking only today's total.** It accepts history entered out of order as
  long as it ends non-negative, but refuses deleting a buy that was sold since,
  and ADR 0004 already rejected it for leaving negative points in the history
  that `PositionAt` reports.
- **A deferred check**, run when the person says the history is complete. It
  needs a notion of complete that nothing else has.
- **A list of positions that go negative anywhere in their history**, instead of
  a warning. It suits history entered out of order, but is seen later than the
  mistake. It can still be added as a read.
- **No warning**, leaving the negative number in the position list as its own
  warning. Nothing tells the person when the mistake is made.

## Consequences

- Movements can be entered in any order, and importers (Batch 5) need not sort
  their input.
- Corrections can be made in any order, and a vest can be deleted while a
  sell-to-cover exists.
- RFC 0003 and RFC 0004 no longer need a holdings check of their own.
- A sale of more than is held is accepted, and is caught by the person reading
  the warning rather than by the ledger.
- The results of the four writes gain a field, and their callers, the workflows,
  the gateway, and the UI, pass it on.
- The tests that expect a refusal for a negative holding expect a warning
  instead.
- Whether the old check should survive anywhere, for example as an opt-in for
  an importer that wants to be told about gaps in a broker's export, is left
  open. `CheckHoldingsTx` is kept unchanged for it, with no caller until then.
- `docs/handover.md` describes the rule as built and is updated when this is
  implemented.

## Amendments to earlier ADRs

ADR 0004:

- "Holdings never go negative" is removed. The write path keeps steps one to
  four, and the fifth, the holdings check, is replaced by the read that produces
  the warnings.
- The consequence "Movements must be entered in chronological order", and the
  note that importers must sort their input, are removed.
- "A reversal is rejected when the units it would restore have since been
  sold", and the vest that cannot be reversed while a sell-to-cover exists, no
  longer hold.
- "Allowing negative holdings" moves from the alternatives rejected to the
  decision. "Checking only the current total" stays rejected.
- "Enforcing the holdings check with a trigger", kept as a possible backstop,
  no longer applies.
