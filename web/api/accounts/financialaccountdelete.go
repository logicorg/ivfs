package accounts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerDeleteFinancialAccount(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-financial-account",
		Method:      http.MethodDelete,
		Path:        "/api/financial-accounts/{id}",
		Summary:     "Delete a financial account with its ledger accounts, entries, and movements",
		Description: "An entry or a movement that also involves other financial accounts is kept, with this account's lines " +
			"moved to the untracked accounts. Deleting an account that does not exist changes nothing. A deletion still " +
			"running after a while is answered with 202 Accepted, and deleting again waits on it.",
		Tags:      tags,
		Errors:    []int{http.StatusConflict, http.StatusUnprocessableEntity},
		Responses: gateway.AccountDeletionResponses(api),
	}, func(ctx context.Context, in *DeleteAccountInput) (*gateway.AccountDeletionOutput, error) {
		return gateway.DeleteAccount(ctx, c, workflows.DeleteFinancialAccountWorkflowID(in.ID), ledgeraccountswf.DeleteFinancialAccount,
			lt.DeleteFinancialAccountInput{FinancialAccountID: in.ID, ReplacementID: gateway.Replacement(in.ReplacementID)},
			func(result *lt.DeleteAccountResult) []lt.NegativeHolding { return result.Warnings })
	})
}
