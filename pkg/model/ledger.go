package model

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// LedgerAccountKind describes the accounting nature of a ledger account.
type LedgerAccountKind string

const (
	LedgerAccountAsset     LedgerAccountKind = "asset"
	LedgerAccountLiability LedgerAccountKind = "liability"
	LedgerAccountEquity    LedgerAccountKind = "equity"
	LedgerAccountIncome    LedgerAccountKind = "income"
	LedgerAccountExpense   LedgerAccountKind = "expense"
)

// ValidLedgerAccountKind returns true if the given kind is valid.
func ValidLedgerAccountKind(kind LedgerAccountKind) bool {
	switch kind {
	case LedgerAccountAsset, LedgerAccountLiability, LedgerAccountEquity, LedgerAccountIncome, LedgerAccountExpense:
		return true
	default:
		return false
	}
}

// PostingSide is the side of the double-entry ledger affected by a posting.
// In double-entry accounting, debits and credits are used to record changes in accounts.
// A debit entry increases asset or expense accounts and decreases liability, equity, or revenue accounts.
// Conversely, a credit entry decreases asset or expense accounts and increases liability, equity, or revenue accounts.
type PostingSide string

const (
	// PostingDebit is the debit side of a posting.
	PostingDebit PostingSide = "debit"
	// PostingCredit is the credit side of a posting.
	PostingCredit PostingSide = "credit"
)

// Opposite returns the opposite of a PostingSide.
// if the receiver side is a debit it will return credit, and vice versa.
func (s PostingSide) Opposite() PostingSide {
	if s == PostingDebit {
		return PostingCredit
	}
	return PostingDebit
}

// IsValidPostingSide returns true if the given posting side is valid.
func IsValidPostingSide(s PostingSide) bool {
	return s == PostingDebit || s == PostingCredit
}

// JournalEntryStatus controls the lifecycle of a journal entry.
type JournalEntryStatus string

const (
	JournalEntryDraft  JournalEntryStatus = "draft"
	JournalEntryPosted JournalEntryStatus = "posted"
)

// IsValidJournalEntryStatus returns true if the given status is valid.
func IsValidJournalEntryStatus(s JournalEntryStatus) bool {
	switch s {
	case JournalEntryDraft, JournalEntryPosted:
		return true
	default:
		return false
	}
}

// FinancialAccount identifies an account relationship held with an
// institution. It is not itself a balance or a currency-denominated ledger
// account; one financial account may group several ledger accounts.
type FinancialAccount struct {
	bun.BaseModel `bun:"table:account.financial_account,alias:fa"`

	ID            int64  `bun:"id,pk,autoincrement" json:"id"`
	InstitutionID int64  `bun:"institution_id,notnull" json:"institutionId"`
	Name          string `bun:"name,notnull" json:"name"`
	// ExternalID is the provider's identifier of the real account, such as an
	// IBAN or a broker account number. It is optional, compared exactly, and
	// unique within the institution when present: two accounts sharing one are
	// the same real account (ADR 0005).
	ExternalID *string    `bun:"external_id" json:"externalId,omitempty"`
	OpenedAt   *time.Time `bun:"opened_at,nullzero" json:"openedAt,omitempty"`
	ClosedAt   *time.Time `bun:"closed_at,nullzero" json:"closedAt,omitempty"`
	CreatedAt  time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt  time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	Institution    *Institution    `bun:"rel:belongs-to,join:institution_id=id,on_delete:restrict" json:"institution,omitempty"`
	LedgerAccounts []LedgerAccount `bun:"rel:has-many,join:id=financial_account_id" json:"ledgerAccounts,omitempty"`
}

// LedgerAccount is the smallest balance-bearing account. Every monetary
// ledger account is denominated in exactly one currency.
type LedgerAccount struct {
	bun.BaseModel `bun:"table:ledger.ledger_account,alias:la"`

	ID int64 `bun:"id,pk,autoincrement" json:"id"`
	// ParentID is the parent ledger account, if any. It is used to create a hierarchy of ledger accounts.
	ParentID *int64 `bun:"parent_id" json:"parentId,omitempty"`
	// FinancialAccountID is the financial account that owns this ledger account, if any.
	//
	// Examples:
	//
	//  * a bank account may have a financial account with ledger accounts for cash, interest, and fees.
	FinancialAccountID *int64            `bun:"financial_account_id" json:"financialAccountId,omitempty"`
	CurrencyID         int64             `bun:"currency_id,notnull" json:"currencyId"`
	Name               string            `bun:"name,notnull" json:"name"`
	Kind               LedgerAccountKind `bun:"kind,notnull" json:"kind"`
	ClosedAt           *time.Time        `bun:"closed_at,nullzero" json:"closedAt,omitempty"`
	CreatedAt          time.Time         `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt          time.Time         `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	Parent           *LedgerAccount    `bun:"rel:belongs-to,join:parent_id=id,on_delete:restrict" json:"parent,omitempty"`
	Children         []LedgerAccount   `bun:"rel:has-many,join:id=parent_id" json:"children,omitempty"`
	FinancialAccount *FinancialAccount `bun:"rel:belongs-to,join:financial_account_id=id,on_delete:restrict" json:"financialAccount,omitempty"`
	Currency         *Currency         `bun:"rel:belongs-to,join:currency_id=id,on_delete:restrict" json:"currency,omitempty"`
}

// The names ADR 0006 gives a brokerage's two balances in each currency.
const (
	BrokerageCashBalance        = "Cash"
	BrokerageInvestmentsBalance = "Investments"
)

// IsBrokerageInvestments reports whether the account is the "Investments"
// balance of a financial account. Its balance is cash put in less cash taken
// out (ADR 0009), neither cost nor value, so net worth leaves it out. The rule
// is the name and the kind, as the trades find it; WhereNotBrokerageInvestments
// is the same rule in SQL.
func (la *LedgerAccount) IsBrokerageInvestments() bool {
	return la.Kind == LedgerAccountAsset && la.FinancialAccountID != nil && strings.EqualFold(la.Name, BrokerageInvestmentsBalance)
}

// WhereNotBrokerageInvestments leaves out the ledger accounts, aliased la,
// that IsBrokerageInvestments matches.
func WhereNotBrokerageInvestments(q *bun.SelectQuery) *bun.SelectQuery {
	return q.Where("NOT (la.kind = ? AND la.financial_account_id IS NOT NULL AND lower(la.name) = lower(?))",
		LedgerAccountAsset, BrokerageInvestmentsBalance)
}

// Counterparty is optional descriptive metadata such as a merchant, employer,
// or person. It is not a substitute for a ledger account. Its name is unique
// compared case-insensitively, through an index on lower(name) rather than a
// column constraint (ADR 0005).
type Counterparty struct {
	bun.BaseModel `bun:"table:ledger.counterparty,alias:cp"`

	ID   int64  `bun:"id,pk,autoincrement" json:"id"`
	Name string `bun:"name,notnull" json:"name"`
	// ExternalID is how an external source identifies the counterparty, such as a
	// merchant ID in a bank export or an employer's payroll reference. It is
	// descriptive only and unconstrained until imports give it a meaning; the
	// counterparty is identified by its name (ADR 0005).
	ExternalID *string   `bun:"external_id" json:"externalId,omitempty"`
	CreatedAt  time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt  time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
}

// JournalEntry is the header for one financial event. Posted entries are
// immutable; corrections are represented by a separate reversing entry.
type JournalEntry struct {
	bun.BaseModel `bun:"table:ledger.journal_entry,alias:je"`

	ID int64 `bun:"id,pk,autoincrement" json:"id"`
	// CounterpartyID is the ID of the counterparty, if any.
	CounterpartyID *int64 `bun:"counterparty_id" json:"counterpartyId,omitempty"`
	// ReversalOfEntryID is the ID of the journal entry that this entry reverses, if any.
	ReversalOfEntryID *int64 `bun:"reversal_of_entry_id" json:"reversalOfEntryId,omitempty"`
	// OccurredAt is the time when the event occurred, as opposed to when it was recorded in the ledger.
	OccurredAt  time.Time `bun:"occurred_at,nullzero,notnull" json:"occurredAt"`
	Description *string   `bun:"description" json:"description,omitempty"`
	// ExternalID is the identifier of the source record the entry was booked
	// from, such as a bank transaction ID, a broker trade ID, or a payslip
	// reference, so the entry can be traced back and matched during
	// reconciliation. It is not the IdempotencyKey, which identifies the request
	// that wrote the entry. Once import sources exist it is unique within its
	// source and becomes the idempotency key for imported events (ADR 0002).
	ExternalID     *string            `bun:"external_id" json:"externalId,omitempty"`
	IdempotencyKey *string            `bun:"idempotency_key,unique" json:"idempotencyKey,omitempty"`
	Metadata       map[string]string  `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	Status         JournalEntryStatus `bun:"status,notnull,default:'draft'" json:"status"`
	PostedAt       *time.Time         `bun:"posted_at,nullzero" json:"postedAt,omitempty"`
	// CreatedAt is the time when the journal entry was created in the ledger.
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	// UpdatedAt is the time when the journal entry was last updated in the ledger.
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	// Relationships

	Counterparty      *Counterparty      `bun:"rel:belongs-to,join:counterparty_id=id,on_delete:restrict" json:"counterparty,omitempty"`
	ReversalOfEntry   *JournalEntry      `bun:"rel:belongs-to,join:reversal_of_entry_id=id,on_delete:restrict" json:"reversalOfEntry,omitempty"`
	Postings          []Posting          `bun:"rel:has-many,join:id=journal_entry_id" json:"postings,omitempty"`
	PositionMovements []PositionMovement `bun:"rel:has-many,join:id=journal_entry_id" json:"positionMovements,omitempty"`
}

// Posting is one immutable accounting line. Its currency is determined by
// LedgerAccount currency. Amounts are always positive.
type Posting struct {
	bun.BaseModel `bun:"table:ledger.posting,alias:p"`

	ID              int64           `bun:"id,pk,autoincrement" json:"id"`
	JournalEntryID  int64           `bun:"journal_entry_id,notnull" json:"journalEntryId"`
	LineNumber      int             `bun:"line_number,notnull" json:"lineNumber"`
	LedgerAccountID int64           `bun:"ledger_account_id,notnull" json:"ledgerAccountId"`
	Amount          decimal.Decimal `bun:"amount,type:numeric(36,18),notnull" json:"amount"`
	Side            PostingSide     `bun:"side,notnull" json:"side"`
	CreatedAt       time.Time       `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	JournalEntry  *JournalEntry  `bun:"rel:belongs-to,join:journal_entry_id=id,on_delete:restrict" json:"journalEntry,omitempty"`
	LedgerAccount *LedgerAccount `bun:"rel:belongs-to,join:ledger_account_id=id,on_delete:restrict" json:"ledgerAccount,omitempty"`
}

// PositionMovementKind says what a movement is, which decides who may write it
// and whether it has a journal entry.
type PositionMovementKind string

const (
	// PositionMovementTrade is a buy or a sale, written with its cash entry.
	PositionMovementTrade PositionMovementKind = "trade"
	// PositionMovementVest is units received from a grant, written with the
	// entry that records the compensation.
	PositionMovementVest PositionMovementKind = "vest"
	// PositionMovementSplit is a change of units with no cash effect.
	PositionMovementSplit PositionMovementKind = "split"
	// PositionMovementTransfer moves units between two of the owner's financial
	// accounts.
	PositionMovementTransfer PositionMovementKind = "transfer"
	// PositionMovementAdjustment is a correction of units that has no ledger
	// effect. It is not the way to record an opening balance.
	PositionMovementAdjustment PositionMovementKind = "adjustment"
)

// ValidPositionMovementKind returns true if the given kind is valid.
func ValidPositionMovementKind(kind PositionMovementKind) bool {
	switch kind {
	case PositionMovementTrade, PositionMovementVest, PositionMovementSplit, PositionMovementTransfer, PositionMovementAdjustment:
		return true
	default:
		return false
	}
}

// HasJournalEntry returns true for the kinds that always belong to a journal
// entry, which are the ones with a monetary effect.
func (k PositionMovementKind) HasJournalEntry() bool {
	return k == PositionMovementTrade || k == PositionMovementVest
}

// PositionMovement is the units subledger for investments. UnitsDelta and
// CostBasisAmount are signed and share a sign; a buy is normally positive and a
// sale negative. A movement of a monetary kind references its journal entry,
// and any other one carries an idempotency key of its own. A movement is never
// changed or deleted: a correction is another movement that references it
// through ReversalOfMovementID.
type PositionMovement struct {
	bun.BaseModel `bun:"table:asset.position_movement,alias:pm"`

	ID                 int64                `bun:"id,pk,autoincrement" json:"id"`
	JournalEntryID     *int64               `bun:"journal_entry_id" json:"journalEntryId,omitempty"`
	Kind               PositionMovementKind `bun:"kind,notnull" json:"kind"`
	FinancialAccountID int64                `bun:"financial_account_id,notnull" json:"financialAccountId"`
	AssetID            int64                `bun:"asset_id,notnull" json:"assetId"`
	UnitsDelta         decimal.Decimal      `bun:"units_delta,type:numeric(36,18),notnull" json:"unitsDelta"`
	// CostBasisAmount is the signed change to the position's total cost basis,
	// in CostBasisCurrencyID. The two fields are either both set or both nil.
	// Nil is not zero: a decrease removes the average cost, worked out when
	// read (ADR 0009), an increase makes the basis unknown, and a split keeps it.
	CostBasisAmount     *decimal.Decimal `bun:"cost_basis_amount,type:numeric(36,18)" json:"costBasisAmount,omitempty"`
	CostBasisCurrencyID *int64           `bun:"cost_basis_currency_id" json:"costBasisCurrencyId,omitempty"`
	OccurredAt          time.Time        `bun:"occurred_at,nullzero,notnull" json:"occurredAt"`
	// IdempotencyKey and LineNumber identify a movement without a journal
	// entry, which the movements of one command share the key of. A movement
	// with an entry has neither and is identified through the entry.
	IdempotencyKey *string `bun:"idempotency_key" json:"idempotencyKey,omitempty"`
	// LineNumber together with IdempotencyKey identify a movement without a jornal entry.
	LineNumber int `bun:"line_number,nullzero" json:"lineNumber,omitempty"`
	// ReversalOfMovementID is the movement this one compensates, if any.
	ReversalOfMovementID *int64            `bun:"reversal_of_movement_id" json:"reversalOfMovementId,omitempty"`
	Description          *string           `bun:"description" json:"description,omitempty"`
	Metadata             map[string]string `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	// ExternalID is the identifier of the source record of the movement. A trade
	// or vest carries its journal entry's external ID, copied when the movement is
	// written, so units trace to the same broker or payroll record as the cash. A
	// split, transfer, or adjustment has no entry, and carries the reference of
	// its own source, such as a corporate action.
	ExternalID *string   `bun:"external_id" json:"externalId,omitempty"`
	CreatedAt  time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	JournalEntry      *JournalEntry     `bun:"rel:belongs-to,join:journal_entry_id=id,on_delete:restrict" json:"journalEntry,omitempty"`
	FinancialAccount  *FinancialAccount `bun:"rel:belongs-to,join:financial_account_id=id,on_delete:restrict" json:"financialAccount,omitempty"`
	Asset             *Asset            `bun:"rel:belongs-to,join:asset_id=id,on_delete:restrict" json:"asset,omitempty"`
	CostBasisCurrency *Currency         `bun:"rel:belongs-to,join:cost_basis_currency_id=id,on_delete:restrict" json:"costBasisCurrency,omitempty"`
}

func FindFinancialAccountByIDQuery(db bun.IDB, id int64) *bun.SelectQuery {
	return db.NewSelect().Model((*FinancialAccount)(nil)).Where("fa.id = ?", id)
}

func FindLedgerAccountByIDQuery(db bun.IDB, id int64) *bun.SelectQuery {
	return db.NewSelect().Model((*LedgerAccount)(nil)).Where("la.id = ?", id)
}

func FindJournalEntryByIDQuery(db bun.IDB, id int64) *bun.SelectQuery {
	return db.NewSelect().Model((*JournalEntry)(nil)).Where("je.id = ?", id)
}

func FindJournalEntryByIDWithRelationshipsQuery(db bun.IDB, id int64) *bun.SelectQuery {
	return db.NewSelect().Model((*JournalEntry)(nil)).
		Relation("Counterparty").
		Relation("Postings").Relation("Postings.LedgerAccount").
		Relation("PositionMovements").
		Where("je.id = ?", id)
}

func FindPostingsByLedgerAccountIDQuery(db *bun.DB, ledgerAccountID int64) *bun.SelectQuery {
	return db.NewSelect().Model((*Posting)(nil)).
		Where("p.ledger_account_id = ?", ledgerAccountID).
		OrderExpr("p.journal_entry_id ASC, p.line_number ASC")
}

func FindPostingsByJournalEntryIDQuery(db *bun.DB, journalEntryID int64) *bun.SelectQuery {
	return db.NewSelect().Model((*Posting)(nil)).Where("p.journal_entry_id = ?", journalEntryID).OrderExpr("p.line_number ASC")
}

func FindPositionMovementsByAssetAndFinancialAccountQuery(db *bun.DB, assetID, financialAccountID int64) *bun.SelectQuery {
	return db.NewSelect().Model((*PositionMovement)(nil)).
		Where("pm.asset_id = ?", assetID).
		Where("pm.financial_account_id = ?", financialAccountID).
		OrderExpr("pm.occurred_at ASC, pm.id ASC")
}
