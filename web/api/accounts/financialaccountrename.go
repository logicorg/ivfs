package accounts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerRenameFinancialAccount(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "rename-financial-account",
		Method:      http.MethodPost,
		Path:        "/api/financial-accounts/{id}/rename",
		Summary:     "Rename a financial account",
		Description: "A financial account can be renamed even once closed, but not to the name of another account at the same institution.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *RenameInput) (*struct{}, error) {
		_, err := gateway.RunWorkflow[act.FinancialAccountMutationResult](ctx, c, ledgeraccountswf.UpdateFinancialAccountMetadata,
			act.UpdateFinancialAccountMetadataInput{FinancialAccountID: in.ID, Name: &in.Body.Name})
		return nil, err
	})
}
