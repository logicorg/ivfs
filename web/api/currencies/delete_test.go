package currencies

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestDelete checks that the id is passed to DeleteCurrency, that a currency
// still in use is a 409, and that an invalid id runs no workflow.
func TestDelete(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"delete a currency": {
			Method: http.MethodDelete, Path: "/api/currencies/3",
			Call: &apitest.WorkflowCall{
				Workflow: currencieswf.DeleteCurrency,
				Params:   &ct.DeleteCurrencyParams{ID: 3},
			},
			WantStatus: http.StatusNoContent,
		},
		"delete a currency still in use": {
			Method: http.MethodDelete, Path: "/api/currencies/3",
			Call: &apitest.WorkflowCall{
				Workflow: currencieswf.DeleteCurrency,
				Params:   &ct.DeleteCurrencyParams{ID: 3},
				Err:      temporal.NewApplicationError("resource Currency conflict: still referenced by ledger.account.currency_id", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource Currency conflict: still referenced by ledger.account.currency_id"),
		},
		"delete a currency with an invalid id": {
			Method: http.MethodDelete, Path: "/api/currencies/0",
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
