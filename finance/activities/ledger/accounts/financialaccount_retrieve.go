package accounts

import (
	"context"
	"database/sql"
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

func (a *Activities) RetrieveFinancialAccount(ctx context.Context, params lt.RetrieveFinancialAccountParams) (*lt.RetrieveFinancialAccountResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}

	account, err := loadFinancialAccount(ctx, db, params.ID)
	if err != nil {
		return nil, err
	}
	return &lt.RetrieveFinancialAccountResult{FinancialAccount: account}, nil
}

// loadFinancialAccount retrieves a FinancialAccount by ID or returns
// errors.ErrNotFound. db accepts both *bun.DB and bun.Tx.
func loadFinancialAccount(ctx context.Context, db bun.IDB, id int64) (*model.FinancialAccount, error) {
	account := &model.FinancialAccount{}
	if err := db.NewSelect().Model(account).Where("id = ?", id).Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return nil, internalerrors.ErrNotFound{ResourceType: "FinancialAccount", ID: id}
		}
		return nil, fmt.Errorf("load financial account: %w", err)
	}
	return account, nil
}
