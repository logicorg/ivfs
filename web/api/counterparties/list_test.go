package counterparties

import (
	"net/http"
	"testing"

	cpt "github.com/logicorg/ivfs/finance/common/types/counterpartytypes"
	counterpartieswf "github.com/logicorg/ivfs/finance/workflows/ledger/counterparties"
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
				Workflow: counterpartieswf.ListCounterparties,
				Params:   cpt.ListCounterpartiesParams{},
				Result: cpt.ListCounterpartiesResult{Counterparties: []model.Counterparty{
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
				Workflow: counterpartieswf.ListCounterparties,
				Params:   cpt.ListCounterpartiesParams{NameContains: "tes"},
				Result:   cpt.ListCounterpartiesResult{},
			},
			WantStatus: http.StatusOK, WantBody: `{"counterparties":[]}`,
		},
	})
}
