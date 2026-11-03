package counterparties

import (
	"net/http"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestCreate checks that the body becomes CreateCounterparty's input, with or
// without an external id, that its refusal keeps its status, and that an
// empty name or external id runs no workflow.
func TestCreate(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"create a counterparty": {
			Method: http.MethodPost, Path: "/api/counterparties", Body: `{"name":"Tesco"}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.CreateCounterparty,
				Params:   lt.CreateCounterpartyInput{Name: "Tesco"},
				Result:   lt.CreateCounterpartyResult{CounterpartyID: 3},
			},
			WantStatus: http.StatusOK, WantBody: `{"id":3}`,
		},
		"create a counterparty with an external id": {
			Method: http.MethodPost, Path: "/api/counterparties", Body: `{"name":"Acme","externalId":"payroll-7"}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.CreateCounterparty,
				Params:   lt.CreateCounterpartyInput{Name: "Acme", ExternalID: ptr.NewPtr("payroll-7")},
				Result:   lt.CreateCounterpartyResult{CounterpartyID: 1},
			},
			WantStatus: http.StatusOK, WantBody: `{"id":1}`,
		},
		"create a counterparty with a blank name": {
			Method: http.MethodPost, Path: "/api/counterparties", Body: `{"name":"  "}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.CreateCounterparty,
				Params:   lt.CreateCounterpartyInput{Name: "  "},
				Err:      temporal.NewApplicationError("missing required parameters: Name", "ErrMissingRequiredParameter"),
			},
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "missing required parameters: Name"),
		},
		"create a counterparty with no name": {
			Method: http.MethodPost, Path: "/api/counterparties", Body: `{"name":""}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"create a counterparty with an empty external id": {
			Method: http.MethodPost, Path: "/api/counterparties", Body: `{"name":"Acme","externalId":""}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
