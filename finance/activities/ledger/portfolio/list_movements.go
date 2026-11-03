package portfolio

import (
	"context"
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// ListPositionMovements returns one page of the history of position movements,
// oldest first, over the movements that count towards holdings. Movements that
// happened at the same instant are ordered by id, the order the holdings check
// uses, so a page boundary can fall between them without skipping or repeating
// any.
func (a *Activities) ListPositionMovements(ctx context.Context, params lt.ListPositionMovementsParams) (*lt.ListPositionMovementsResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}

	query := positions.EffectiveMovementsQuery(db).OrderExpr("pm.occurred_at, pm.id")
	if params.FinancialAccountID > 0 {
		query = query.Where("pm.financial_account_id = ?", params.FinancialAccountID)
	}
	if params.AssetID > 0 {
		query = query.Where("pm.asset_id = ?", params.AssetID)
	}
	if !params.From.IsZero() {
		query = query.Where("pm.occurred_at >= ?", params.From)
	}
	if !params.To.IsZero() {
		query = query.Where("pm.occurred_at < ?", params.To)
	}
	if len(params.Kinds) > 0 {
		query = query.Where("pm.kind IN (?)", bun.List(params.Kinds))
	}
	if params.After != nil {
		query = query.Where("(pm.occurred_at, pm.id) > (?, ?)", params.After.OccurredAt, params.After.ID)
	}

	// One more than the page is read to know whether there is a next page.
	pageSize := params.PageSize()
	var movements []model.PositionMovement
	if err := query.Limit(pageSize+1).Scan(ctx, &movements); err != nil {
		return nil, fmt.Errorf("list position movements: %w", err)
	}

	result := &lt.ListPositionMovementsResult{Movements: movements}
	if len(movements) > pageSize {
		result.Movements = movements[:pageSize]
		last := result.Movements[pageSize-1]
		result.Next = &lt.PositionMovementCursor{OccurredAt: last.OccurredAt, ID: last.ID}
	}
	return result, nil
}
