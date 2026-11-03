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

func registerCloseFinancialAccount(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "close-financial-account",
		Method:      http.MethodPost,
		Path:        "/api/financial-accounts/{id}/close",
		Summary:     "Close a financial account",
		Description: "A financial account can be closed only once each of its ledger accounts is, and it holds no units of an asset. Closing a closed account changes nothing.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *CloseInput) (*struct{}, error) {
		return nil, gateway.RunWorkflowWithoutResult(ctx, c, ledgeraccountswf.CloseFinancialAccount, lt.CloseFinancialAccountInput{FinancialAccountID: in.ID})
	})
}
