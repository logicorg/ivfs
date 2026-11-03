package accounts

import (
	"net/http"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestRenameFinancialAccount checks that renaming runs
// UpdateFinancialAccountMetadata with only the name set, so the external ID
// is kept, and that its refusals keep their statuses.
func TestRenameFinancialAccount(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"rename it": {
			Method: http.MethodPost, Path: "/api/financial-accounts/3/rename", Body: `{"name":"Joint account"}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.UpdateFinancialAccountMetadata,
				Params:   lt.UpdateFinancialAccountMetadataInput{FinancialAccountID: 3, Name: ptr.NewPtr("Joint account")},
				Result:   lt.FinancialAccountMutationResult{FinancialAccountID: 3},
			},
			WantStatus: http.StatusNoContent,
		},
		"a name another account at the institution has": {
			Method: http.MethodPost, Path: "/api/financial-accounts/3/rename", Body: `{"name":"Revolut"}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.UpdateFinancialAccountMetadata,
				Params:   lt.UpdateFinancialAccountMetadataInput{FinancialAccountID: 3, Name: ptr.NewPtr("Revolut")},
				Err:      temporal.NewApplicationError("a FinancialAccount with this name already exists", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "a FinancialAccount with this name already exists"),
		},
		"a blank name": {
			Method: http.MethodPost, Path: "/api/financial-accounts/3/rename", Body: `{"name":"  "}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.UpdateFinancialAccountMetadata,
				Params:   lt.UpdateFinancialAccountMetadataInput{FinancialAccountID: 3, Name: ptr.NewPtr("  ")},
				Err:      temporal.NewApplicationError("invalid parameter: name must not be blank", "ErrInvalidParameter"),
			},
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "invalid parameter: name must not be blank"),
		},
		"an unknown id": {
			Method: http.MethodPost, Path: "/api/financial-accounts/99/rename", Body: `{"name":"Revolut"}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.UpdateFinancialAccountMetadata,
				Params:   lt.UpdateFinancialAccountMetadataInput{FinancialAccountID: 99, Name: ptr.NewPtr("Revolut")},
				Err:      temporal.NewApplicationError("resource FinancialAccount not found with ID: 99", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource FinancialAccount not found with ID: 99"),
		},
		"an empty name": {
			Method: http.MethodPost, Path: "/api/financial-accounts/3/rename", Body: `{"name":""}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
