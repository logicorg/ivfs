# RFC 0005: Holdings that go negative

- Status: Accepted as [ADR 0010](../adrs/0010-negative-holdings.md)
- Date: 2026-10-06
- Related: [ADR 0004](../adrs/0004-position-subledger.md) (holdings never
  negative), [RFC 0003](0003-deleting-journal-entries.md) (deleting journal
  entries), [RFC 0004](0004-deleting-accounts.md) (deleting accounts)

## Problem

ADR 0004 forbids a holding from being negative at any point in time. Every
write that adds a movement, a trade, a position movement, or a reversal, runs
`positions.CheckHoldingsTx` in its transaction, and is refused with
`ErrResourceConflict{ResourceType: "Position"}` if the running sum of units,
in date order, dips below zero anywhere.

The rule was meant to keep bad data out, such as a sale of shares never owned.
In practice it also refuses ordinary ways of using the app:

- **Entering history out of order.** A person who records this month's sale
  first, and backfills the buy from two years ago later, is refused: the sale
  comes before any units exist. Movements must be entered in chronological
  order, and an importer must sort its input.
- **Correcting in the wrong order.** Reversing a buy whose units were sold
  since is refused until the sale is reversed first. The vest of ADR 0004 is
  the same: it cannot be reversed while a sell-to-cover exists.
- **Deleting** (RFC 0003). Deleting a buy whose units were sold later would be
  refused for the same reason, so a person could not delete a mistaken
  operation without first deleting everything after it.
- **Deleting accounts** (RFC 0004). The stray financial account holds the other
  side of transfers to and from deleted accounts, which is negative whenever
  units moved from a deleted account to a surviving one.
- **Short positions.** A short sale is a real negative holding, which some
  brokers allow.

RFC 0003 and RFC 0004 depend on lifting the rule.

## The units ledger is the only one with this rule

Money is already allowed to go negative. A "Cash" balance below zero is an
overdraft, and nothing refuses an expense that takes it there. An amount typed
as 1000 instead of 100 is accepted on both ledgers alike. The units ledger is
the only place where one kind of typo, selling more than is held, is refused,
and the price is that the order of entry matters for every other write.

With RFC 0003, a mistake is fixed by deleting the operation, so refusing it up
front is less needed than when the only fix was a reversal.

## Proposal

### Remove the check

Holdings may be negative at any point in time. The four writes that call
`CheckHoldingsTx` stop calling it:

| Activity | File |
| --- | --- |
| `RegisterInvestmentTrade` | `finance/activities/ledger/portfolio/trade.go` |
| `RegisterPositionMovement` | `finance/activities/ledger/portfolio/register_movement.go` |
| `ReversePositionMovement` | `finance/activities/ledger/portfolio/reverse_movement.go` |
| `ReverseJournalEntry` | `finance/activities/ledger/journal/reversal.go` |

Movements can then be entered in any order. Their position in the history is
still `(occurred_at, id)`, which every report keeps using.

### What stays

- **The account lock.** `LockFinancialAccountsTx` still serialises writers to
  one account. It also makes a concurrent identical request wait and find the
  entry by its idempotency key, and makes a concurrent close wait for the
  write, so it is needed without the check.
- **Closing.** `CloseFinancialAccount` keeps refusing an account whose holdings
  are not zero. A negative holding is as stranded as a positive one.
- **The movement rules.** Non-zero units, the kinds, a transfer's two equal and
  opposite legs, and the cost basis pairing are unchanged.

### Cost basis

The basis is a signed sum, like the units, and needs no change. A sale removes
the basis the caller supplies, so a sale with no buy before it leaves a
negative basis, which a later backdated buy brings back. An average cost per
unit is undefined while the units are zero or negative; position results do
not expose one yet (ADR 0004), and whatever adds it must allow for that.

### Making a negative holding visible

Refusing was also the only way a person learnt that a holding went negative.
Without it, a sale of 100 instead of 10 would be accepted silently.

`CheckHoldingsTx` therefore becomes a read that returns the first negative
point of each position instead of an error. Each of the four writes returns it
in its result as a warning, and the UI shows it: "ACME in IBKR is -90 units on
3 March". The write is not refused. The warning reaches the person at the
moment the mistake is made, when it is cheapest to fix, by deleting the
operation (RFC 0003) or recording the missing buy.

### Short positions

A negative holding is tolerated, not modelled. A short sale is recorded as a
sale and shows as a negative number of units, with the signed basis above.
There is no notion of a short position, of its own cost basis, or of its
realised gains, which stay out of scope as they are for long positions
(ADR 0004).

## Options considered

- **Keeping the rule, with exceptions** for deletion, reversals, and the stray
  financial account. It keeps the typo check, but every exception is a path
  that has to prove it cannot hurt, and entering history out of order stays
  refused.
- **Allowing it per account**, with a flag on the financial account for the
  accounts that may hold short positions. It is the most faithful to a real
  broker, but it does not help backfilling or deleting, which happen on any
  account.
- **Checking only today's total.** It accepts history entered out of order as
  long as it ends non-negative, but refuses deleting a buy that was sold since,
  and ADR 0004 already rejected it for leaving negative points in the history
  that `PositionAt` reports.
- **A deferred check**, run when the person says the history is complete. It
  needs a notion of complete that nothing else has.
- **A list of positions that go negative at any point in their history**,
  instead of a warning. It suits history entered out of order, which is
  negative only until the backfill is done, but it is seen later than the
  mistake. It can be added later as a read.
- **No warning at all**, leaving a negative number in the position list as its
  own warning. Nothing tells the person when the mistake is made.

## What this would change

If adopted, the decision would be written as an ADR that amends ADR 0004:

- "Holdings never go negative" is removed, with the fifth step of the write
  path.
- The consequence "Movements must be entered in chronological order", and the
  note that importers must sort their input, are removed.
- "A reversal is rejected when the units it would restore have since been
  sold", and the vest that cannot be reversed while a sell-to-cover exists, no
  longer hold.
- "Allowing negative holdings" moves from the alternatives rejected to the
  decision.

`docs/handover.md` and `docs/todos.md` describe the rule and are updated with
it.

## Open questions

- **The check as a backstop.** Should the old check survive anywhere, for
  example as an opt-in for an importer that wants to be told about gaps in a
  broker's export?
