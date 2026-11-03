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

// TestPreviewFinancialAccountDeletion checks that the preview runs PreviewAccountDeletion on the
// financial account and returns its preview.
func TestPreviewFinancialAccountDeletion(t *testing.T) {
	preview := func(result lt.AccountDeletionPreview, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: ledgeraccountswf.PreviewAccountDeletion, Params: lt.PreviewAccountDeletionInput{FinancialAccountID: ptr.Int64(5)},
			Result: result, Err: err,
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"preview it": {
			Method: http.MethodGet, Path: "/api/financial-accounts/5/deletion-preview",
			Call:       preview(lt.AccountDeletionPreview{DeletedEntries: 3, DeletedMovementGroups: 1}, nil),
			WantStatus: http.StatusOK,
			WantBody:   `{"deletedEntries":3,"keptEntries":0,"deletedMovementGroups":1,"keptMovementGroups":0}`,
		},
		"an unknown id": {
			Method: http.MethodGet, Path: "/api/financial-accounts/5/deletion-preview",
			Call:       preview(lt.AccountDeletionPreview{}, temporal.NewApplicationError("resource FinancialAccount not found with ID: 5", "ErrNotFound")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource FinancialAccount not found with ID: 5"),
		},
		"id 0": {
			Method: http.MethodGet, Path: "/api/financial-accounts/0/deletion-preview", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
