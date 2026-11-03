# ADR 0015: Net worth over time

- Status: Implemented
- Date: 2026-10-24
- Related: [ADR 0007](0007-web-ui-and-http-gateway.md) (one workflow per
  endpoint), [ADR 0009](0009-intent-level-investments.md) (what "Investments"
  holds), [ADR 0013](0013-users-and-per-user-databases.md) (where settings
  live), [ADR 0014](0014-asset-value-providers.md) (values and rates, which
  left net worth out of scope), [RFC 0002](../rfcs/0002-balances-of-every-account.md)
  (balances read at one moment)

## Context

A dashboard should chart net worth over a range, in one currency, and export
the series as CSV. Net worth on a date is the asset balances minus the
liability balances, plus the market value of the units held, all converted at
that date's rates.

Three constraints shape it:

- The ledger cannot value holdings. The balance of a brokerage's "Investments"
  is cash put in minus cash taken out (ADR 0009), neither cost nor market
  value, so it must be left out and replaced by units × value.
- Values and rates come from providers, chosen per asset in the browser
  (ADR 0014). A server-side valuation would need that choice sent to it.
- A workflow result is a Temporal payload: 2 MB at most, with a warning from
  512 KB, inside a 50 MB history. A dense daily series over ten years, three
  currencies and ten assets, is about 47,000 cells, near the limit.

## Decision

### The ledger answers quantities; the UI values them

One workflow, `NetWorthSeries`, returns what the database knows: the balances
per currency and the units per asset. The UI fetches values and rates from the
existing endpoints, through the provider chosen for each asset, and does the
arithmetic. The server never adds amounts in different currencies.

### The series is sparse

`NetWorthSeriesParams{From, To}` returns two lists, oldest first:

```text
Balances  [{Date, CurrencyID, Assets, Liabilities}]
Units     [{Date, AssetID, Units}]
```

- A row is emitted at `From` for every non-zero currency and asset, carrying
  in what was held before the range, and then only on dates where that
  currency's or asset's figure changes. The UI carries each figure forward,
  with the same rule as `assetvalue.CarryForward`.
- The value of a date is everything effective before its UTC day ends: posted
  entries by `occurred_at`, effective movements (ADR 0004). This is the
  providers' rule for values, so balances and prices line up by date.
- `Assets` and `Liabilities` are kept apart, each in its natural direction,
  so the chart can show both. Equity, income and expense are left out; they
  are the other side of the same postings.
- The asset sum excludes "Investments" on brokerage financial accounts, found
  by the rule `retrieveBrokerage` uses. Closed accounts are included: they
  held money back then.
- `Units` are summed across financial accounts. Negative holdings (ADR 0010)
  stay negative.
- There is no `interval`. Sparse rows cost the same whatever the chart's step,
  and the UI picks the chart's dates with `Range.Dates()`'s rules.
- More than 10,000 rows in total fails with `ErrInvalidParameter`, "the range
  holds N changes; shorten it". This is a guard on the payload, not a rule
  anyone should hit: 10,000 rows is under 1 MB.

One activity, `NetWorthSeries` in `ledger/reporting`, reads both lists in one
`REPEATABLE READ` transaction, so an entry posted mid-read shows in both or in
neither. It is read-only, so it has no idempotency test.

Exposed as `GET /api/net-worth/series?from=&to=`.

### A default currency per user

A `user_settings` table in the user's database holds a single row with a
nullable `default_currency_id` and a `gap_policy` (below). `RetrieveUserSettings` and
`UpdateUserSettings` are exposed as `GET` and `PUT /api/settings`; the
retrieve returns empty settings when the row does not exist yet. The dashboard
takes the currency from its URL (`?currencyId=`), then the setting; with
neither, it asks the person to pick one rather than guessing.

### Valuation in the UI

For each chart date, a module in `web/ui` computes:

```text
net worth = Σ_ccy (assets − liabilities) × rate(ccy → reporting)
          + Σ_asset units × value(asset) × rate(value ccy → reporting)
```

- Values come from `GET /api/assets/{id}/values` and rates from
  `GET /api/currencies/{id}/rates/{quoteId}`, both with the chart's `interval`:
  one request per held asset and per currency pair, whatever the range.
- The arithmetic is exact, on the decimal strings with `web/ui`'s BigInt
  helpers (`lib/decimal.ts`). A JS `number` is used only for chart
  coordinates, so the CSV export is exact.
- The module has Vitest tests, since no Go test covers it.

### The chart and the export

Recharts draws stacked areas for balances (the ledger's assets), investments
(units × value) and liabilities, with a net worth line. The CSV is built from
the same computed rows:
`date,balances,investments,liabilities,net_worth,currency,flags`.

### Dates without a value or a rate: a policy, exclude only for now

Values and rates are carried forward, so a gap only occurs before an asset's
first value or a pair's first rate, or for an asset that no provider values.
What a gap does is a per-user setting, `gap_policy` on `user_settings`:

- **`exclude`**, the default and the only one implemented. The point leaves
  out the holding without a value, or the figures in a currency without a
  rate, and lists each in `flags`. The chart shades flagged stretches. The
  line jumps on the day a price appears, but it never shows a figure nobody
  observed.
- **`cost_basis`**, reserved. A holding without a value counts at its cost
  basis, flagged. It needs cost per asset per date, so it adds a `CostBasis`
  list to `NetWorthSeriesResult`, read from the holdings replay (ADR 0009).
  A rate gap still excludes: cash has no cost to fall back on.

The column is text with a check constraint allowing `exclude` only, so
`UpdateUserSettings` rejects `cost_basis` until it is built. Building it widens
the constraint, adds the list and its branch in the valuation module; nothing
else changes.

## Alternatives considered

- **Net worth computed on the server**, in a workflow or in the gateway. Exact
  decimals in Go and available to the MCP server, but the provider choice
  lives in the browser and would have to be sent, and the gateway's fan-out
  would grow a second, larger use.
- **Dense rows**, one per date. Simpler to consume, but the size grows with
  range × step and hits the payload limit for daily ranges over years.
- **Rows per ledger account.** It allows a breakdown by account, but makes the
  payload bigger by the number of accounts. It can be added as a separate read.
- **Two activities**, for balances and for units. They fit the package layout
  better, but read at two moments.
- **Net per currency**, without assets and liabilities apart. Half the cells,
  but the chart could not show debt.
- **The reporting currency only as a request parameter.** Every device would
  start in a currency nobody chose.
- **Backfilling the first value or rate** over a gap. No jump, but it invents
  prices for anything volatile.
- **The gap policy as a dashboard toggle**, not a setting. Every visit would
  start from the default, and the CSV would depend on a toggle nobody sees.

## Consequences

- Net worth exists only in the UI. The MCP server, or a scheduled report,
  would have to repeat the valuation or move it to Go.
- Excluding "Investments" depends on the name and the account type, as
  `retrieveBrokerage` does. Renaming that balance would silently add the
  money put in to net worth; a key on the ledger account would remove the
  dependency.
- A dashboard visit makes one workflow execution, one settings read, and one
  provider fan-out per held asset and per currency pair.
- `web/ui` gains one dependency, `recharts`: the existing BigInt decimals
  cover the multiplications, so `big.js`, planned at first, was not needed.
- The settings table is the home for later per-user preferences, such as the
  provider choice ADR 0014 keeps in the browser.
- Until `cost_basis` is built, a holding priced only recently shows as a jump
  in net worth on its first priced day.
