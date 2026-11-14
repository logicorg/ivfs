# ADR 0002: Define the ledger domain model and core application API

- Status: Implemented
- Date: 2026-08-26
- Depends on: [ADR 0001](0001-double-entry-ledger.md)
- Amended by: [ADR 0004](0004-position-subledger.md),
  [ADR 0016](0016-deleting-counterparties.md), and
  [ADR 0017](0017-editing-journal-entries.md)

## Context

ADR 0001 adopts a double-entry ledger.  This ADR turns that decision into a
concrete Go-oriented domain model and a small, general application API.

The backend is intended to support varied clients without becoming a
feature-specific budgeting or portfolio application.  It owns durable
financial facts and the commands and queries necessary to maintain them.  A
client can build budgeting, reporting, portfolio, or import features on top.

## Decision

### Domain structs

The following structs illustrate the persistent model.  Exact JSON tags,
indexes, and Bun relationship tags are implementation details, but the field
ownership and relationships are part of this decision.

```go
type LedgerAccountKind string

const (
	LedgerAccountAsset     LedgerAccountKind = "asset"
	LedgerAccountLiability LedgerAccountKind = "liability"
	LedgerAccountEquity    LedgerAccountKind = "equity"
	LedgerAccountIncome    LedgerAccountKind = "income"
	LedgerAccountExpense   LedgerAccountKind = "expense"
)

type PostingSide string

const (
	PostingDebit  PostingSide = "debit"
	PostingCredit PostingSide = "credit"
)

type JournalEntryStatus string

const (
	JournalEntryDraft  JournalEntryStatus = "draft"
	JournalEntryPosted JournalEntryStatus = "posted"
	JournalEntryVoided JournalEntryStatus = "voided" // drafts only
)

// FinancialAccount identifies a real provider-side account relationship, such
// as a Revolut multi-currency account or an IBKR brokerage account. It is not
// itself a general ledger category or a single balance.
type FinancialAccount struct {
	ID            int64
	InstitutionID int64
	Name          string
	ExternalID    *string
	OpenedAt      *time.Time
	ClosedAt      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// LedgerAccount is a chart-of-accounts bucket. A financial account can have
// more than one ledger account, for example Revolut EUR and Revolut USD.
// Every monetary ledger account tracks one currency.
type LedgerAccount struct {
	ID                 int64
	ParentID           *int64
	FinancialAccountID *int64
	Name               string
	Kind               LedgerAccountKind
	CurrencyID         int64
	ClosedAt           *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Counterparty struct {
	ID        int64
	Name      string
	ExternalID *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// JournalEntry is the header for one financial event. A posted entry can be
// edited or deleted, and stays balanced through every change (ADR 0017).
type JournalEntry struct {
	ID                   int64
	CounterpartyID       *int64
	OccurredAt           time.Time
	Description          *string
	ExternalID           *string
	IdempotencyKey       *string
	Metadata             map[string]string
	Status               JournalEntryStatus
	PostedAt             *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Postings             []Posting
	PositionMovements    []PositionMovement
}

// Amount is positive and expressed in the currency of LedgerAccountID.
type Posting struct {
	ID              int64
	JournalEntryID  int64
	LineNumber      int
	LedgerAccountID int64
	Amount          decimal.Decimal
	Side            PostingSide
	CreatedAt       time.Time
}

// PositionMovement is the source of truth for units owned. UnitsDelta and
// CostBasisBaseAmount are signed: purchases are normally positive and sales
// normally negative. Monetary purchases and sales must reference their entry.
type PositionMovement struct {
	ID                  int64
	JournalEntryID      *int64
	FinancialAccountID  int64
	AssetID             int64
	UnitsDelta          decimal.Decimal
	CostBasisBaseAmount *decimal.Decimal
	OccurredAt          time.Time
	ExternalID          *string
	CreatedAt           time.Time
}
```

`FinancialAccount` replaces the current institution-held `Account` naming. It
identifies the provider-side account relationship and may group balances in
many currencies. `LedgerAccount` replaces both the directional account fields
and the separate expense/income category concept.  It is the smallest
balance-bearing unit, holds `CurrencyID`, and is linked to a financial account
only when it represents a real institution-held balance.

The category and tag policy, including the bounded `ParentID` hierarchy and
the client-facing category workflow facade, is defined in [ADR 0003](0003-categories-and-tags.md).

`Counterparty` is optional entry metadata.  It is not a polymorphic account
reference and is not used to calculate balances.

`Asset` and `Currency` remain independent reference data.  An asset's current
unit position is derived by summing `PositionMovement.UnitsDelta`, grouped by
`FinancialAccountID` and `AssetID`.

### Persistence and invariants

The database must enforce the following in addition to the field-level
constraints in ADR 0001:

- `Posting.LineNumber` is unique within one journal entry.
- Every posted entry has two or more postings.
- For every currency in a posted entry, debit and credit amounts total to the
  same value.
- A posted entry, its postings, and its linked monetary position movements
  can be changed or deleted (ADR 0011, ADR 0017). The database checks the first
  two invariants on every write of a posted entry or of one of its postings.
- A posted entry is corrected by editing or deleting it (ADR 0017).
- A monetary purchase or sale position movement references the journal entry
  containing its monetary postings.
- `JournalEntry.ExternalID` is unique within its import source once import
  sources are introduced. It is the idempotency key for imported events.

FX entries use single-currency FX-clearing ledger accounts. Optional entry
metadata holds the provider reference and exact quoted or effective rate as a
decimal string. A fee is a normal posting in the currency in which it was
charged; it is never silently absorbed into the exchange rate or a rounding
difference.

The write that posts an entry must insert the entry header as posted, insert
its postings, and validate all invariants in one SQL database transaction
(ADR 0017).  A trade write must include both the posted
entry and its position movement in that same transaction.

### Activities

Activities are the database boundary.  Any activity that changes financial
state owns its SQL transaction and is independently idempotent.  In
particular, workflows must never call one activity to insert an entry and a
second activity to insert its postings or position movements.

The backend should provide these activity groups.

| Activity group | Required operations | Notes |
| --- | --- | --- |
| Reference data | Create, retrieve, list, update, and archive `Currency`, `Institution`, and `Asset` | Assets are instruments; prices and analytics remain outside this core. |
| Financial accounts | Create, retrieve, list, update metadata, close `FinancialAccount` | Creation may create its first requested single-currency cash ledger account in the same transaction. No delete after use. |
| Chart of accounts | Create, retrieve, list tree, rename, and close `LedgerAccount` | Kinds and financial-account links are immutable after postings exist. |
| Counterparties | Create, retrieve, list/search, rename, delete `Counterparty` | Counterparties may be created from imports. |
| Journal entries | Create draft, retrieve, list/filter, update draft, post, void draft | `Post` validates balance. |
| Ledger queries | List postings for an account, calculate account balance at a time, trial balance, list entries by counterparty/external ID | Queries include only posted entries by default. |
| Positions | Record a non-monetary position movement, register an investment trade, list movements, calculate units held at a time | `RegisterInvestmentTrade` creates the entry, postings, and position movement atomically. |

An activity should accept a command DTO, rather than a persistence model.  The
central command is deliberately generic:

```go
type PostingInput struct {
	LedgerAccountID int64
	Amount          decimal.Decimal
	Side            PostingSide
}

type PostJournalEntryInput struct {
	IdempotencyKey  string
	OccurredAt      time.Time
	Description     *string
	CounterpartyID  *int64
	ExternalID      *string
	Metadata        map[string]string
	Postings        []PostingInput
}
```

The API does not need separate persistence operations for “expense”,
“income”, “transfer”, or “foreign exchange”: all are a balanced
`PostJournalEntryInput`.  This is what keeps the backend useful to different
applications without encoding their features.

`RegisterInvestmentTrade` is the one domain-specific financial command.  It
accepts the same entry input plus an asset, financial account, unit delta, and
cost-basis data, and commits all resulting rows together.

### Workflows

Workflows are the durable application-facing orchestration layer.  They
perform authorization and reference checks, attach consistent activity retry
and idempotency policies, and call atomic write activities.  They do not
derive balances or perform partial financial writes themselves.

The initial workflow surface is:

| Workflow | Calls / outcome |
| --- | --- |
| `CreateFinancialAccount` | Creates a financial account and, when requested, its first single-currency cash ledger account atomically. |
| `CloseFinancialAccount` | Validates closure policy and closes the account. |
| `CreateLedgerAccount` / `CloseLedgerAccount` | Maintains the chart of accounts. |
| `CreateCounterparty` / `DeleteCounterparty` | Maintains optional entry metadata. |
| `CreateDraftJournalEntry` | Creates an editable, unposted entry. |
| `PostJournalEntry` | Calls the atomic post activity for a balanced generic entry. |
| `VoidDraftJournalEntry` | Discards an unposted entry without changing balances. |
| `RegisterInvestmentTrade` | Records a buy or sell entry and the linked position movement atomically. |
| `RegisterPositionMovement` | Records a non-monetary position movement, such as a split or share transfer, under explicit policy. |
| `AccountBalanceAt` / `TrialBalanceAt` | Read-only ledger projections at a specified point in time. |
| `PositionAt` | Read-only sum of units held for an asset and financial account at a specified point in time. |

Simple retrieval/list operations may be exposed directly as read activities or
through read-only workflows, depending on the transport API.  They do not
need long-running orchestration.

### Deliberately out of scope

This core does not implement:

- budgeting rules, goals, recurring-transaction generation, or dashboard UI;
- market-data ingestion, valuation snapshots, performance analytics, or tax
  lot selection;
- bank or broker integrations, although their imported records can later use
  external IDs and idempotency keys; or
- specialised transfer, expense, income, or foreign-exchange tables.

Those features consume the ledger's stable facts or submit generic ledger
commands.  They do not replace the ledger as the source of truth.

## Consequences

- The application has one expressive monetary write path, plus the necessary
  investment trade extension for units.
- Clients can add new experiences without changing the accounting schema for
  each transaction type.
- Financial consistency is maintained inside database transactions, even when
  Temporal retries workflows.
- Each concrete balance is denominated in one currency; foreign exchange is
  explicit through FX-clearing postings and optional journal-entry metadata.
- The existing activity and workflow packages will be substantially replaced
  during implementation.
