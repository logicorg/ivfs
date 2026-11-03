package ledgertypes

import (
	"time"

	"cuelang.org/go/cuego"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// RegisterInvestmentTradeInput records one buy or sell and its cash-side
// journal entry in one atomic operation.
type RegisterInvestmentTradeInput struct {
	Entry              PostJournalEntryInput `json:"entry" cue:"{idempotencyKey: !=\"\"}"`
	FinancialAccountID int64                 `json:"financialAccountID" cue:">0"`
	AssetID            int64                 `json:"assetID" cue:">0"`
	UnitsDelta         decimal.Decimal       `json:"unitsDelta" cue:"!=\"0\""`
	// CostBasisAmount is what the units cost, or minus what was removed from the
	// cost basis by a sale, in CostBasisCurrencyID. The two are given together or
	// not at all. A sale without one removes the average cost, worked out when
	// read (ADR 0009), and a buy without one leaves the basis unknown.
	CostBasisAmount     *decimal.Decimal `json:"costBasisAmount,omitempty"`
	CostBasisCurrencyID *int64           `json:"costBasisCurrencyID,omitempty" cue:">0"`
}

// Validate checks the trade. The units are compared as a string because a
// decimal is encoded as one, so a numeric bound would never match it. The rules
// that CUE cannot express are checked here: the entry has to be valid on its own,
// and the cost basis amount and currency are given together or not at all.
func (i *RegisterInvestmentTradeInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errors.ValidationError{Err: err}
	}

	if err := i.Entry.Validate(); err != nil {
		return err
	}

	if (i.CostBasisAmount == nil) != (i.CostBasisCurrencyID == nil) {
		return errors.ErrInvalidParameter{Description: "cost basis amount and currency must be given together"}
	}

	return nil
}

type RegisterInvestmentTradeResult struct {
	JournalEntryID     int64 `json:"journalEntryId"`
	PositionMovementID int64 `json:"positionMovementId"`
	// Warnings lists the holdings that are negative after the trade, which was
	// written regardless.
	Warnings []NegativeHolding `json:"warnings,omitempty"`
}

type PositionAtParams struct {
	FinancialAccountID int64     `json:"financialAccountId" cue:">0"`
	AssetID            int64     `json:"assetId" cue:">0"`
	At                 time.Time `json:"at"`
}

func (i *PositionAtParams) Validate() error {
	err := cuego.Validate(i)
	if err != nil {
		return errors.ValidationError{Err: err}
	}
	return nil
}

type PositionAtResult struct {
	FinancialAccountID int64
	AssetID            int64
	Units              decimal.Decimal
}
