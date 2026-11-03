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

func (a *Activities) RetrieveLedgerAccount(ctx context.Context, params lt.RetrieveLedgerAccountParams) (*lt.RetrieveLedgerAccountResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.ID <= 0 {
		return nil, internalerrors.ErrInvalidParameter{Description: "ledger account ID must be positive"}
	}

	account := &model.LedgerAccount{}
	err = model.FindLedgerAccountByIDQuery(db, params.ID).
		Relation("Currency").
		Relation("FinancialAccount").
		Relation("Parent").
		Scan(ctx, account)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, internalerrors.ErrNotFound{ResourceType: "LedgerAccount", ID: params.ID}
		}
		return nil, fmt.Errorf("retrieve ledger account: %w", err)
	}
	return &lt.RetrieveLedgerAccountResult{LedgerAccount: account}, nil
}

func (a *Activities) ListLedgerAccounts(ctx context.Context, params lt.ListLedgerAccountsParams) (*lt.ListLedgerAccountsResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}

	query := db.NewSelect().Model((*model.LedgerAccount)(nil)).
		Relation("Currency").
		OrderExpr("la.parent_id NULLS FIRST, la.id ASC")
	if params.FinancialAccountID != nil {
		query = query.Where("la.financial_account_id = ?", *params.FinancialAccountID)
	}
	if params.ParentID != nil {
		query = query.Where("la.parent_id = ?", *params.ParentID)
	}
	if params.CurrencyID != nil {
		query = query.Where("la.currency_id = ?", *params.CurrencyID)
	}
	if params.NameContains != "" {
		query = query.Where("la.name ILIKE ?", "%"+params.NameContains+"%")
	}
	if len(params.Kinds) > 0 {
		query = query.Where("la.kind IN (?)", bun.List(params.Kinds))
	}
	if !params.IncludeClosed {
		query = query.Where("la.closed_at IS NULL")
	}

	accounts := make([]model.LedgerAccount, 0)
	if err := query.Scan(ctx, &accounts); err != nil {
		return nil, fmt.Errorf("list ledger accounts: %w", err)
	}
	return &lt.ListLedgerAccountsResult{LedgerAccounts: accounts}, nil
}
