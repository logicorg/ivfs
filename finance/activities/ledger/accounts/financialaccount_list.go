package accounts

import (
	"context"
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
)

// ListFinancialAccounts searches financial accounts by institution and/or a
// case-insensitive substring of the account's or the institution's name, the
// way a caller resolving something like "revolut eur" from natural language
// would. Each result carries its Institution and its LedgerAccounts (with
// their Currency) so the caller can disambiguate between candidates without
// a further round trip.
func (a *Activities) ListFinancialAccounts(ctx context.Context, params lt.ListFinancialAccountsParams) (*lt.ListFinancialAccountsResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}

	accounts := make([]model.FinancialAccount, 0)
	query := db.NewSelect().Model(&accounts).
		Relation("Institution").
		Relation("LedgerAccounts").
		Relation("LedgerAccounts.Currency").
		OrderExpr("fa.name ASC").
		Limit(lt.MaxFinancialAccountsResult)
	if params.InstitutionID != nil {
		query = query.Where("fa.institution_id = ?", *params.InstitutionID)
	}
	if params.NameContains != "" {
		query = query.Where("fa.name ILIKE ?", "%"+params.NameContains+"%")
	}
	if params.InstitutionNameContains != "" {
		query = query.Where(
			"EXISTS (SELECT 1 FROM institution.institution AS i WHERE i.id = fa.institution_id AND i.name ILIKE ?)",
			"%"+params.InstitutionNameContains+"%",
		)
	}
	if !params.IncludeClosed {
		query = query.Where("fa.closed_at IS NULL")
	}

	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list financial accounts: %w", err)
	}
	return &lt.ListFinancialAccountsResult{FinancialAccounts: accounts}, nil
}
