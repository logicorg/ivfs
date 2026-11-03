package journal

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	"github.com/logicorg/ivfs/finance/activities/ledger/internal/posting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// ReverseJournalEntry creates a new, balanced entry with every posting side
// inverted. Posted entries remain immutable and are never deleted or edited.
// When the units it gives back were sold since, the reversal is written and the
// holdings that are negative afterwards are returned as warnings.
func (a *Activities) ReverseJournalEntry(ctx context.Context, input lt.ReverseJournalEntryInput) (*lt.ReverseJournalEntryResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	// TODO: validate with Cuego
	if input.JournalEntryID <= 0 {
		return nil, internalerrors.ErrInvalidParameter{Description: "journal entry ID must be positive"}
	}
	if input.IdempotencyKey == "" {
		return nil, internalerrors.ErrMissingRequiredParameter{Parameters: []string{"IdempotencyKey"}}
	}

	result := &lt.ReverseJournalEntryResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// The movements are immutable once the entry is posted, so they can be
		// read before the locks that protect the holdings they change.
		var movements []model.PositionMovement
		if err := tx.NewSelect().Model(&movements).Where("journal_entry_id = ?", input.JournalEntryID).Order("id ASC").Scan(ctx); err != nil {
			return fmt.Errorf("load position movements: %w", err)
		}
		financialAccountIDs := make([]int64, 0, len(movements))
		for _, movement := range movements {
			financialAccountIDs = append(financialAccountIDs, movement.FinancialAccountID)
		}
		if _, err := positions.LockFinancialAccountsTx(ctx, tx, financialAccountIDs); err != nil {
			return err
		}

		existing, err := posting.FindEntryByIdempotencyKeyTx(ctx, tx, input.IdempotencyKey)
		if err != nil {
			return err
		}
		if existing != nil {
			result.JournalEntryID = existing.ID
			var written []model.PositionMovement
			if err := tx.NewSelect().Model(&written).Where("journal_entry_id = ?", existing.ID).Scan(ctx); err != nil {
				return fmt.Errorf("load position movements of the reversal: %w", err)
			}
			result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, positions.PairsOf(written))
			return err
		}

		var source model.JournalEntry
		if err := tx.NewSelect().Model(&source).Where("id = ?", input.JournalEntryID).Scan(ctx); err != nil {
			if err == sql.ErrNoRows {
				return internalerrors.ErrNotFound{ResourceType: "JournalEntry", ID: input.JournalEntryID}
			}
			return fmt.Errorf("load journal entry: %w", err)
		}
		if source.Status != model.JournalEntryPosted {
			return internalerrors.ErrInvalidParameter{Description: "only posted journal entries can be reversed"}
		}
		var priorReversal model.JournalEntry
		if err := tx.NewSelect().Model(&priorReversal).Where("reversal_of_entry_id = ?", source.ID).Limit(1).Scan(ctx); err == nil {
			return internalerrors.ErrResourceConflict{ResourceType: "JournalEntry", Reason: "entry has already been reversed"}
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("check existing reversal: %w", err)
		}
		var postings []model.Posting
		if err := tx.NewSelect().Model(&postings).Where("journal_entry_id = ?", source.ID).Order("line_number ASC").Scan(ctx); err != nil {
			return fmt.Errorf("load journal postings: %w", err)
		}
		description := "Reversal of journal entry " + fmt.Sprint(source.ID)
		reversal := lt.PostJournalEntryInput{IdempotencyKey: input.IdempotencyKey, ReversalOfEntryID: &source.ID, OccurredAt: time.Now().UTC(), Description: &description, CounterpartyID: source.CounterpartyID, Metadata: source.Metadata, Postings: make([]lt.PostingInput, len(postings))}
		for i, posting := range postings {
			reversal.Postings[i] = lt.PostingInput{LedgerAccountID: posting.LedgerAccountID, Amount: posting.Amount, Side: posting.Side.Opposite()}
		}
		inverses := make([]model.PositionMovement, 0, len(movements))
		pairs := make([]positions.Pair, 0, len(movements))
		for _, movement := range movements {
			var costBasis *decimal.Decimal
			if movement.CostBasisAmount != nil {
				value := movement.CostBasisAmount.Neg()
				costBasis = &value
			}
			reversed := movement.ID
			inverses = append(inverses, model.PositionMovement{
				Kind:                 movement.Kind,
				FinancialAccountID:   movement.FinancialAccountID,
				AssetID:              movement.AssetID,
				UnitsDelta:           movement.UnitsDelta.Neg(),
				CostBasisAmount:      costBasis,
				CostBasisCurrencyID:  movement.CostBasisCurrencyID,
				ReversalOfMovementID: &reversed,
			})
			pairs = append(pairs, positions.Pair{FinancialAccountID: movement.FinancialAccountID, AssetID: movement.AssetID})
		}

		entryID, err := posting.PostEntryWithMovementsTx(ctx, tx, reversal, inverses)
		if err != nil {
			return err
		}
		result.JournalEntryID = entryID
		result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, pairs)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
