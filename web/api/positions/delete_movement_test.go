package positions

import (
	"net/http"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	portfoliowf "github.com/logicorg/ivfs/finance/workflows/ledger/portfolio"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestDeleteMovement checks that deleting runs DeletePositionMovement with the
// id, returns its warnings, turns a movement of a journal entry into a 422, and
// that an invalid id runs no workflow.
func TestDeleteMovement(t *testing.T) {
	remove := func(result lt.DeletePositionMovementResult, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: portfoliowf.DeletePositionMovement,
			Params:   lt.DeletePositionMovementInput{MovementID: 7},
			Result:   result,
			Err:      err,
		}
	}
	const ofEntry = "invalid parameter: the movement belongs to a journal entry, delete the journal entry instead"

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"delete a movement group": {
			Method: http.MethodDelete, Path: "/api/position-movements/7",
			Call:       remove(lt.DeletePositionMovementResult{}, nil),
			WantStatus: http.StatusOK, WantBody: `{}`,
		},
		"delete a movement group that leaves a holding negative": {
			Method: http.MethodDelete, Path: "/api/position-movements/7",
			Call: remove(lt.DeletePositionMovementResult{Warnings: []lt.NegativeHolding{{
				FinancialAccountID: 3, FinancialAccountName: "IBKR", AssetID: 12, AssetName: "ACME",
				At: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), Units: decimal.NewFromInt(-10),
			}}}, nil),
			WantStatus: http.StatusOK,
			WantBody: `{"warnings":[{"financialAccountId":3,"financialAccountName":"IBKR",` +
				`"assetId":12,"assetName":"ACME","at":"2026-03-03T00:00:00Z","units":"-10"}]}`,
		},
		"a movement of a journal entry": {
			Method: http.MethodDelete, Path: "/api/position-movements/7",
			Call:       remove(lt.DeletePositionMovementResult{}, temporal.NewApplicationError(ofEntry, "ErrInvalidParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, ofEntry),
		},
		"an id that is not one": {
			Method: http.MethodDelete, Path: "/api/position-movements/0", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
