package counterparties

import (
	"net/http"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestList checks that the search reaches ListCounterparties and that each
// counterparty keeps its external id only when it has one.
func TestList(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"every counterparty": {
			Method: http.MethodGet, Path: "/api/counterparties",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.ListCounterparties,
				Params:   lt.ListCounterpartiesParams{},
				Result: lt.ListCounterpartiesResult{Counterparties: []model.Counterparty{
					{ID: 1, Name: "Acme", ExternalID: ptr.NewPtr("payroll-7")},
					{ID: 2, Name: "Tesco"},
				}},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"counterparties":[{"id":1,"name":"Acme","externalId":"payroll-7"},{"id":2,"name":"Tesco"}]}`,
		},
		"the counterparties matching a search": {
			Method: http.MethodGet, Path: "/api/counterparties?nameContains=tes",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.ListCounterparties,
				Params:   lt.ListCounterpartiesParams{NameContains: "tes"},
				Result:   lt.ListCounterpartiesResult{},
			},
			WantStatus: http.StatusOK, WantBody: `{"counterparties":[]}`,
		},
	})
}
