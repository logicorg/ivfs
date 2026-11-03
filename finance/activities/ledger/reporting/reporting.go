package reporting

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

func (a *Activities) FinancialAccountBalances(ctx context.Context, params lt.FinancialAccountBalancesParams) (*lt.FinancialAccountBalancesResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.FinancialAccountID <= 0 {
		return nil, internalerrors.ErrInvalidParameter{Description: "financial account ID must be positive"}
	}
	var account model.FinancialAccount
	if err := db.NewSelect().Model(&account).Where("id = ?", params.FinancialAccountID).Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return nil, internalerrors.ErrNotFound{ResourceType: "FinancialAccount", ID: params.FinancialAccountID}
		}
		return nil, fmt.Errorf("load financial account: %w", err)
	}
	var accounts []model.LedgerAccount
	if err := db.NewSelect().Model(&accounts).Where("financial_account_id = ?", params.FinancialAccountID).Order("id ASC").Scan(ctx); err != nil {
		return nil, fmt.Errorf("load ledger accounts: %w", err)
	}
	result := &lt.FinancialAccountBalancesResult{FinancialAccountID: params.FinancialAccountID, Balances: make([]lt.LedgerAccountBalanceResult, 0, len(accounts))}
	for _, account := range accounts {
		balance, err := a.LedgerAccountBalance(ctx, lt.LedgerAccountBalanceParams{LedgerAccountID: account.ID, At: params.At})
		if err != nil {
			return nil, err
		}
		result.Balances = append(result.Balances, *balance)
	}
	return result, nil
}

// PositionsAt returns the units held of each asset in each financial account,
// with their cost basis, counting the effective movements at or before At, or
// every one when At is zero.
func (a *Activities) PositionsAt(ctx context.Context, params lt.PositionsAtParams) (*lt.PositionsAtResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	filter := positionsAtParamsToFilter(params)

	var rows []positions.Units
	if err := positions.UnitsQuery(db, filter).Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("load positions: %w", err)
	}
	var movements []model.PositionMovement
	if err := positions.CostBasisMovementsQuery(db, filter).Scan(ctx, &movements); err != nil {
		return nil, fmt.Errorf("load position movements: %w", err)
	}
	bases := positions.ReplayCostBasis(movements)

	result := &lt.PositionsAtResult{Positions: make([]lt.PositionResult, 0, len(rows))}
	for _, item := range rows {
		position := unitsToPositionResult(item)
		// A position whose movements were all reversed has nothing replayed,
		// and its basis is a known zero.
		if basis, ok := bases[positions.Pair{FinancialAccountID: item.FinancialAccountID, AssetID: item.AssetID}]; ok {
			addCostBasis(&position, basis)
		}
		result.Positions = append(result.Positions, position)
	}
	return result, nil
}

func addCostBasis(position *lt.PositionResult, basis *positions.CostBasis) {
	currencyIDs := make([]int64, 0, len(basis.Amounts))
	for currencyID := range basis.Amounts {
		currencyIDs = append(currencyIDs, currencyID)
	}
	slices.Sort(currencyIDs)
	for _, currencyID := range currencyIDs {
		position.CostBasis = append(position.CostBasis, lt.CostBasisAmount{CurrencyID: currencyID, Amount: basis.Amounts[currencyID]})
	}
	position.UnknownCostBasis = basis.Unknown
	if currencyID, amount, ok := basis.Average(); ok {
		position.AverageCost = &lt.CostBasisAmount{CurrencyID: currencyID, Amount: amount}
	}
}

func positionsAtParamsToFilter(i lt.PositionsAtParams) positions.Filter {
	return positions.Filter{
		FinancialAccountID: i.FinancialAccountID,
		AssetID:            i.AssetID,
		At:                 i.At,
	}
}

func unitsToPositionResult(i positions.Units) lt.PositionResult {
	return lt.PositionResult{
		FinancialAccountID: i.FinancialAccountID,
		AssetID:            i.AssetID,
		Units:              i.Units,
		CostBasis:          []lt.CostBasisAmount{},
	}
}
