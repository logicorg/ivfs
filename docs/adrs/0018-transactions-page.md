# ADR 0018: A transactions page: filtered, cursor-paged listing

- Status: Proposed
- Date: 2026-11-07
- Related: [ADR 0003](0003-categories-and-tags.md) (tags deferred),
  [ADR 0007](0007-web-ui-and-http-gateway.md) (one workflow per endpoint,
  filters in typed search params), [ADR 0008](0008-intent-level-transactions.md)
  (income, expense, transfer are intents, not stored kinds),
  [ADR 0015](0015-net-worth-over-time.md) (payload limits).
  Issue: logicorg/ivfs#7

## Context

Users can record income, expense, and transfer, but have no page to browse
them. The wanted page lists transactions and filters them by date range (with
a default), name, tag, category, and flow (expense, transfer, income).

What exists:

- `GET /api/journal-entries` runs `ListJournalEntries`, which filters by one
  ledger account, counterparty, external id, status, and a half-open date
  range, and pages with a keyset cursor on `(occurred_at, id)`. A page is 100
  entries by default and 500 at most, and the response carries each entry's
  postings.
- There is no text search, no category or flow filter, and no tag filter. Tags
  have no assignment table yet (ADR 0003 deferred it), so a tag filter has
  nothing to filter on.
- The flow is not stored. ADR 0008 records an intent as a two-posting entry
  between account kinds: a balance to an expense category is an expense, an
  income category to a balance an income, and two balances a transfer.
- The issue's concern: results approaching Temporal's 2 MB payload limit might
  need an external store such as Redis.

## Decision

### The payload limit is met by paging, not by storage

A page is capped at 500 entries (`MaxJournalEntriesLimit`). With two to four
postings each, a page is on the order of 200 KB, well under the 512 KB warning
and 2 MB limit (ADR 0015). The page never grows with the number of
transactions, so no external store is introduced: Redis would add a second
system, a lifetime for cached results, and a path around the workflow
boundary, to solve a size that cannot occur. The UI loads further pages on
scroll with the cursor (infinite scroll through the generated TanStack Query
hooks); no offset pagination and no total count are offered, since both
degrade or cost a full scan on large ledgers.

### Extend the existing list, do not add a second one

`ListJournalEntriesParams` gains optional filters that are ANDed with the
existing ones, evaluated in SQL inside the same activity so cursor paging
stays correct:

| Filter | Meaning |
| --- | --- |
| `Query` | case-insensitive substring of the description or the counterparty's name |
| `CategoryIDs` | entries with a posting on any of these category ledger accounts (and their descendants) |
| `Flow` | `expense`, `income`, or `transfer`, derived from the posting accounts' kinds as ADR 0008 defines them |
| `TagIDs` | deferred, see below |

The endpoint stays `GET /api/journal-entries`, with the new filters as query
parameters. A new `/api/transactions` list is not added: it would duplicate
the cursor, the posting loading, and the tests for the same rows.

The page is at `/transactions`, and every filter lives in the route's typed
search params (ADR 0007) so a filtered view is linkable. The date range
defaults to the current month in the UI, not in the API, whose zero value
keeps meaning "no bound".

### Tags are a follow-up

A tag filter needs the `journal_entry_tag` table and the tagging workflows
that ADR 0003 deferred. They are a separate decision and piece of work. This
ADR reserves the `TagIDs` filter and keeps it out of the first version of the
page.

## Options considered

1. **Extend `ListJournalEntries` with SQL filters (chosen).** Reuses cursor,
   postings, and the gateway; filters stay consistent with the page boundary.
2. **A new transactions read model**: a table or view with flow, category,
   and description denormalised per entry. Fast to filter and index, but a
   second source of truth to keep in step with deletes and reversals
   (ADR 0011). Revisit if the `EXISTS` filters prove slow.
3. **Fetch everything and filter in the browser.** Fine for hundreds of rows,
   fails at the volumes the issue worries about.
4. **Result in external storage (Redis)**, with only a key in the workflow
   result. Rejected above: paging removes the need.
5. **Offset pagination.** Simple page numbers, but unstable under concurrent
   inserts and slow at depth; keyset is already implemented.
6. **Store the flow on the entry** when it is recorded. Cheap to filter, but
   old entries need a backfill, and ADR 0008 deliberately keeps intents out of
   the ledger.

## Consequences

- `ListJournalEntriesParams` and its activity grow; the existing callers (an
  account's statement) are unaffected because a zero filter is not applied.
- Substring search uses `ILIKE`, which scans. A `pg_trgm` index on
  `journal_entry.description` is the cheap next step if it matters.
- Flow filtering runs a join over postings and accounts per candidate row.
- Category filtering interacts with ADR 0016 (proposed): if categories become
  currency-neutral, `CategoryIDs` should name categories, not ledger accounts.
- The page needs the display fields the list already returns, plus
  counterparty name per row; `Entry` in the API gains it.

## Open questions

1. Should search match postings' account names too, or only description and
   counterparty?
2. Should the default range be the current month, or the last 30 days?
3. Is the tag filter wanted before the tagging feature itself?
