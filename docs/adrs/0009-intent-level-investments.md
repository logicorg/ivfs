# ADR 0009: Intent-level investments

- Status: Implemented
- Date: 2026-10-04
- Depends on: [ADR 0004](0004-position-subledger.md), [ADR 0006](0006-intent-level-account-setup.md), [ADR 0007](0007-web-ui-and-http-gateway.md), and [ADR 0008](0008-intent-level-transactions.md)
- Amends: ADR 0004 and ADR 0010 (the basis a sale removes: a decrease without
  a basis now removes the average cost, worked out when read)

## Context

ADR 0008 raised expenses, incomes, and transfers to the level a person thinks
at, and left trades and vests out.
Those workflows, `RegisterInvestmentTrade` and `RegisterVest`, still speak the
ledger's language: the caller supplies the postings, the signed units, and the
cost basis, and has to know that a buy debits "Investments" and credits
"Cash", etc. The web UI has no screen for investments at all: no way to choose
or create an asset, to see what an account holds, or to record a buy, etc.

A sale has a further gap. `RegisterInvestmentTrade` removes whatever basis the
caller gives, and `docs/todos.md` leaves open "a policy for the basis a sale
removes". The example sales so far are at cost, so no gain or loss has ever
had an account to go to.

Whatever the policy, the basis a sale removes depends on every movement of the
holding before it, and those can arrive in any order: an import may bring a
sale before the buys it sold (ADR 0010), or a forgotten buy may be backdated
later. A number derived from history and frozen in an immutable entry is wrong
as soon as history changes.

## Decision

### One workflow, a builder per kind

A new facade package, `finance/workflows/facade/investments`, has a single
workflow, `RecordInvestment`. Its input names a kind and carries that kind's
facts, and each kind has a **builder**: a pure function, one per file, that
turns the facts and the accounts the workflow looked up into an entry's
postings and a position movement. The first three kinds are:

| Kind | A person says | Entry | Movement |
| --- | --- | --- | --- |
| `buy` | "I bought 10 VWCE for 1,050 EUR on IBKR, with a 3 EUR fee." | debit Investments 1,053, credit Cash 1,053 | trade, +10, basis 1,053 |
| `sell` | "I sold 4 VWCE for 480 EUR on IBKR, with a 3 EUR fee." | debit Cash 477, credit Investments 477 | trade, -4, no basis |
| `vest` | "100 ACME RSUs vested at 10 USD each, and the employer kept 40 for tax." | credit the income category 1,000, debit Investments 600, debit the tax category 400 | vest, +60, basis 600 |

The workflow validates the input and looks up the accounts it references with
read-only activities. It then calls the kind's builder in the workflow itself
(builder does no I/O, is deterministic, so it is safe to replay).
Last, a single activity writes the entry and the movement in one transaction: 
`RegisterInvestmentTrade` for a trade movement, `RegisterVest` for a vest. 
Neither activity changes. What may change between the lookups and the
write, a closed account or a deleted asset, the write activity refuses inside
its transaction (see Checks). A kind that needs more than one movement, such as
a transfer between brokerages, brings with it a ledger activity that writes an
entry with several movements.

Every kind acts on a `brokerage` financial account (ADR 0006), whose "Cash" and
"Investments" balances in the trade's currency the workflow finds by their key.
The person never picks a ledger account for the brokerage side, only the
financial account and the currency.

The special cases an investment account accumulates (an ESPP purchase, an
opening holding, a transfer, a spin-off) differ only in their arithmetic, and
arithmetic needs no workflow. A new kind is a builder, its table-driven test,
and a value in the kind's enum: no new workflow or activity.

A fee is part of the cost of a buy and reduces the proceeds of a sale, so it
enters the basis and the realised gain rather than an expense category. This
keeps a buy and a sale to the cash and investments accounts.

### Selling: store the facts, work out the basis when read

The ledger and the subledger record only what happened. A buy records what was
paid and the units received, so its movement carries the cost as its basis. A
sale records what was received and the units given, so its entry moves the net
proceeds from "Investments" to "Cash", and its movement carries **no basis**.
`RegisterInvestmentTrade` already accepts both and needs no change.

The basis a sale removes, and so its realised gain, is derived when read, by
replaying the effective movements of a holding in the order they occurred
(`occurred_at`, then `id`, as the negative-holdings check does). Per holding,
a running units total and a running basis per currency start at zero, and each
movement changes them:

| Movement | Units | Basis |
| --- | --- | --- |
| an increase with a basis (buy, vest, transfer in) | add | add the basis |
| an increase without a basis | add | unknown from here on |
| a decrease with a basis (a caller-chosen basis, a transfer out) | subtract | add the basis, which is negative |
| a decrease without a basis (a `sell`) | subtract | remove the **average cost**: basis × units sold ÷ units held |

The average is undefined while the units held are zero or negative (ADR 0010),
while the basis is unknown, or when the basis is spread over more than one
currency. The removal is then unknown, and so is the basis after it. A
decrease without a basis that brings the units to exactly zero removes all the
basis, whatever it was, so the holding starts again from a known zero. None of
this refuses a write: a backdated buy, or the import of the buys a sale sold,
simply changes the answer on the next read.

A movement that gives a basis on a decrease is honoured as given, so a sale
priced from a broker's statement, or a transfer's equal and opposite legs
(ADR 0004), keep working. Carrying the average cost across a transfer, without
the caller giving it, is left for later.

Average cost has no reversals to skip, since ADR 0017 removes them.

Lot selection (FIFO, specific lots) stays out of scope (ADR 0004). Average cost
is chosen because it needs no lots, is additive, and matches how the basis is
already stored.

### Realised gains

There is no realised-gains ledger account. A sale's gain is its proceeds, the
amount its entry credits to "Investments", minus the basis the replay removes
for it, and a report computes it the same way the holdings read does. As a
consequence, the balance of "Investments" is the cash put in minus the cash
taken out, not the cost of what is held: the two differ by the realised gains
to date. The UI shows an investment account's holdings, from the holdings read,
rather than its ledger balance.

Bringing "Investments" to cost in the ledger, with an adjustment entry against
an income account that is recomputed and replaced (ADR 0011) when history
changes, is left for later.

### Vests

The `vest` kind takes the facts and builds the entry, so the caller no longer
supplies `RegisterVest`'s postings:

| Field | Required | Meaning |
| --- | --- | --- |
| `financialAccountId` | yes | the brokerage or stock plan the units are delivered to |
| `assetId` | yes | |
| `unitsVested` | yes, positive | before any withholding |
| `unitsWithheld` | no, zero or more and fewer than vested | units the employer kept for tax |
| `fairMarketValue` | yes, positive | per unit, on the vesting date |
| `currencyCode` | yes | of the fair market value |
| `incomeCategoryLedgerAccountId` | yes, an income category | e.g. "RSU compensation" |
| `taxCategoryLedgerAccountId` | when units are withheld, an expense category | e.g. "Tax withheld" |

The entry credits the income with vested × value, debits "Investments" with
delivered × value, and debits the tax category with withheld × value. The
movement delivers `unitsVested - unitsWithheld` units at a basis of delivered
× value. Values are rounded to the currency's precision; the income is the sum
of the two debits, so the entry always balances.

**Sell-to-cover** is not a vest: it is a `sell` recorded after it. The UI's
vest form offers to record one in the same submission, as a second request
with its own idempotency key, sent only once the vest succeeded. Its gain is
measured against the average cost of the whole holding, so it is zero only
when the vest is all the account holds.

An ESPP purchase, and an opening holding (a buy against "Opening balances",
ADR 0004), are left for later. Each fits the same shape as `buy` with a
different credit account.

### Inputs

`RecordInvestment` takes a `kind` and one field per kind (`buy`, `sell`,
`vest`), of which exactly the one named by `kind` is set. The facts of the
first three kinds are:

| Field | Buy | Sell | Vest |
| --- | --- | --- | --- |
| `idempotencyKey` | yes | yes | yes |
| `financialAccountId` | yes, a brokerage | yes, a brokerage | yes, a brokerage |
| `assetId` | yes | yes | yes |
| `units` | yes, positive | yes, positive | see above |
| `amount` | yes, positive, before the fee | yes, positive, before the fee | |
| `fee` | no, zero or more | no, zero or more, less than `amount` | |
| `currencyCode` | yes | yes | yes |
| `occurredAt` | yes | yes | yes |
| `description` | no | no | no |

The input types live in `finance/common/types/investmenttypes`, validated with
CUE constraints as the other facades' are.

### Checks

The workflow fails with `ValidationError`, when the financial account has no
"Cash" and "Investments" balance in the currency, which is how a non-brokerage
account or a missing currency shows up. It fails the same way when a category
a kind references is not of the kind it needs (an income category for a vest's
income, an expense one for its tax) or is not in the trade's currency. A closed
account and an unknown asset are left to the activity, which already refuses
them inside the transaction that posts, with `ErrResourceClosed` and `ErrNotFound`.
A sale of more units than are held is written, and it gives a warning (ADR 0010).

### Idempotency

As in ADR 0008, the key is required and made by the UI per opening of a form,
and it is posted with the kind in front of it: `buy:<key>`, `sell:<key>`,
`vest:<key>`. A retry returns the entry and movement first written, unchanged.

### Results

The workflow returns the journal entry id and the position movement id. A
correction is an update (ADR 0017) or a deletion of the entry, which takes
the movement with it (ADR 0011).

### Reads the UI needs

- **Assets**: `GET /api/assets` over `ListAssets`, and `POST /api/assets` over
  `InsertAsset`, so the forms can pick an asset or create one by name and
  ticker.
- **Holdings**: `GET /api/holdings` over `ListPositionsAt`, with the
  financial account, asset, and an optional `at`. Each row gains the basis the
  replay above leaves, per currency, its average cost when defined, and a flag
  for an unknown basis, which closes the todo about basis totals in position
  results. Market value is not shown: there are no prices.
- **History**: `GET /api/holdings/movements` over `ListPositionMovements`.

### Exposure

The gateway exposes `POST /api/investments/buys`, `/sells`, and `/vests`, each
starting `RecordInvestment` with its kind, so every form keeps a typed request
and the gateway only chooses the kind. The
UI gains an "Investments" screen: holdings grouped by brokerage account, with
units, cost basis, and average cost, and buttons to buy, sell, and record a
vest; a holding opens its movement history, where an entry can be deleted
through the existing journal entry sheet.

## Alternatives considered

- **A workflow per intent** (`BuyAsset`, `SellAsset`, `RecordVest`, ...), as
  this ADR first decided. Each is thin over a shared core, but every special
  case then adds a workflow, its registration, and its tests of the same
  lookups, and their number grows with every case an investment account
  meets.
- **Building the postings in the UI over the existing endpoints.** The rules
  about sides, basis, and gains would exist only in TypeScript, and MCP would
  repeat them. Building them in the gateway has the same problem, and ADR 0007
  keeps the gateway free of logic.
- **Booking the gain at the sale, computed inside the trade's transaction.**
  The sale's entry would credit "Investments" with the basis removed and a
  "Realised gains" income account with the rest, so the ledger alone would
  show the cost held and the gains. But the basis is computed from the history
  at the time of writing and frozen in an immutable entry: a buy imported or
  backdated later leaves it wrong for good, and a sale before its buys, which
  ADR 0010 allows, would have to be refused for want of an average. Computing
  it in the workflow, then trading, is worse still, as a movement can land
  between the read and the write.
- **The caller always supplies the basis removed.** It is today's behaviour,
  and stays available, but a person filling in a form does not know it.
- **FIFO or specific lots.** Needs lot tracking, which ADR 0004 left out.
- **Fees to an expense category.** Simpler postings, but the basis and the
  realised gain would then disagree with how the fee is taxed in most places.
- **Recording sell-to-cover inside the vest.** Two facts in one command; it
  would duplicate the sale's checks and its basis computation.

## Consequences

- The UI records buys, sales, and RSU vests without the person choosing sides,
  basis, or gains.
- A new kind of investment transaction is a builder and its test. The builders
  are pure, so they are tested without Temporal or a database; changing one
  changes the workflow's code, which is safe as long as its runs are short.
- `RegisterInvestmentTrade` is unchanged. Writes depend on no history, so
  movements can be recorded in any order and a backdated one corrects every
  later basis and gain by itself.
- A decrease without a basis means "at average cost", no longer "unknown"; an
  increase without one still means unknown.
- The ledger shows no realised gains, and "Investments" holds net cash flow
  rather than cost. Gains are a report over the movements and their entries,
  and every read of a basis replays the holding's history.
- Position results carry their basis; market value, unrealised gains, and
  performance wait for prices.

## Delivery

Each step is one reviewable change:

1. The basis in position results: the replay, per currency, with average cost
   and the unknown flag. Done.
2. `investmenttypes`, `BuyAsset`, and `SellAsset`, as a workflow each. Done.
3. The gateway endpoints for assets, holdings, buys, and sales, and the UI's
   Investments screen with its buy and sell forms. Done.
4. `RecordInvestment` with the `buy` and `sell` builders, replacing `BuyAsset`
   and `SellAsset`; the endpoints start it instead. Done.
5. The `vest` builder, and `POST /api/investments/vests`. Done.
6. The UI's vest form, with its sell-to-cover. Done.
7. The UI's movement history of a holding, with reversal. Done.
