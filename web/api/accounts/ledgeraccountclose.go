package accounts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerCloseLedgerAccount(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "close-ledger-account",
		Method:      http.MethodPost,
		Path:        "/api/ledger-accounts/{id}/close",
		Summary:     "Close a ledger account",
		Description: "An asset or liability ledger account can be closed only without a balance, and none can be closed with open accounts under it. Closing a closed account changes nothing.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *CloseInput) (*struct{}, error) {
		return nil, gateway.RunWorkflowWithoutResult(ctx, c, ledgeraccountswf.CloseLedgerAccount,
			lt.CloseLedgerAccountInput{LedgerAccountID: in.ID})
	})
}
