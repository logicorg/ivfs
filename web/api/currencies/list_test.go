package currencies

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestList checks that the currencies are listed as the API's own type, and
// that none is an empty list rather than null.
func TestList(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"list the currencies": {
			Method: http.MethodGet, Path: "/api/currencies",
			Call: &apitest.WorkflowCall{
				Workflow: currencieswf.ListCurrencies, Params: ct.ListCurrenciesParams{},
				Result: ct.ListCurrenciesResult{Currencies: []*model.Currency{
					{ID: 1, Code: "EUR", Name: "Euro", Symbol: "€", Precision: 2},
					{ID: 2, Code: "JPY", Name: "Yen", Precision: 0},
				}},
			},
			WantStatus: http.StatusOK,
			WantBody: `{"currencies":[{"id":1,"code":"EUR","name":"Euro","symbol":"€","precision":2},` +
				`{"id":2,"code":"JPY","name":"Yen","precision":0}]}`,
		},
		"list no currencies": {
			Method: http.MethodGet, Path: "/api/currencies",
			Call: &apitest.WorkflowCall{
				Workflow: currencieswf.ListCurrencies, Params: ct.ListCurrenciesParams{},
				Result: ct.ListCurrenciesResult{},
			},
			WantStatus: http.StatusOK, WantBody: `{"currencies":[]}`,
		},
	})
}
