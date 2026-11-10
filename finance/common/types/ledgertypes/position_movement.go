package ledgertypes

import (
	"fmt"
	"strings"
	"time"

	"cuelang.org/go/cuego"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// SplitRatioMetadataKey is the metadata key that records the ratio of a split,
// such as "2:1". The movements carry the units, so the ratio is kept only for
// provenance and is never multiplied in.
const SplitRatioMetadataKey = "split_ratio"

// PositionMovementLeg is the change of one financial account in a position
// movement command.
type PositionMovementLeg struct {
	FinancialAccountID int64           `json:"financialAccountID"`
	UnitsDelta         decimal.Decimal `json:"unitsDelta"`
	// CostBasisAmount is the signed change to the account's cost basis in
	// CostBasisCurrencyID, and shares the sign of the units. The two are given
	// together or not at all. Without one, a leg that removes units removes
	// their average cost, worked out when read (ADR 0009), one that adds units
	// leaves the basis unknown, and a split leaves it as it is.
	CostBasisAmount     *decimal.Decimal `json:"costBasisAmount,omitempty"`
	CostBasisCurrencyID *int64           `json:"costBasisCurrencyID,omitempty"`
}

// RegisterPositionMovementInput records a change of units that has no cash
// effect, as one atomic write of all its legs. Trades are registered with
// RegisterInvestmentTrade, and vests with the command that also records their
// compensation.
//
// Kind decides what the legs may be:
//   - split: any non-zero units for each account, no cost basis effect, and the
//     ratio in the metadata under SplitRatioMetadataKey;
//   - transfer: exactly two legs on different accounts, with equal and opposite
//     units and cost basis, in one currency;
//   - adjustment: any legs, and a description.
type RegisterPositionMovementInput struct {
	IdempotencyKey string                     `json:"idempotencyKey" cue:"!=\"\""`
	Kind           model.PositionMovementKind `json:"kind"`
	AssetID        int64                      `json:"assetID" cue:">0"`
	OccurredAt     time.Time                  `json:"occurredAt"`
	Description    *string                    `json:"description,omitempty"`
	ExternalID     *string                    `json:"externalId,omitempty"`
	Metadata       map[string]string          `json:"metadata,omitempty"`
	Legs           []PositionMovementLeg      `json:"legs"`
}

// RegisterPositionMovementResult holds the ids of the movements written, in the
// order of the legs.
type RegisterPositionMovementResult struct {
	PositionMovementIDs []int64 `json:"positionMovementIds"`
	// Warnings lists the holdings that are negative after the movement, which
	// was written regardless.
	Warnings []NegativeHolding `json:"warnings,omitempty"`
}

// Validate checks the command without the database.
func (i *RegisterPositionMovementInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return internalerrors.ValidationError{Err: err}
	}
	if i.OccurredAt.IsZero() {
		return internalerrors.ErrMissingRequiredParameter{Parameters: []string{"OccurredAt"}}
	}

	switch i.Kind {
	case model.PositionMovementSplit, model.PositionMovementTransfer, model.PositionMovementAdjustment:
	case model.PositionMovementTrade, model.PositionMovementVest:
		return internalerrors.ErrInvalidParameter{Description: "a " + string(i.Kind) + " has a journal entry and is not registered as a position movement"}
	default:
		return internalerrors.ErrInvalidParameter{Description: "invalid position movement kind"}
	}

	if len(i.Legs) == 0 {
		return internalerrors.ErrInvalidParameter{Description: "at least one leg is required"}
	}
	seen := make(map[int64]struct{}, len(i.Legs))
	for _, leg := range i.Legs {
		if err := leg.validate(); err != nil {
			return err
		}
		if _, ok := seen[leg.FinancialAccountID]; ok {
			return internalerrors.ErrInvalidParameter{Description: "the legs must be on different financial accounts"}
		}
		seen[leg.FinancialAccountID] = struct{}{}
	}

	switch i.Kind {
	case model.PositionMovementSplit:
		return i.validateSplit()
	case model.PositionMovementTransfer:
		return i.validateTransfer()
	default:
		return i.validateAdjustment()
	}
}

func (l PositionMovementLeg) validate() error {
	if l.FinancialAccountID <= 0 {
		return internalerrors.ErrInvalidParameter{Description: "financial account ID must be positive"}
	}
	if l.UnitsDelta.IsZero() {
		return internalerrors.ErrInvalidParameter{Description: "units delta must not be zero"}
	}
	if (l.CostBasisAmount == nil) != (l.CostBasisCurrencyID == nil) {
		return internalerrors.ErrInvalidParameter{Description: "cost basis amount and currency must be given together"}
	}
	if l.CostBasisCurrencyID != nil && *l.CostBasisCurrencyID <= 0 {
		return internalerrors.ErrInvalidParameter{Description: "cost basis currency ID must be positive"}
	}
	if l.CostBasisAmount != nil && !l.CostBasisAmount.IsZero() && l.CostBasisAmount.Sign() != l.UnitsDelta.Sign() {
		return internalerrors.ErrInvalidParameter{Description: "cost basis must have the sign of the units"}
	}
	return nil
}

func (i *RegisterPositionMovementInput) validateSplit() error {
	for _, leg := range i.Legs {
		if leg.CostBasisAmount != nil && !leg.CostBasisAmount.IsZero() {
			return internalerrors.ErrInvalidParameter{Description: "a split does not change the cost basis"}
		}
	}
	if strings.TrimSpace(i.Metadata[SplitRatioMetadataKey]) == "" {
		return internalerrors.ErrMissingRequiredParameter{Parameters: []string{"Metadata." + SplitRatioMetadataKey}}
	}
	return nil
}

func (i *RegisterPositionMovementInput) validateTransfer() error {
	if len(i.Legs) != 2 {
		return internalerrors.ErrInvalidParameter{Description: "a transfer has exactly two legs"}
	}
	from, to := i.Legs[0], i.Legs[1]
	if !from.UnitsDelta.Add(to.UnitsDelta).IsZero() {
		return internalerrors.ErrInvalidParameter{Description: "the units of a transfer must be equal and opposite"}
	}
	if from.CostBasisAmount == nil && to.CostBasisAmount == nil {
		return nil
	}
	if from.CostBasisAmount == nil || to.CostBasisAmount == nil {
		return internalerrors.ErrInvalidParameter{Description: "the cost basis of a transfer must be given on both legs or on neither"}
	}
	if !from.CostBasisAmount.Add(*to.CostBasisAmount).IsZero() || *from.CostBasisCurrencyID != *to.CostBasisCurrencyID {
		return internalerrors.ErrInvalidParameter{Description: "the cost basis of a transfer must be equal and opposite, in one currency"}
	}
	return nil
}

func (i *RegisterPositionMovementInput) validateAdjustment() error {
	if i.Description == nil || strings.TrimSpace(*i.Description) == "" {
		return internalerrors.ErrMissingRequiredParameter{Parameters: []string{"Description"}}
	}
	return nil
}

// FinancialAccountIDs returns the financial account of each leg, in leg order.
func (i *RegisterPositionMovementInput) FinancialAccountIDs() []int64 {
	ids := make([]int64, len(i.Legs))
	for n, leg := range i.Legs {
		ids[n] = leg.FinancialAccountID
	}
	return ids
}

// ToMovements returns one movement per leg. They share the key of the command
// and are told apart by their line number, which starts at 1.
func (i *RegisterPositionMovementInput) ToMovements() []model.PositionMovement {
	movements := make([]model.PositionMovement, len(i.Legs))
	for n, leg := range i.Legs {
		key := i.IdempotencyKey
		movements[n] = model.PositionMovement{
			Kind:                i.Kind,
			FinancialAccountID:  leg.FinancialAccountID,
			AssetID:             i.AssetID,
			UnitsDelta:          leg.UnitsDelta,
			CostBasisAmount:     leg.CostBasisAmount,
			CostBasisCurrencyID: leg.CostBasisCurrencyID,
			OccurredAt:          i.OccurredAt,
			IdempotencyKey:      &key,
			LineNumber:          n + 1,
			Description:         i.Description,
			Metadata:            i.Metadata,
			ExternalID:          i.ExternalID,
		}
	}
	return movements
}

const (
	// DefaultPositionMovementsLimit is the page size when Limit is not set.
	DefaultPositionMovementsLimit = 100
	// MaxPositionMovementsLimit is the largest page a caller can ask for.
	MaxPositionMovementsLimit = 500
)

// PositionMovementCursor is where a page of the history ends, and the next one
// starts after it. It is the position of the last movement returned in the order
// of the history, which is by date, then by when it was recorded, then by id.
type PositionMovementCursor struct {
	OccurredAt time.Time `json:"occurredAt"`
	CreatedAt  time.Time `json:"createdAt"`
	ID         int64     `json:"id"`
}

// ListPositionMovementsParams selects the history of position movements. It
// lists the movements that count towards holdings, which are those without a
// journal entry and those of a posted one, so the sum of the units up to a date
// is what PositionAt reports for it. A zero filter is not applied.
//
// The date range is half-open: From is included and To is not, so consecutive
// ranges neither overlap nor leave a gap.
type ListPositionMovementsParams struct {
	FinancialAccountID int64                        `json:"financialAccountId" cue:">=0"`
	AssetID            int64                        `json:"assetId" cue:">=0"`
	From               time.Time                    `json:"from"`
	To                 time.Time                    `json:"to"`
	Kinds              []model.PositionMovementKind `json:"kinds,omitempty"`
	// Limit is the size of the page, DefaultPositionMovementsLimit when zero.
	Limit int `json:"limit" cue:">=0"`
	// After is the Next of the previous page, and is nil for the first one.
	After *PositionMovementCursor `json:"after,omitempty"`
}

// ListPositionMovementsResult is one page of the history, oldest first. Next is
// nil on the last page.
type ListPositionMovementsResult struct {
	Movements []model.PositionMovement `json:"movements"`
	Next      *PositionMovementCursor  `json:"next,omitempty"`
}

// PageSize returns the number of movements to return in a page.
func (p *ListPositionMovementsParams) PageSize() int {
	if p.Limit == 0 {
		return DefaultPositionMovementsLimit
	}
	return p.Limit
}

// Validate checks the parameters without the database.
func (p *ListPositionMovementsParams) Validate() error {
	if err := cuego.Validate(p); err != nil {
		return internalerrors.ValidationError{Err: err}
	}
	if p.Limit > MaxPositionMovementsLimit {
		return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("limit must not be more than %d", MaxPositionMovementsLimit)}
	}
	if !p.From.IsZero() && !p.To.IsZero() && !p.To.After(p.From) {
		return internalerrors.ErrInvalidParameter{Description: "the end of the date range must be after its start"}
	}
	for _, kind := range p.Kinds {
		if !model.ValidPositionMovementKind(kind) {
			return internalerrors.ErrInvalidParameter{Description: "invalid position movement kind"}
		}
	}
	if p.After != nil && (p.After.ID <= 0 || p.After.OccurredAt.IsZero() || p.After.CreatedAt.IsZero()) {
		return internalerrors.ErrInvalidParameter{Description: "the cursor needs the date, the creation time, and the id of a movement"}
	}
	return nil
}
