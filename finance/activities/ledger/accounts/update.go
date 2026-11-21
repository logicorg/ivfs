package accounts

import (
	"context"
	"fmt"
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
	"strings"
)

func (a *Activities) RenameLedgerAccount(ctx context.Context, input act.RenameLedgerAccountInput) (*act.LedgerAccountMutationResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	input.Name = strings.TrimSpace(input.Name)

	result := &act.LedgerAccountMutationResult{LedgerAccountID: input.LedgerAccountID}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		account, err := loadLedgerAccount(ctx, tx, input.LedgerAccountID)
		if err != nil {
			return err
		}
		account.Name = input.Name
		if _, err := tx.NewUpdate().Model(account).Column("name", "updated_at").WherePK().Exec(ctx); err != nil {
			if conflict := sqllib.AlreadyExists(err, "LedgerAccount"); conflict != nil {
				return conflict
			}
			return fmt.Errorf("rename ledger account: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
