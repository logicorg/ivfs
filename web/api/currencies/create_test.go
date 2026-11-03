package currencies

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestCreate checks that the body is passed to InsertCurrency as it is, that
// a conflict is answered with the workflow's message, and that a body Huma
// rejects runs no workflow.
func TestCreate(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"create a currency": {
			Method: http.MethodPost, Path: "/api/currencies", Body: `{"code":"eur","name":"Euro","precision":2}`,
			Call: &apitest.WorkflowCall{
				Workflow: currencieswf.InsertCurrency,
				Params:   ct.InsertCurrencyParams{Code: "eur", Name: "Euro", Precision: ptr.NewPtr(2)},
				Result:   ct.InsertCurrencyResult{CurrencyID: 3},
			},
			WantStatus: http.StatusOK, WantBody: `{"currencyId":3}`,
		},
		"create a currency whose name another has": {
			Method: http.MethodPost, Path: "/api/currencies", Body: `{"code":"EUX","name":"Euro"}`,
			Call: &apitest.WorkflowCall{
				Workflow: currencieswf.InsertCurrency, Params: ct.InsertCurrencyParams{Code: "EUX", Name: "Euro"},
				Err: temporal.NewApplicationError("a Currency with this name already exists", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "a Currency with this name already exists"),
		},
		"create a currency without a name": {
			Method: http.MethodPost, Path: "/api/currencies", Body: `{"code":"EUR"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"create a currency with an unknown field": {
			Method: http.MethodPost, Path: "/api/currencies", Body: `{"code":"EUR","name":"Euro","country":"EU"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
