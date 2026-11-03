package fx

import (
	"context"
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/posting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// RegisterFXConversion posts a foreign exchange conversion as one balanced
// journal entry, validating that every account involved is open and of the
// right kind and currency. Retrying with the same idempotency key returns
// the original entry without comparing the rest of the request.
func (a *Activities) RegisterFXConversion(ctx context.Context, input lt.RegisterFXConversionInput) (*lt.RegisterFXConversionResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	result := &lt.RegisterFXConversionResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		existing, err := posting.FindEntryByIdempotencyKeyTx(ctx, tx, input.IdempotencyKey)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.Metadata["type"] != lt.FXConversionEntryType {
				return internalerrors.ErrResourceConflict{ResourceType: "JournalEntry", Reason: "idempotency key belongs to an entry that is not an FX conversion"}
			}
			result.JournalEntryID = existing.ID
			return nil
		}

		accounts, err := loadAccounts(ctx, tx, input.AccountIDs())
		if err != nil {
			return err
		}
		if err := validateAccounts(input, accounts); err != nil {
			return err
		}

		result.JournalEntryID, err = posting.PostJournalEntryTx(ctx, tx, input.ToJournalEntry())
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func loadAccounts(ctx context.Context, tx bun.Tx, ids []int64) (map[int64]*model.LedgerAccount, error) {
	var loaded []model.LedgerAccount
	if err := tx.NewSelect().Model(&loaded).Where("la.id IN (?)", bun.List(ids)).Scan(ctx); err != nil {
		return nil, fmt.Errorf("load fx ledger accounts: %w", err)
	}

	accounts := make(map[int64]*model.LedgerAccount, len(loaded))
	for i := range loaded {
		accounts[loaded[i].ID] = &loaded[i]
	}
	return accounts, nil
}
