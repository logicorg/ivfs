package accounts

import (
	"context"
	"fmt"
	"strings"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// UpdateFinancialAccountMetadata renames a FinancialAccount and/or updates
// its ExternalID. It never changes InstitutionID, currency, or linked ledger
// accounts, and it does not require the account to be open.
func (a *Activities) UpdateFinancialAccountMetadata(ctx context.Context, input lt.UpdateFinancialAccountMetadataInput) (*lt.FinancialAccountMutationResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}
	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		return nil, internalerrors.ErrInvalidParameter{Description: "name must not be blank"}
	}

	result := &lt.FinancialAccountMutationResult{FinancialAccountID: input.FinancialAccountID}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		account, err := loadFinancialAccount(ctx, tx, input.FinancialAccountID)
		if err != nil {
			return err
		}

		columns := make([]string, 0, 2)
		if input.Name != nil {
			account.Name = strings.TrimSpace(*input.Name)
			columns = append(columns, "name")
		}
		if input.ExternalID != nil {
			account.ExternalID = input.ExternalID
			columns = append(columns, "external_id")
		}
		if len(columns) == 0 {
			return nil
		}

		columns = append(columns, "updated_at")
		if _, err := tx.NewUpdate().Model(account).Column(columns...).WherePK().Exec(ctx); err != nil {
			if conflict := sqllib.AlreadyExists(err, "FinancialAccount"); conflict != nil {
				return conflict
			}
			return fmt.Errorf("update financial account: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
