# ADR 0014: Asset value providers

- Status: Implemented
- Date: 2026-10-11
- Amends: [ADR 0007](0007-web-ui-and-http-gateway.md) (the gateway adds no logic)
- Related: [ADR 0009](0009-intent-level-investments.md) (cost basis, which this
  is not), [ADR 0013](0013-users-and-per-user-databases.md) (whose database the
  manual values are read from)
- Contract: [`docs/specs/asset-value-provider.v1.yaml`](../specs/asset-value-provider.v1.yaml)

## Context

The only way to know what 20 AAPL are worth is an `AssetValue` row, entered by
hand through the `assetvalues` workflows. That suits an asset whose price is 
hard to find, but for a listed stock or ETF it is busywork: the price is on 
Yahoo Finance and on many other sites. The same goes for exchange rates,
entered by hand as `Conversion` rows, and needed to value a holding in another
currency.

What is wanted:

- prices from several sources, starting with the manual values and Yahoo;
- exchange rates from the same kind of sources, by a source that has them,
  whether or not it also has prices;
- a source added without changing ivfs, by anyone, in any language;
- a value for every day of a range, for a chart or a valuation over time,
  as well as for a single day;
- a choice of source per asset, defaulting to the first one that answers;
- the choice and the fetched prices kept out of Temporal and out of the
  database.

## Decision

### A provider is an HTTP service implementing one endpoint or both

The contract is an OpenAPI document,
[`asset-value-provider.v1.yaml`](../specs/asset-value-provider.v1.yaml), with
two endpoints: the values of an asset and the rates between two currencies. A
provider implements either or both. The one it does not implement answers
`404`, as any HTTP server does for a path it does not serve, which reads as
"no value here", so a provider declares nothing about what it implements.

For asset values, a provider answers:

```
GET {baseURL}/v1/values?isin=US0378331005&ticker=AAPL&exchange=NASDAQ&kind=stock&from=2026-10-01&to=2026-10-10
```

- The query carries the identifiers the asset has, as stored on `Asset`. ivfs
  does not translate them: mapping them to the provider's own symbols, such as
  Yahoo's `VWCE.DE`, is the provider's job.
  Asset IDs are not sent, they mean nothing outside one user's database.
- `from` and `to` are dates, both included and both required. A single day
  is `from` equal to `to`; today's value is both set to today.
- The value of a date is the latest one observed before the UTC day ends, the
  rule `RetrieveAssetValue` applies to the manual values, taken date by date.
  A day with no observation of its own, such as a Saturday, repeats the last
  one before it, so there are no gaps.
- `interval` is `day` (the default), `week` or `month`. The answer has one
  value per interval, dated its last day: every day, every Sunday, or the last
  day of every month. When `to` falls inside an interval, the last value is
  dated `to`, so the latest value is always there. A year by month is 12
  values rather than 365, for a chart over decades.
- The values start at the first of those dates the provider has a value for
  and go on to `to`.
- `200` returns `{"values": [{"date": "2026-10-10", "value": "231.45",
  "currency": "USD", "observedAt": "2026-10-09T20:00:00Z"}, ...]}`, oldest
  first and never empty: the value of one unit, as a decimal string so it
  stays exact, an ISO 4217 code, and when it was really observed.
- `404` means the provider has no value for the asset on any day of the range.
  It is an ordinary answer, not a failure, and is how a provider declines an
  asset it does not cover.
- Errors are RFC 9457 problems, as the gateway's own.

For exchange rates, it answers:

```
GET {baseURL}/v1/rates?base=EUR&quote=USD&from=2026-10-01&to=2026-10-10
```

- `base` and `quote` are ISO 4217 codes; the rate is how many units of quote
  one unit of base is worth, as on a `Conversion` row from base to quote.
- Days, `from`, `to`, `interval`, the carrying forward, `404` and errors
  are as for values.
- `200` returns `{"rates": [{"date": "2026-10-10", "rate": "1.0921",
  "observedAt": "2026-10-09T16:00:00Z"}, ...]}`, with up to 18 decimal places,
  the scale of `Conversion.rate`.
- Whether a provider inverts a pair it only has the other way round, or
  crosses two pairs, is up to it. ivfs asks for the pair as it needs it.

The version is in the path, so a v2 can live next to v1 on the same provider.

### The providers ivfs knows

The gateway reads an ordered list from its configuration:

```
IVFS_ASSET_VALUE_PROVIDERS=manual,yahoo=http://yahoo-provider:8080
```

The order is the default preference. Each entry is a name, which is what the UI
shows and remembers, and a base URL. Providers are configured by whoever runs
ivfs, for every user alike. A user cannot add one.

`manual` is built in and has no URL, and reads the requesting user's database
(ADR 0013): values through the existing `RetrieveAssetValue` workflow, rates
through a new workflow over the existing
`RetrieveMostRecentCurrencyConversionRatesAtDate` activity, which no workflow
exposes yet. In Go both kinds sit behind one interface, so the gateway treats
them the same:

```go
type Provider interface {
	Values(ctx context.Context, asset model.Asset, r Range) ([]Value, error)
	Rates(ctx context.Context, base, quote string, r Range) ([]Rate, error)
}

type Range struct {
	From, To time.Time
	Interval Interval // day, week or month
}
```

with an HTTP client implementing the spec for every external provider. A
provider without one of the two returns `ErrNotFound` from it, as `404` on
the wire. The manual values and rates are not served through the spec,
because the gateway would have to call itself and forward the user's
identity, for no gain.

Yahoo is the first external provider: a separate binary in this repository
with its own compose service, written against the spec as anyone else's would
be.

### The gateway asks every provider and the UI chooses

A new endpoint, `GET /api/assets/{id}/values?from=&to=&interval=`, retrieves the asset with
the existing workflow, then asks every provider at once, each bounded by a
timeout (5 seconds by default), and answers in the configured order:

```json
{
  "values": [
    {"provider": "manual", "values": [
      {"date": "2026-10-10", "value": "229.10", "currencyId": 2, "observedAt": "2026-10-08T00:00:00Z"}
    ]},
    {"provider": "yahoo", "values": [
      {"date": "2026-10-10", "value": "231.45", "currencyId": 2, "observedAt": "2026-10-09T20:00:00Z"}
    ]}
  ],
  "errors": [
    {"provider": "other", "detail": "no answer within 5s"}
  ]
}
```

- A provider that answers `404` is left out.
- A provider that fails, times out, or answers something that breaks the spec
  is listed in `errors`. One provider being down never hides the rest.
- A value in a currency the user has not set up is an error for that provider,
  "currency USD is not set up; add it under Currencies", so the response speaks
  in currency IDs like every other endpoint.

Rates have the same shape at
`GET /api/currencies/{id}/rates/{quoteId}?from=&to=&interval=`,
with the pair given as currency IDs and sent to providers as codes, and
`rates` lists holding `{"date", "rate", "observedAt"}`.

The UI shows the first value and lets the person pick another provider for
each asset. The choice is remembered in the browser, keyed by asset ID. Nothing
is stored in Temporal or in the database, and fetched values are not written
back as `AssetValue` rows.

### No new activity

The fan-out runs in the gateway, not in a workflow. This is the exception to ADR 
0007's "it adds no business logic": the gateway composes the workflows' answer 
with the providers', and decides nothing about the ledger.

## Example

A person holds 20 AAPL and 100 units of an unlisted fund. The gateway is
configured with `manual,yahoo`.

- For AAPL, there is no manual value, so `manual` is left out and Yahoo's
  close of 231.45 USD is shown: the holding is worth 4,629 USD.
- For the fund, Yahoo answers `404` for an asset with no ticker, and the
  manual value entered last week is shown.
- For a stock they also enter by hand, both answer. The UI shows the manual
  value, first in the configured order, until they pick Yahoo for it.
- To see the AAPL holding in EUR, the UI asks for the USD to EUR rate. With no
  `Conversion` row for the pair, `manual` is left out and Yahoo's rate is
  used. A provider serving only prices answers `404` and is left out too.

## Alternatives considered

- **A facade workflow with a new activity** calling the providers. It keeps
  ADR 0007's rule intact, but adds an activity, and Temporal history for every
  price read, to a read that needs neither.
- **Providers in Go, compiled into the worker.** Simpler for Yahoo, but every
  new source means a change to ivfs, and only in Go.
- **The choice of provider on the asset**, as a nullable column. It would be
  shared by every device and by the MCP server, but it stores a display
  preference in the ledger's database, which was not wanted. It can be added
  later without changing the spec.
- **The gateway stops at the first provider that answers.** Fewer calls, but
  the UI could not offer the others without a second request per asset.
- **Storing fetched values as `AssetValue` rows.** It would give a price
  history for free, but mixes what a person entered with what a site said, and
  makes the manual provider answer for Yahoo.
- **A single value at an instant**, `GET /v1/value?at=`, as first drafted. A
  chart over a year would need a request per day, and a single day is a range
  of one anyway.
- **A step of N days** in place of `interval`. It does weeks but not months,
  and its dates depend on `from`, so two charts of one asset would not line
  up.
- **Every day, always**, with ivfs keeping the dates it needs. A provider
  would fetch, and send, 30 times what a monthly chart shows.
- **Only the days with an observation**, leaving the gaps to ivfs. Simpler for
  a provider, but the first days of a range would have no value whenever it
  starts on a weekend, unless ivfs asked again for the days before it.
- **A `/v1/info` endpoint** for a provider to name itself, or to declare which
  endpoints it implements. The name comes from the configuration, and `404`
  already says an endpoint is missing, so a provider has nothing to say about
  itself yet.
- **A separate spec, and list of providers, for rates.** The same rules for
  days and errors would be written twice, and a source with both, as Yahoo,
  would be configured twice.
- **Rates as the values of a currency asset**, through `/v1/values`. A rate
  has a pair, not identifiers, and needs more decimal places than a price.

## Consequences

- A new source is a service implementing one endpoint or two, in any
  language, and a line in the configuration.
- Every provider is asked for rates as well as values, and one serving only
  one of them spends a `404` on the other.
- The choice of provider is per browser. Another device, or the MCP server,
  sees the default order.
- Valuing the holdings screen costs one request per asset, and each of those
  one call per provider. If that proves slow, a batch endpoint for several
  assets can be added to the spec as optional, with the gateway falling back
  to `GET /v1/values` for providers that do not implement it.
- A provider carries the last value forward over the days without one, so a
  range starting on a Saturday needs it to look back to Friday's close.
- The spec sets no limit on the length of a range. A provider that cannot
  answer one as long answers `400`, and a limit can be added to v1 if one
  is needed.
- Nothing is cached. A provider that is rate limited, as Yahoo can be, caches
  on its own side.
- The asset identifiers of every asset priced, but no holdings or amounts, are
  sent to every configured provider.
- v1 has no authentication: a provider is expected to be reachable only from
  the network ivfs runs on. A provider on the internet would need a token in
  the spec first.
- A value is in the currency the provider reports. The rates make converting
  it possible, but choosing a reporting currency, and adding values into a
  net worth, are out of scope.
- The rates are for display. A transfer between currencies still records the
  rate it happened at, through `RegisterFXConversion`.
