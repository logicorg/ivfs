package accounts

import (
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
	"net/http"
	"testing"
)

// TestPreviewLedgerAccountDeletion checks that the preview runs
// PreviewAccountDeletion on the ledger account and returns its preview, with
// amounts as strings.
func TestPreviewLedgerAccountDeletion(t *testing.T) {
	preview := func(result act.AccountDeletionPreview, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: ledgeraccountswf.PreviewAccountDeletion, Params: act.PreviewAccountDeletionInput{LedgerAccountID: ptr.Int64(7)},
			Result: result, Err: err,
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"preview it": {
			Method: http.MethodGet, Path: "/api/ledger-accounts/7/deletion-preview",
			Call: preview(act.AccountDeletionPreview{
				DeletedEntries: 2, KeptEntries: 1, KeptMovementGroups: 1,
				StrayAmounts: []act.StrayAmount{{CurrencyCode: "EUR", Amount: decimal.RequireFromString("100.50")}},
				StrayUnits:   []act.StrayUnits{{AssetID: 12, AssetName: "ACME", Units: decimal.NewFromInt(-5)}},
				Conflicts:    []string{"ledger account 7 has child accounts; delete them first"},
			}, nil),
			WantStatus: http.StatusOK,
			WantBody: `{"deletedEntries":2,"keptEntries":1,"deletedMovementGroups":0,"keptMovementGroups":1,` +
				`"strayAmounts":[{"currencyCode":"EUR","amount":"100.5"}],` +
				`"strayUnits":[{"assetId":12,"assetName":"ACME","units":"-5"}],` +
				`"conflicts":["ledger account 7 has child accounts; delete them first"]}`,
		},
		"an unknown id": {
			Method: http.MethodGet, Path: "/api/ledger-accounts/7/deletion-preview",
			Call:       preview(act.AccountDeletionPreview{}, temporal.NewApplicationError("resource LedgerAccount not found with ID: 7", "ErrNotFound")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource LedgerAccount not found with ID: 7"),
		},
		"id 0": {
			Method: http.MethodGet, Path: "/api/ledger-accounts/0/deletion-preview", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
