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

// TestUpdate checks that only the fields in the body are passed to
// UpdateCurrency, that a missing currency is a 404, and that an invalid id
// runs no workflow.
func TestUpdate(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"rename a currency": {
			Method: http.MethodPatch, Path: "/api/currencies/3", Body: `{"name":"Euro"}`,
			Call: &apitest.WorkflowCall{
				Workflow: currencieswf.UpdateCurrency,
				Params:   &ct.UpdateCurrencyParams{ID: 3, Name: ptr.NewPtr("Euro")},
			},
			WantStatus: http.StatusNoContent,
		},
		"change every field of a currency": {
			Method: http.MethodPatch, Path: "/api/currencies/3", Body: `{"code":"EUR","name":"Euro","symbol":"€"}`,
			Call: &apitest.WorkflowCall{
				Workflow: currencieswf.UpdateCurrency,
				Params:   &ct.UpdateCurrencyParams{ID: 3, Code: ptr.NewPtr("EUR"), Name: ptr.NewPtr("Euro"), Symbol: ptr.NewPtr("€")},
			},
			WantStatus: http.StatusNoContent,
		},
		"change a currency that does not exist": {
			Method: http.MethodPatch, Path: "/api/currencies/9", Body: `{"name":"Euro"}`,
			Call: &apitest.WorkflowCall{
				Workflow: currencieswf.UpdateCurrency,
				Params:   &ct.UpdateCurrencyParams{ID: 9, Name: ptr.NewPtr("Euro")},
				Err:      temporal.NewApplicationError("resource Currency 9 not found", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Currency 9 not found"),
		},
		"change a currency with an invalid id": {
			Method: http.MethodPatch, Path: "/api/currencies/0", Body: `{"name":"Euro"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
