package holdings

import (
	"net/http"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	portfoliowf "github.com/logicorg/ivfs/finance/workflows/ledger/portfolio"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestListMovements checks that each filter and the cursor reach
// ListPositionMovements, that the next page's cursor goes back as after, that
// a movement's basis and entry are listed only when it has them, and that a
// cursor the API did not return is refused.
func TestListMovements(t *testing.T) {
	day := time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC)
	recorded := time.Date(2026, 2, 3, 9, 30, 0, 123456000, time.UTC)
	buy := model.PositionMovement{
		ID: 20, JournalEntryID: ptr.Int64(5), Kind: model.PositionMovementTrade, FinancialAccountID: 3, AssetID: 7,
		UnitsDelta: decimal.RequireFromString("12.5"), CostBasisAmount: ptr.NewPtr(decimal.RequireFromString("1250.25")),
		CostBasisCurrencyID: ptr.Int64(1), OccurredAt: day, Description: ptr.NewPtr("Monthly buy"), ExternalID: ptr.NewPtr("ibkr-42"),
	}
	buyBody := `{"id":20,"kind":"trade","financialAccountId":3,"assetId":7,"unitsDelta":"12.5",` +
		`"costBasis":{"currencyId":1,"amount":"1250.25"},"occurredAt":"2026-01-31T10:00:00Z","journalEntryId":5,` +
		`"description":"Monthly buy","externalId":"ibkr-42"}`
	sale := model.PositionMovement{
		ID: 21, JournalEntryID: ptr.Int64(6), Kind: model.PositionMovementTrade, FinancialAccountID: 3, AssetID: 7,
		UnitsDelta: decimal.RequireFromString("-2.5"), OccurredAt: day.Add(time.Hour),
	}
	saleBody := `{"id":21,"kind":"trade","financialAccountId":3,"assetId":7,"unitsDelta":"-2.5",` +
		`"occurredAt":"2026-01-31T11:00:00Z","journalEntryId":6}`
	adjustment := model.PositionMovement{
		ID: 22, Kind: model.PositionMovementAdjustment, FinancialAccountID: 3, AssetID: 7,
		UnitsDelta: decimal.RequireFromString("1"), OccurredAt: day.Add(2 * time.Hour),
	}
	adjustmentBody := `{"id":22,"kind":"adjustment","financialAccountId":3,"assetId":7,"unitsDelta":"1",` +
		`"occurredAt":"2026-01-31T12:00:00Z"}`
	const backwards = "invalid parameter: the end of the date range must be after its start"

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"an account's trades of an asset, in pages": {
			Method: http.MethodGet, Path: "/api/holdings/movements?financialAccountId=3&assetId=7&kind=trade&limit=2",
			Call: &apitest.WorkflowCall{
				Workflow: portfoliowf.ListPositionMovements,
				Params: lt.ListPositionMovementsParams{
					FinancialAccountID: 3, AssetID: 7, Kinds: []model.PositionMovementKind{model.PositionMovementTrade}, Limit: 2,
				},
				Result: lt.ListPositionMovementsResult{
					Movements: []model.PositionMovement{buy, sale},
					Next:      &lt.PositionMovementCursor{OccurredAt: day.Add(time.Hour), CreatedAt: recorded, ID: 21},
				},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"movements":[` + buyBody + `,` + saleBody + `],"next":"2026-01-31T11:00:00Z_2026-02-03T09:30:00.123456Z_21"}`,
		},
		"the next page": {
			Method: http.MethodGet, Path: "/api/holdings/movements?after=2026-01-31T11:00:00Z_2026-02-03T09:30:00.123456Z_21",
			Call: &apitest.WorkflowCall{
				Workflow: portfoliowf.ListPositionMovements,
				Params: lt.ListPositionMovementsParams{
					After: &lt.PositionMovementCursor{OccurredAt: day.Add(time.Hour), CreatedAt: recorded, ID: 21},
				},
				Result: lt.ListPositionMovementsResult{Movements: []model.PositionMovement{adjustment}},
			},
			WantStatus: http.StatusOK, WantBody: `{"movements":[` + adjustmentBody + `]}`,
		},
		"every movement, in a date range and several kinds": {
			Method: http.MethodGet,
			Path:   "/api/holdings/movements?from=2026-01-01T00:00:00Z&to=2026-02-01T00:00:00Z&kind=split&kind=adjustment",
			Call: &apitest.WorkflowCall{
				Workflow: portfoliowf.ListPositionMovements,
				Params: lt.ListPositionMovementsParams{
					Kinds: []model.PositionMovementKind{model.PositionMovementSplit, model.PositionMovementAdjustment},
					From:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				},
				Result: lt.ListPositionMovementsResult{Movements: []model.PositionMovement{adjustment}},
			},
			WantStatus: http.StatusOK, WantBody: `{"movements":[` + adjustmentBody + `]}`,
		},
		"no movements": {
			Method: http.MethodGet, Path: "/api/holdings/movements",
			Call: &apitest.WorkflowCall{
				Workflow: portfoliowf.ListPositionMovements, Params: lt.ListPositionMovementsParams{},
				Result: lt.ListPositionMovementsResult{},
			},
			WantStatus: http.StatusOK, WantBody: `{"movements":[]}`,
		},
		"a date range that ends before it starts": {
			Method: http.MethodGet, Path: "/api/holdings/movements?to=2026-01-01T00:00:00Z&from=2026-02-01T00:00:00Z",
			Call: &apitest.WorkflowCall{
				Workflow: portfoliowf.ListPositionMovements,
				Params: lt.ListPositionMovementsParams{
					From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				Err: temporal.NewApplicationError(backwards, "ErrInvalidParameter"),
			},
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, backwards),
		},
		"a cursor the API did not return": {
			Method: http.MethodGet, Path: "/api/holdings/movements?after=yesterday", WantStatus: http.StatusUnprocessableEntity,
		},
		"a cursor without when the movement was recorded": {
			Method: http.MethodGet, Path: "/api/holdings/movements?after=2026-01-31T11:00:00Z_21", WantStatus: http.StatusUnprocessableEntity,
		},
		"a kind that is not one": {
			Method: http.MethodGet, Path: "/api/holdings/movements?kind=gift", WantStatus: http.StatusUnprocessableEntity,
		},
		"a page that is too large": {
			Method: http.MethodGet, Path: "/api/holdings/movements?limit=501", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
