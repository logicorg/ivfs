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

func registerRenameLedgerAccount(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "rename-ledger-account",
		Method:      http.MethodPost,
		Path:        "/api/ledger-accounts/{id}/rename",
		Summary:     "Rename a ledger account",
		Description: "A ledger account can be renamed even once closed, but not to the name another ledger account in its currency has under the same financial account or parent.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *RenameInput) (*struct{}, error) {
		_, err := gateway.RunWorkflow[lt.LedgerAccountMutationResult](ctx, c, ledgeraccountswf.RenameLedgerAccount,
			lt.RenameLedgerAccountInput{LedgerAccountID: in.ID, Name: in.Body.Name})
		return nil, err
	})
}
