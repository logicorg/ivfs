package accounts

import (
	"net/http"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestPreviewLedgerAccountDeletion checks that the preview runs
// PreviewAccountDeletion on the ledger account and returns its preview, with
// amounts as strings.
func TestPreviewLedgerAccountDeletion(t *testing.T) {
	preview := func(result lt.AccountDeletionPreview, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: ledgeraccountswf.PreviewAccountDeletion, Params: lt.PreviewAccountDeletionInput{LedgerAccountID: ptr.Int64(7)},
			Result: result, Err: err,
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"preview it": {
			Method: http.MethodGet, Path: "/api/ledger-accounts/7/deletion-preview",
			Call: preview(lt.AccountDeletionPreview{
				DeletedEntries: 2, KeptEntries: 1, KeptMovementGroups: 1,
				StrayAmounts: []lt.StrayAmount{{Name: "Untracked accounts", Kind: model.LedgerAccountEquity, CurrencyCode: "EUR", Amount: decimal.RequireFromString("100.50")}},
				StrayUnits:   []lt.StrayUnits{{AssetID: 12, AssetName: "ACME", Units: decimal.NewFromInt(-5)}},
				Conflicts:    []string{"ledger account 7 has child accounts; delete them first"},
			}, nil),
			WantStatus: http.StatusOK,
			WantBody: `{"deletedEntries":2,"keptEntries":1,"deletedMovementGroups":0,"keptMovementGroups":1,` +
				`"strayAmounts":[{"name":"Untracked accounts","kind":"equity","currencyCode":"EUR","amount":"100.5"}],` +
				`"strayUnits":[{"assetId":12,"assetName":"ACME","units":"-5"}],` +
				`"conflicts":["ledger account 7 has child accounts; delete them first"]}`,
		},
		"an unknown id": {
			Method: http.MethodGet, Path: "/api/ledger-accounts/7/deletion-preview",
			Call:       preview(lt.AccountDeletionPreview{}, temporal.NewApplicationError("resource LedgerAccount not found with ID: 7", "ErrNotFound")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource LedgerAccount not found with ID: 7"),
		},
		"id 0": {
			Method: http.MethodGet, Path: "/api/ledger-accounts/0/deletion-preview", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
