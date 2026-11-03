# RFC 0002: Reading the balances of every account at once

- Status: Under discussion
- Date: 2026-10-03
- Related: [ADR 0007](../adrs/0007-web-ui-and-http-gateway.md) (one workflow per endpoint)

## Problem

The UI's accounts screen shows every financial account with its balances. The
API has no read for that, so the screen makes one request per account:

```text
GET /api/financial-accounts                   ListFinancialAccounts
GET /api/currencies                           ListCurrencies, for each currency's precision
GET /api/financial-accounts/{id}/balances     FinancialAccountBalances, once per account
```

With N financial accounts, each with L ledger accounts, one visit to the
screen costs:

- **N + 2 workflow executions.** Each endpoint runs a workflow (ADR 0007), and
  each workflow execution is recorded in Temporal's history, so a read that
  looks free in the UI is a write on the Temporal side.
- **About N × (2 + 2L) SQL queries.** The `FinancialAccountBalances` activity
  loads the account and its ledger accounts, then calls `LedgerAccountBalance`
  for each one, which loads the ledger account with its currency and sums its
  postings in a query of its own.
- **N HTTP requests from the browser**, which the browser runs a few at a time.

For the handful of accounts a person has today, this is fast enough. With a
few dozen, the screen fills in slowly and each visit leaves dozens of workflow
histories behind. The balances are also read at slightly different moments,
since every account is a separate query in a separate activity, so an entry
posted between two of them shows in one balance and not in the other.

## Option A: keep a request per account

Leave the API as it is.

- **For:** nothing to build; each card loads and fails on its own; the
  existing endpoint stays the only balances read.
- **Against:** the costs above grow with the number of accounts; no single
  moment the balances are read at.

## Option B: a workflow that reads every account's balances

A new activity and workflow, `ListFinancialAccountBalances`, return each
financial account with the balances of its ledger accounts, in one query that
sums the postings grouped by ledger account and joins the financial account,
its institution, and the currency:

```text
ListFinancialAccountBalancesParams   At (zero means now), IncludeClosed
ListFinancialAccountBalancesResult   FinancialAccounts: [{ID, Name, InstitutionID,
                                     InstitutionName, Closed, Balances: [...]}]
```

The balances keep the natural direction `LedgerAccountBalance` gives them
today. The API exposes it as `GET /api/financial-account-balances?at=&includeClosed=`,
and the screen makes one request instead of N + 1.

- **For:** one workflow execution and one query per visit, whatever the number
  of accounts; every balance is read at the same moment; the per-account
  endpoint can stay for a screen that shows one account.
- **Against:** a new activity, workflow, and endpoint to test; the query that
  computes a natural-direction balance now lives in two places, unless
  `LedgerAccountBalance` and `FinancialAccountBalances` are rewritten on top of
  the new one.

## Option C: let `FinancialAccountBalances` take several accounts

`FinancialAccountBalancesParams` takes a list of IDs instead of one, and the
activity reads them all in one query.

- **For:** no new workflow; one execution per visit.
- **Against:** the screen still needs the account list first, so it is two
  sequential requests; a missing ID among many needs a rule (fail the whole
  read, or skip it); the endpoint's path no longer names one account.

## Option D: build it in the gateway

The endpoint runs `ListFinancialAccounts` and `TrialBalanceAt`, which already
sums every ledger account in one query, and joins them.

- **Rejected.** It breaks ADR 0007's rule that an endpoint runs one workflow
  and keeps no logic of its own: the gateway would apply the natural direction
  and match ledger accounts to financial accounts, which the workflow tests
  would not cover.

## Open questions

- Is the number of accounts expected to grow enough to justify B or C now, or
  is A fine until the screen is visibly slow?
- If B: should `LedgerAccountBalance` and `FinancialAccountBalances` be rebuilt
  on the new query, so the balance arithmetic exists once?
- Should the overview also carry totals per currency, or per kind (assets
  minus liabilities)? That is the start of the reporting queries the UI will
  need for charts, and may deserve a reporting read of its own rather than
  being folded in here.
- More generally: should every read run as a workflow? ADR 0007 decided yes,
  for one backend and one place for the rules. The cost above, a workflow
  history per read, applies to every screen, and is worth measuring before
  more read-heavy screens are built.
