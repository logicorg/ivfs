# ADR 0020: A default preset of currencies, categories, and tags

- Status: Implemented
- Date: 2026-11-15
- Depends on: [ADR 0005](0005-uniqueness-and-idempotent-creation.md)
  (retrieve-or-insert creation), [ADR 0013](0013-users-and-per-user-databases.md)
  (a new user's database is empty), and
  [ADR 0019](0019-currency-neutral-categories.md) (a category has no currency)
- Related: [ADR 0006](0006-intent-level-account-setup.md) (facades),
  [ADR 0007](0007-web-ui-and-http-gateway.md) (the gateway).
  Issue: logicorg/ivfs#1

## Context

A user's first login provisions an empty database (ADR 0013): no currencies,
no categories, no tags. That is deliberate, since the setup is the user's to
decide, but it means that nothing can be recorded until several things have
been created by hand, and a new user has no example of what a reasonable setup
looks like.

Three facts keep this small:

- `InsertCurrency`, `InsertTag`, and `CreateCategory` are retrieve-or-insert
  (ADR 0005). Running any of them twice changes nothing.
- A category has no currency (ADR 0019), so currencies, categories, and tags
  are three independent lists. The only order that matters is a parent
  category before its children.
- Everything a preset creates can already be renamed, closed, or deleted
  through the existing screens.

## Decision

### Opt-in, offered while the database is empty

The Overview page shows a preset card while the user has no currencies, no
categories, and no tags. The card lists the default preset as a form: each
item can be unticked or renamed, and items can be added. "Create" applies what
is on the form. Nothing is written before that.

Whether to show the card is decided by the client, from the three lists
(the Overview loads currencies and categories already, and gains a query for
tags). No dismissal is stored: the card goes away when anything
exists, and a user who wants to start empty creates their first currency by
hand, which hides it. A settings column to remember "no thanks" is not worth
its migration for a card that disappears on first use.

### The preset lives on the server

The default preset is a Go value in
`finance/common/types/presettypes`, returned by `GET /api/presets/default`.
One definition serves every client, and a Go test checks that it passes the
validation its own apply workflow runs.

### `ApplyPreset`

A facade workflow, `finance/workflows/facade/presets.ApplyPreset`, whose input
lives in `presettypes` next to the default:

```go
type ApplyPresetInput struct {
	Currencies []currencytypes.InsertCurrencyParams
	Tags       []tagtypes.InsertTagParams
	Categories []PresetCategory
}

type PresetCategory struct {
	Name     string
	Kind     model.LedgerAccountKind // income or expense; top level only
	Children []PresetCategory
}
```

- It validates the whole input before any write: each currency with its own
  `Validate`, non-empty tag and category names, a kind on every top-level
  category, at most two parent links (ADR 0003), and at most 100 items per
  list (a category counting with its children), which bounds the workflow's
  history. A child carries no kind of its own, and an item given twice in the
  same list, or twice under the same parent, is refused rather than found
  twice.
- It then runs `InsertCurrency`, `InsertTag`, and `CreateCategory` as child
  workflows, one at a time: currencies, tags, then categories parent first,
  each child created with its parent's id and kind.
- It returns the ids it created or found, per list, the categories as a tree
  in the shape of the input.

It adds no activity and no SQL. Sequential is deliberate: the preset is a few
dozen inserts, and a fan-out would buy nothing but a harder history to read.

`POST /api/presets/apply` runs it with the request's body.

### Failure and retries

The workflow is not atomic and does not need to be. If it fails part-way, what
it created stays, and submitting the same form again finds those rows and
creates the rest. For the same reason the server does not require an empty
database: that check would refuse exactly this retry, and applying a preset to
a database that has data only adds what is missing.

A currency whose code is new but whose name is taken fails with
`ErrResourceConflict`, as it does from the currencies screen. The card shows
the problem and the user corrects the form.

A category that exists closed is found like any other, so creating a child
under it fails with `ErrResourceClosed`, and so does every retry, until the
user reopens the category or unticks its children. The card only shows on an
empty database, so only another client can get there; skipping such children,
or checking for closed parents up front, is not worth the code.

### The default preset

Currencies: EUR (Euro, €), USD (US Dollar, $), GBP (British Pound, £), CHF
(Swiss Franc, CHF), JPY (Japanese Yen, ¥, precision 0).

Expense categories:

```text
Housing     Rent or mortgage, Utilities, Maintenance
Food        Groceries, Eating out
Transport   Public transport, Fuel, Vehicle
Leisure     Subscriptions, Travel, Hobbies
Health
Shopping
Taxes
Fees
```

Income categories: Salary, Interest, Dividends, Gifts, Other income.

Tags: vacation, reimbursable, tax-deductible, recurring.

## Alternatives considered

- **Applying the preset at provisioning.** No prompt and no card, but no
  choice either, and `EnsureUser` runs without a user on a worker that cannot
  reach user databases (ADR 0013).
- **Individual API calls from the client.** No new workflow, but a closed tab
  leaves the setup half done with nothing to finish it, and every other client
  would repeat the loop.
- **A flat list of operations**, as the issue first sketches. A child category
  needs its parent's id, which exists only once the parent is created, so the
  input is a tree.
- **Running the inserts concurrently.** See `ApplyPreset`.
- **The preset as a UI constant.** One endpoint fewer, and a preset only the
  web UI knows.
- **A stored dismissal.** See "Opt-in".
- **One transaction for the whole preset.** It would need a new activity that
  repeats the rules of three others, to make atomic something that is safe to
  repeat.

## Consequences

- A new user can go from first login to recording a transaction in one click,
  or ignore the card and build their own setup.
- The preset is code: changing it is a commit, and it is the same for every
  user. Presets per locale, or more than one, would add a parameter to the
  GET, not a new design.
- Out of scope: default accounts and institutions, choosing
  `defaultCurrencyId` from the preset, and an MCP tool to apply one.
- The gateway gains a `presets` area with two endpoints, and the finance
  worker one workflow.
