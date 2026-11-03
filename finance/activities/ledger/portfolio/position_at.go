package portfolio

import (
	"context"
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/shopspring/decimal"
)

// PositionAt returns the units of one asset held in one financial account at a
// point in time, or now when At is zero. Movements without a journal entry
// count, and movements of draft entries do not.
func (a *Activities) PositionAt(ctx context.Context, params lt.PositionAtParams) (*lt.PositionAtResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}

	var rows []positions.Units
	err = positions.UnitsQuery(db, positions.Filter{FinancialAccountID: params.FinancialAccountID, AssetID: params.AssetID, At: params.At}).Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("calculate position: %w", err)
	}

	units := decimal.Zero
	if len(rows) > 0 {
		units = rows[0].Units
	}
	return &lt.PositionAtResult{FinancialAccountID: params.FinancialAccountID, AssetID: params.AssetID, Units: units}, nil
}
