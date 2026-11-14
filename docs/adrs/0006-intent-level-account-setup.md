# ADR 0006: Intent-level account setup

- Status: Implemented
- Date: 2026-10-03
- Depends on: [ADR 0002](0002-ledger-domain-and-application-api.md), [ADR 0003](0003-categories-and-tags.md), [ADR 0004](0004-position-subledger.md), and [ADR 0005](0005-uniqueness-and-idempotent-creation.md)
- Amends: ADR 0003 (where the category facade lives and how it is built) and ADR 0005 (a create that matches a closed row)
- Related: [RFC 0001](../rfcs/0001-financial-account-identifiers.md) (recording account identifiers)

## Context

Setting up an account today takes several low-level commands in the right
order: find or create the institution, create the financial account, then
create each ledger account it needs, plus the equity and clearing accounts
that opening balances and currency conversions post to. Each command is
idempotent (ADR 0005), but the order, the names, and the kinds are left to the
caller.

Whoever calls them, a person filling in a form or a language model, has to
reinvent the structure every time. One session creates "Cash", another "Revolut
EUR", a third "Checking". Those choices are hard to undo: names are reserved
forever, even when closed (ADR 0005), and a ledger account with postings cannot
change its kind or currency. So the structure of an account is decided once, in
code, and the caller asks for an account in the terms a person uses: "open a
Revolut EUR account".

Accounts are opened rarely, a handful in the life of the system, so the aim is
consistency, not saving round trips.

## Decision

### The facade layer

A facade workflow presents what a person means to do, such as "open an
account", and composes the lower-level workflows that do it. Facades live
under `finance/workflows/facade`, one package per area, so the path tells a
facade from a workflow that wraps a single activity.

A facade calls the lower-level workflows as child workflows, not their
activities, so it is plainly a composition: each step keeps the activity
options, validation, and defaults of its own workflow, and the facade adds
only the order of the steps and the rules between them. A facade calls an
activity directly only where no workflow exists for that step, and adding the
workflow is preferred.

A child workflow's error reaches the facade with its cause chain intact, but
the first `temporal.ApplicationError` in the chain is the child's own
`fmt.Errorf` wrapper, not the activity's structured error. A facade that
branches on an error type, such as `ErrNotFound`, therefore walks the whole
chain instead of using a single `errors.As`.

The category facade (ADR 0003) moves to `finance/workflows/facade/categories`
and is rebuilt the same way.

### Two workflows

A new facade package, `finance/workflows/facade/accountsetup`, has two
workflows:

- `OpenAccount` finds or creates a financial account, with the balances its
  type needs in each currency it is given, and posts their opening balances.
  The same call adds currencies to an account that already exists.
- `AddBalance` adds one named balance to an existing financial account, such
  as a savings vault, and posts its opening balance.

Every step is an existing workflow: `RetrieveCurrency`, `RetrieveInstitution`,
`InsertInstitution`, `CreateFinancialAccount`, `RetrieveFinancialAccount`,
`CreateLedgerAccount`, and `PostJournalEntry`. The facades add no persistence of
their own. They are transport-independent; which of them a UI or MCP client
exposes is decided with the client-facing surface, not here.

CLAUDE.md allows a workflow at most one atomic activity per financial write.
Creating the institution, the financial account, and the ledger accounts
writes reference data, not financial facts, and each step is a workflow
around one atomic, idempotent activity, so their order only decides what exists
if the facade stops halfway. A financial account without its balances, or a
balance without postings, is a valid state that a retry completes. Each opening
balance is one financial write in one activity, through `PostJournalEntry`. The
rule holds.

They are Temporal workflows rather than a sequence of calls in a client, so
they finish even if the client disconnects, resume after a worker restart, and
serve every transport unchanged. The low-level commands stay, for what these
two do not cover.

### Account types

The type says what the account holds, not who the institution is:

| Type | Holds | Examples | Balances per currency, under the financial account |
| --- | --- | --- | --- |
| `bank` | money | current and savings accounts, Revolut, a debit or prepaid card, a pension tracked only as a value | "Cash" (asset) |
| `card` | a debt | a credit or charge card | "Card balance" (liability) |
| `brokerage` | units of assets, and money | a broker, a crypto exchange, an employer stock plan, a pension invested in funds | "Cash" (asset) and "Investments" (asset) |
| `cash` | physical money | a wallet | "Cash" (asset) |

A debit or prepaid card spends money the person holds, so it is a `bank`
account, not a `card`. A brokerage holds money in "Cash" and the cost of its
holdings in "Investments", as in example 5 of the ledger examples; the units
themselves are position movements on the financial account (ADR 0004), so
holdings bought in USD need USD among its currencies.

The type is given on every call and not stored. Balance names do not repeat
the currency, since a ledger account's key already includes it (ADR 0005), and
balances have no parent: the financial account groups them.

### Shared accounts per currency

Every currency either workflow touches also gets two top-level ledger accounts,
which belong to no financial account and are shared by every account in that
currency:

| Name | Kind | Used by |
| --- | --- | --- |
| "Opening balances" | equity | opening balances, here and for holdings (ADR 0004) |
| "FX clearing" | asset | `RegisterFXConversion`, one per currency |

They are created by retrieve-or-insert, so the first account in a currency
creates them and later ones find them.

### Opening an account: the form

`OpenAccount` takes the answers to a short form. Everything else is derived.

| Question | Field | Required |
| --- | --- | --- |
| What type of account is it? | `type` | yes |
| Which institution? | `institutionName` | yes, except for `cash` |
| Create the institution if it does not exist? | `createInstitution` | no; see below |
| What is the account called? | `accountName` | no; defaults below |
| Which currencies does it hold? | `currencies[].code` | yes, at least one |
| What is the balance in each today? | `currencies[].openingBalance` | no |
| As of when? | `openingBalancesAt` | no; defaults to the time of the request |

Not asked, because they are derived:

- the institution's kind: "bank" for `bank`, "card issuer" for `card`,
  "broker" for `brokerage`, and "cash" for `cash`;
- the balances' names and kinds, from the type;
- the shared accounts.

Not asked yet, and left for later:

- the account's details, such as its IBAN (RFC 0001);
- the date the account was opened, which `FinancialAccount.OpenedAt` could
  hold but `CreateFinancialAccount` does not accept yet. It is not the opening
  balances' date: that is when tracking starts, often long after the account
  was opened.

### The institution

The institution is looked up by name. If it exists, it is used. If it does not,
`OpenAccount` writes nothing and returns the outcome `institution_not_found`,
unless `createInstitution` is set, in which case it creates it. A misspelt name
therefore never silently becomes a new institution, whose name would be
reserved forever: the caller confirms with the person, then sends the same
request again with `createInstitution` set.

A `cash` account belongs to the institution "Physical cash", which the caller
never names. It is created on first use without asking, since its name is a
constant in code and cannot be misspelt, and found afterwards by
retrieve-or-insert. A real institution of the same name would be the same row,
which is why the name is unlikely to be one.

MCP elicitation, where a server asks the person directly in the middle of a
tool call, was considered for the confirmation. It cannot happen inside a
running workflow, which has no channel back to the client, so it could at most
wrap the same contract in an MCP handler. The contract itself works with a
form, an MCP client, or any other transport.

### The financial account

`accountName` defaults to the institution's name, or to "Wallet" for `cash`.
The financial account is then found or created by that name, as
`CreateFinancialAccount` does (ADR 0005). A name that does not exist yet is a
new financial account; opening an account again with a name that exists, with
a currency it does not hold yet, adds that currency's balances to it; repeating
a request changes nothing.

The type is not stored, so it cannot be checked against the account's
existing balances: opening an existing brokerage account again as a `bank`
adds only "Cash" in the new currency. Getting the type right is the caller's
part of the form.

### Adding a balance

`AddBalance` takes:

| Question | Field | Required |
| --- | --- | --- |
| Which account? | `financialAccountId` | yes, from `ListAccounts` |
| What is the balance called? | `name` | yes, such as "Savings vault" |
| Which currency? | `currencyCode` | yes |
| What is in it today? | `openingBalance` | no |
| As of when? | `openingBalanceAt` | no; defaults to the time of the request |

The balance is an asset. It creates the shared accounts for its currency and
posts its opening balance like `OpenAccount`. A new currency for an account
goes through `OpenAccount`, which creates every balance the type needs;
`AddBalance` is for the extra balances a type does not create.

### Closed institutions and accounts

If the institution or the financial account a request matches is closed,
both workflows fail with `ErrResourceClosed` before any write. ADR 0005 has a
create that matches a closed row return it unchanged, and leaves reporting it
for a later revision. For these workflows that revision is made now: the caller
asked to use the account, and a closed one cannot be used, so returning it
would only move the failure to the first posting. The low-level creates keep
ADR 0005's behaviour.

### Opening balances

An opening balance is posted as one journal entry between the balance and
"Opening balances" in its currency, at the given date:

| Balance | A positive opening balance |
| --- | --- |
| "Cash", or a balance added with `AddBalance` | debits the balance, credits "Opening balances" |
| "Card balance" | credits "Card balance", debits "Opening balances": the amount owed |

The amount is in the balance's natural direction, so a negative one, such as an
overdraft, swaps the two sides, and zero posts nothing. A brokerage's opening
balance is its cash, posted to "Cash" and never to "Investments". Its holdings
are each a `RegisterInvestmentTrade` against "Opening balances", as ADR 0004
decides.

Each opening balance is posted with an idempotency key derived from its ledger
account, `opening-balance:<ledger account id>`, rather than the random key
`PostJournalEntry` defaults to. A retry, or the same request sent twice, finds
the entry instead of posting a second one, so a balance has at most one
opening balance. A different amount sent later is not reconciled, as with any
retrieve-or-insert; a wrong opening balance is corrected by editing or
deleting its entry (ADR 0017).

### Results

Both workflows return an outcome and every id and name the caller needs
afterwards:

| Outcome | Meaning | Writes |
| --- | --- | --- |
| `opened` | The account exists with its balances, created now or before. | as needed |
| `institution_not_found` | The institution does not exist and `createInstitution` was not set. Only `OpenAccount` returns it. | none |

`institution_not_found` is a normal result, not an error: it asks the caller a
question. An `opened` result lists the institution, the financial account, its
balances and the shared accounts in the flat form `ListAccounts` returns, and
the id of each opening balance's entry.

### Not covered

These stay with the low-level commands until there is a need to raise them to
the same level:

- loans and mortgages;
- top-level accounts that are not categories, such as money a friend owes;
- balances nested under another balance;
- categories, which `CreateCategory` already covers at this level;
- recording an account's details (RFC 0001) and its opening date.

## Examples

Opening the first Revolut account. The institution is new, so the first request
asks back:

```text
OpenAccount{type: bank, institutionName: "Revolut", accountName: "Revolut Personal",
            currencies: [{code: EUR, openingBalance: 1200}]}
  -> institution_not_found, nothing written

OpenAccount{... the same, createInstitution: true}
  -> opened: Revolut / Revolut Personal / Cash (EUR), 1200 posted against
     Opening balances (EUR); Opening balances (EUR) and FX clearing (EUR) created
```

A brokerage with two currencies, which gets a third later through the same
workflow:

```text
OpenAccount{type: brokerage, institutionName: "Interactive Brokers", accountName: "IBKR",
            currencies: [{code: EUR, openingBalance: 5000}, {code: USD}], createInstitution: true}
  -> opened: IBKR / Cash and Investments in EUR and in USD

OpenAccount{type: brokerage, institutionName: "Interactive Brokers", accountName: "IBKR",
            currencies: [{code: GBP}]}
  -> opened: IBKR / Cash and Investments in GBP; EUR and USD unchanged
```

A savings vault inside the Revolut account, and a wallet:

```text
AddBalance{financialAccountId: <Revolut Personal>, name: "Savings vault",
           currencyCode: EUR, openingBalance: 500}
  -> opened: Revolut Personal / Savings vault (EUR), 500 posted

OpenAccount{type: cash, currencies: [{code: EUR, openingBalance: 80}]}
  -> opened: Physical cash / Wallet / Cash (EUR); Physical cash created on first use
```

Sending any of these requests a second time returns the same ids and posts
nothing new.

## Alternatives considered

- **Composition in the client.** It needs no workflow, but a client that
  disconnects leaves a half-built setup that nothing finishes, and every
  transport would repeat it.
- **Facades that call activities directly.** It saves a workflow execution
  per step, but hides the composition: a facade would repeat each step's
  activity options and defaults, and could drift from the workflow that owns
  them.
- **A new activity that opens the account in one transaction.** It would make
  the whole setup atomic, but duplicate the rules the existing creates already
  own, for a gain the idempotent steps make unnecessary.
- **Storing the type on the financial account.** It would let a request leave
  the type out and catch a type that contradicts the account, at the cost of a
  column that a low-level create could contradict in turn. The type is the
  first answer on the form, so asking it each time is cheap.
- **Reusing an institution's only account when no name is given, and asking
  when there are several.** It saves typing a name, but needs a list read and a
  third outcome, and guesses when an institution has several accounts. A
  default name is a plain retrieve-or-insert.
- **Creating the institution without asking.** Simplest, but a typo becomes a
  name reserved forever.
- **An "FX fees" expense category with the shared accounts.** Left to the
  person, since not everyone pays conversion fees and the category is easy to
  create when needed.
- **Accepting holdings in the opening balance.** It would duplicate
  `RegisterInvestmentTrade` without its checks on units and cost basis.

## Consequences

- An account is opened with one request, and every account of a type has the
  same structure and names.
- Opening an account can stop and ask instead of failing, through an outcome
  that is not an error.
- Each step of a facade is a child workflow, with its own history, shown under
  the facade in the Temporal UI. Facade tests mock child workflows instead of
  activities.
- Telling an error type through a child workflow needs a helper that walks the
  cause chain, recorded in `docs/gotchas.md`.
- The opening balance relies on `PostJournalEntry` returning the existing entry
  for a known idempotency key, which it already does.
- `docs/onboarding/entity_relationships.md` names its balances "EUR cash" and
  its clearing account "FX clearing EUR". It is updated to the names above.

## Amendments to earlier ADRs

ADR 0003:

- The category facade lives in `finance/workflows/facade/categories` and calls
  the ledger account workflows as child workflows rather than their activities.

ADR 0005:

- `OpenAccount` and `AddBalance` fail with `ErrResourceClosed` when the
  institution or the financial account they match is closed. The low-level
  creates still return a closed row unchanged.
