package ledgertypes

import (
	"cuelang.org/go/cuego"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// RegisterVestInput records the units received from a vest and the journal entry
// that books the compensation as one atomic operation. The caller supplies the
// postings, so the command does not know how the compensation is taxed or
// settled: income, units withheld for tax, and tax paid in cash are all just
// postings of the entry. Sell-to-cover is a separate trade, and an ESPP purchase
// is a trade too, since it has cash on the other side.
type RegisterVestInput struct {
	Entry              PostJournalEntryInput `json:"entry" cue:"{idempotencyKey: !=\"\"}"`
	FinancialAccountID int64                 `json:"financialAccountID" cue:">0"`
	AssetID            int64                 `json:"assetID" cue:">0"`
	// Units is the number of units delivered to the account, net of any that the
	// employer withheld for tax.
	Units decimal.Decimal `json:"units" cue:"!=\"0\""`
	// CostBasisAmount is the fair market value of the delivered units on the
	// vesting date, in CostBasisCurrencyID.
	CostBasisAmount     decimal.Decimal `json:"costBasisAmount" cue:"!=\"0\""`
	CostBasisCurrencyID int64           `json:"costBasisCurrencyID" cue:">0"`
}

// RegisterVestResult holds the entry and the movement that were written.
type RegisterVestResult struct {
	JournalEntryID     int64 `json:"journalEntryId"`
	PositionMovementID int64 `json:"positionMovementId"`
}

// Validate checks the vest without the database. The units and the cost basis
// are compared as strings because a decimal is encoded as one, so CUE only
// rejects zero and the sign is checked here, together with the rule that the
// entry has to be valid on its own.
func (i *RegisterVestInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errors.ValidationError{Err: err}
	}

	if err := i.Entry.Validate(); err != nil {
		return err
	}

	if !i.Units.IsPositive() {
		return errors.ErrInvalidParameter{Description: "the units of a vest must be positive"}
	}
	if !i.CostBasisAmount.IsPositive() {
		return errors.ErrInvalidParameter{Description: "the cost basis of a vest must be positive"}
	}

	return nil
}
